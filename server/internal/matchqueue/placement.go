package matchqueue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

var ErrHostGone = errors.New("matchqueue: no host listens for the placement")

type Dispatcher struct {
	client redis.UniversalClient
	prefix string
	node   string
}

func NewDispatcher(client redis.UniversalClient, prefix, node string) *Dispatcher {
	return &Dispatcher{client: client, prefix: prefix, node: node}
}

func (d *Dispatcher) placeChannel(node string) string {
	return d.prefix + "matchmaking:place:" + node
}

func (d *Dispatcher) foundChannel(node string) string {
	return d.prefix + "matchmaking:found:" + node
}

func (d *Dispatcher) Place(ctx context.Context, host string, placement Placement) error {
	raw, err := json.Marshal(placement)
	if err != nil {
		return err
	}
	receivers, err := d.client.Publish(ctx, d.placeChannel(host), raw).Result()
	if err != nil {
		return err
	}
	if receivers == 0 {
		return fmt.Errorf("%w: %s", ErrHostGone, host)
	}
	return nil
}

func (d *Dispatcher) Found(ctx context.Context, ticket Ticket) error {
	return d.client.Publish(ctx, d.foundChannel(ticket.Node), ticket.Session).Err()
}

func (d *Dispatcher) Listen(ctx context.Context, host func(context.Context, Placement), found func(session string)) error {
	sub := d.client.Subscribe(ctx, d.placeChannel(d.node), d.foundChannel(d.node))
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
			if message.Channel == d.foundChannel(d.node) {
				found(message.Payload)
				continue
			}
			var placement Placement
			if err := json.Unmarshal([]byte(message.Payload), &placement); err == nil {
				host(ctx, placement)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
