package cluster

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrNoRoute     = errors.New("cluster: no route")
	ErrNodeGone    = errors.New("cluster: node gone")
	ErrNotOwner    = errors.New("cluster: route owned by another node")
	ErrInvalidNode = errors.New("cluster: invalid node")
)

type Config struct {
	NodeID     string
	Addr       string
	Capacity   int
	Heartbeat  time.Duration
	NodeTTL    time.Duration
	RouteTTL   time.Duration
	ScaleOutAt float64
	ScaleInAt  float64
	KeyPrefix  string
}

func DefaultConfig() Config {
	return Config{
		Capacity:   10000,
		Heartbeat:  2 * time.Second,
		NodeTTL:    6 * time.Second,
		RouteTTL:   2 * time.Hour,
		ScaleOutAt: 0.7,
		ScaleInAt:  0.3,
		KeyPrefix:  "hd:",
	}
}

func (c Config) Validate() error {
	var errs []error
	if strings.TrimSpace(c.NodeID) == "" {
		errs = append(errs, fmt.Errorf("%w: empty node id", ErrInvalidNode))
	}
	if c.Capacity <= 0 {
		errs = append(errs, errors.New("cluster: capacity must be positive"))
	}
	if c.Heartbeat <= 0 || c.NodeTTL <= c.Heartbeat {
		errs = append(errs, errors.New("cluster: node ttl must exceed the heartbeat"))
	}
	if c.RouteTTL <= 0 {
		errs = append(errs, errors.New("cluster: route ttl must be positive"))
	}
	if c.ScaleOutAt <= c.ScaleInAt || c.ScaleOutAt > 1 || c.ScaleInAt < 0 {
		errs = append(errs, errors.New("cluster: scale thresholds must satisfy 0 <= in < out <= 1"))
	}
	return errors.Join(errs...)
}

type Load struct {
	Rooms       int  `json:"rooms"`
	Connections int  `json:"connections"`
	Draining    bool `json:"draining"`
}

type Node struct {
	ID       string    `json:"id"`
	Addr     string    `json:"addr"`
	Capacity int       `json:"capacity"`
	Load     Load      `json:"load"`
	SeenAt   time.Time `json:"seenAt"`
}

func (n Node) Free() int {
	return max(0, n.Capacity-n.Load.Rooms)
}

type Summary struct {
	Nodes       int     `json:"nodes"`
	Rooms       int     `json:"rooms"`
	Connections int     `json:"connections"`
	Capacity    int     `json:"capacity"`
	Utilization float64 `json:"utilization"`
	Advice      string  `json:"advice"`
}

const (
	AdviceOut  = "out"
	AdviceIn   = "in"
	AdviceHold = "hold"
)

type Cluster struct {
	config Config
	client redis.UniversalClient
	logger *slog.Logger
	now    func() time.Time
	mu     sync.Mutex
	load   func() Load
}

func New(config Config, client redis.UniversalClient, logger *slog.Logger) (*Cluster, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Cluster{config: config, client: client, logger: logger, now: time.Now}, nil
}

func (c *Cluster) NodeID() string {
	return c.config.NodeID
}

func (c *Cluster) key(parts ...string) string {
	return c.config.KeyPrefix + strings.Join(parts, ":")
}

func (c *Cluster) nodesKey() string {
	return c.key("nodes")
}

func (c *Cluster) nodeKey(id string) string {
	return c.key("node", id)
}

func (c *Cluster) routeKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return c.key("route", hex.EncodeToString(sum[:16]))
}

func (c *Cluster) membersChannel() string {
	return c.key("members")
}

func (c *Cluster) Observe(load func() Load) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.load = load
}

func (c *Cluster) Self() Node {
	c.mu.Lock()
	source := c.load
	c.mu.Unlock()
	var load Load
	if source != nil {
		load = source()
	}
	return Node{ID: c.config.NodeID, Addr: c.config.Addr, Capacity: c.config.Capacity, Load: load, SeenAt: c.now()}
}

func (c *Cluster) Accepting() bool {
	self := c.Self()
	return !self.Load.Draining && self.Load.Rooms < self.Capacity
}

func (c *Cluster) Announce(ctx context.Context) error {
	self := c.Self()
	data, err := json.Marshal(self)
	if err != nil {
		return err
	}
	pipe := c.client.TxPipeline()
	pipe.Set(ctx, c.nodeKey(self.ID), data, c.config.NodeTTL)
	pipe.SAdd(ctx, c.nodesKey(), self.ID)
	_, err = pipe.Exec(ctx)
	return err
}

func (c *Cluster) Leave(ctx context.Context) error {
	pipe := c.client.TxPipeline()
	pipe.Del(ctx, c.nodeKey(c.config.NodeID))
	pipe.SRem(ctx, c.nodesKey(), c.config.NodeID)
	pipe.Publish(ctx, c.membersChannel(), c.config.NodeID)
	_, err := pipe.Exec(ctx)
	return err
}

func (c *Cluster) Run(ctx context.Context) error {
	if err := c.Announce(ctx); err != nil {
		return err
	}
	c.client.Publish(ctx, c.membersChannel(), c.config.NodeID)
	ticker := time.NewTicker(c.config.Heartbeat)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := c.Announce(ctx); err != nil && ctx.Err() == nil {
				c.logger.Warn("cluster heartbeat failed", "node", c.config.NodeID, "error", err)
			}
		case <-ctx.Done():
			leaveCtx, cancel := context.WithTimeout(context.Background(), c.config.Heartbeat)
			defer cancel()
			return c.Leave(leaveCtx)
		}
	}
}

func (c *Cluster) Nodes(ctx context.Context) ([]Node, error) {
	ids, err := c.client.SMembers(ctx, c.nodesKey()).Result()
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = c.nodeKey(id)
	}
	values, err := c.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}
	nodes := make([]Node, 0, len(ids))
	var gone []any
	for i, value := range values {
		text, ok := value.(string)
		if !ok {
			gone = append(gone, ids[i])
			continue
		}
		var node Node
		if err := json.Unmarshal([]byte(text), &node); err != nil {
			gone = append(gone, ids[i])
			continue
		}
		nodes = append(nodes, node)
	}
	if len(gone) > 0 {
		c.client.SRem(ctx, c.nodesKey(), gone...)
	}
	return nodes, nil
}

func (c *Cluster) Node(ctx context.Context, id string) (Node, error) {
	text, err := c.client.Get(ctx, c.nodeKey(id)).Result()
	if errors.Is(err, redis.Nil) {
		return Node{}, ErrNodeGone
	}
	if err != nil {
		return Node{}, err
	}
	var node Node
	if err := json.Unmarshal([]byte(text), &node); err != nil {
		return Node{}, err
	}
	return node, nil
}

func (c *Cluster) Least(ctx context.Context) (Node, bool, error) {
	nodes, err := c.Nodes(ctx)
	if err != nil {
		return Node{}, false, err
	}
	var best Node
	found := false
	for _, node := range nodes {
		if node.Load.Draining || node.Free() == 0 {
			continue
		}
		if !found || node.Free() > best.Free() || (node.Free() == best.Free() && node.ID < best.ID) {
			best, found = node, true
		}
	}
	return best, found, nil
}

func (c *Cluster) Summarize(ctx context.Context) (Summary, error) {
	nodes, err := c.Nodes(ctx)
	if err != nil {
		return Summary{}, err
	}
	s := Summary{Advice: AdviceHold}
	for _, node := range nodes {
		if node.Load.Draining {
			continue
		}
		s.Nodes++
		s.Rooms += node.Load.Rooms
		s.Connections += node.Load.Connections
		s.Capacity += node.Capacity
	}
	if s.Capacity > 0 {
		s.Utilization = float64(s.Rooms) / float64(s.Capacity)
	}
	switch {
	case s.Nodes == 0 || s.Utilization >= c.config.ScaleOutAt:
		s.Advice = AdviceOut
	case s.Nodes > 1 && s.Utilization <= c.config.ScaleInAt:
		s.Advice = AdviceIn
	}
	return s, nil
}

var claimScript = redis.NewScript(`
local owner = redis.call('GET', KEYS[1])
if owner == false or owner == ARGV[1] then
  redis.call('SET', KEYS[1], ARGV[1], 'PX', ARGV[2])
  return 1
end
return 0
`)

var releaseScript = redis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then
  return redis.call('DEL', KEYS[1])
end
return 0
`)

func (c *Cluster) Claim(ctx context.Context, token string) error {
	ok, err := claimScript.Run(ctx, c.client, []string{c.routeKey(token)}, c.config.NodeID, c.config.RouteTTL.Milliseconds()).Int()
	if err != nil {
		return err
	}
	if ok == 0 {
		return ErrNotOwner
	}
	return nil
}

func (c *Cluster) Release(ctx context.Context, token string) error {
	_, err := releaseScript.Run(ctx, c.client, []string{c.routeKey(token)}, c.config.NodeID).Int()
	return err
}

func (c *Cluster) Elsewhere(ctx context.Context, token string) (bool, error) {
	owner, err := c.client.Get(ctx, c.routeKey(token)).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	return err == nil && owner != c.config.NodeID, err
}

func (c *Cluster) Lookup(ctx context.Context, token string) (Node, error) {
	owner, err := c.client.Get(ctx, c.routeKey(token)).Result()
	if errors.Is(err, redis.Nil) {
		return Node{}, ErrNoRoute
	}
	if err != nil {
		return Node{}, err
	}
	if owner == c.config.NodeID {
		return c.Self(), nil
	}
	node, err := c.Node(ctx, owner)
	if errors.Is(err, ErrNodeGone) {
		c.client.Del(ctx, c.routeKey(token))
	}
	return node, err
}

func (c *Cluster) Subscribe(ctx context.Context, changed func(nodeID string)) error {
	sub := c.client.Subscribe(ctx, c.membersChannel())
	defer sub.Close()
	if _, err := sub.Receive(ctx); err != nil {
		return err
	}
	for {
		select {
		case message, ok := <-sub.Channel():
			if !ok {
				return nil
			}
			changed(message.Payload)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
