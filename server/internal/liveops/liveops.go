package liveops

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
)

const Domain = "liveops"

var ErrInvalid = errors.New("liveops: invalid liveops snapshot")

type Key struct {
	Name    string
	Unit    string
	Min     int
	Max     *int
	Default int
}

const (
	AnalyticsBatchSize    = "AnalyticsBatchSize"
	AnalyticsQueueLimit   = "AnalyticsQueueLimit"
	AnalyticsFlushSeconds = "AnalyticsFlushSeconds"
	AnalyticsMaxBatchSize = 400
	ReviewSlaHours        = "ReviewSlaHours"
	ReportSlaHours        = "ReportSlaHours"
	AppealSlaHours        = "AppealSlaHours"
	MaxSlaHours           = 720
	AppealTextMaxLength   = "AppealTextMaxLength"
	AppealTokenMinutes    = "AppealTokenMinutes"
)

var keys = []Key{
	{Name: "ContentGateMinAvatarsAnimated", Default: 30},
	{Name: "ContentGateMinAvatarsStatic", Default: 60},
	{Name: "ContentGateMinFrames", Default: 20},
	{Name: "ContentGateMinTables", Default: 15},
	{Name: "ContentGateMinEmoji", Default: 40},
	{Name: "ContentGateMinMedals", Default: 40},
	{Name: AnalyticsBatchSize, Unit: "events", Min: 1, Max: bound(AnalyticsMaxBatchSize), Default: 50},
	{Name: AnalyticsQueueLimit, Unit: "events", Min: 1, Max: bound(5000), Default: 500},
	{Name: AnalyticsFlushSeconds, Unit: "seconds", Min: 10, Max: bound(3600), Default: 60},
	{Name: ReviewSlaHours, Unit: "hours", Min: 1, Max: bound(MaxSlaHours), Default: 24},
	{Name: ReportSlaHours, Unit: "hours", Min: 1, Max: bound(MaxSlaHours), Default: 48},
	{Name: AppealSlaHours, Unit: "hours", Min: 1, Max: bound(MaxSlaHours), Default: 72},
	{Name: AppealTextMaxLength, Unit: "characters", Min: 50, Max: bound(2000), Default: 500},
	{Name: AppealTokenMinutes, Unit: "minutes", Min: 5, Max: bound(1440), Default: 30},
}

func bound(limit int) *int {
	return &limit
}

func Keys() []Key {
	return keys
}

type Values map[string]int

func Parse(raw []byte) (Values, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return nil, fmt.Errorf("%w: not a JSON object: %v", ErrInvalid, err)
	}
	values := Values{}
	for name, field := range fields {
		key, known := lookup(name)
		if !known {
			return nil, fmt.Errorf("%w: unknown key %s", ErrInvalid, name)
		}
		number, err := strconv.Atoi(string(field))
		if err != nil {
			return nil, fmt.Errorf("%w: %s is not an integer: %s", ErrInvalid, name, field)
		}
		if number < key.Min {
			return nil, fmt.Errorf("%w: %s %d below %d", ErrInvalid, name, number, key.Min)
		}
		if key.Max != nil && number > *key.Max {
			return nil, fmt.Errorf("%w: %s %d above %d", ErrInvalid, name, number, *key.Max)
		}
		values[name] = number
	}
	if analytics := values.Analytics(); analytics.QueueLimit < analytics.BatchSize {
		return nil, fmt.Errorf("%w: %s %d below %s %d", ErrInvalid, AnalyticsQueueLimit, analytics.QueueLimit, AnalyticsBatchSize, analytics.BatchSize)
	}
	return values, nil
}

type Analytics struct {
	BatchSize    int
	QueueLimit   int
	FlushSeconds int
}

func (v Values) Analytics() Analytics {
	return Analytics{BatchSize: v.value(AnalyticsBatchSize), QueueLimit: v.value(AnalyticsQueueLimit), FlushSeconds: v.value(AnalyticsFlushSeconds)}
}

func (v Values) AnalyticsConfig() *pb.AnalyticsConfig {
	analytics := v.Analytics()
	return &pb.AnalyticsConfig{BatchSize: uint32(analytics.BatchSize), QueueLimit: uint32(analytics.QueueLimit), FlushSeconds: uint32(analytics.FlushSeconds)}
}

type QueueSla struct {
	Review time.Duration
	Report time.Duration
	Appeal time.Duration
}

func (v Values) QueueSla() QueueSla {
	return QueueSla{Review: v.hours(ReviewSlaHours), Report: v.hours(ReportSlaHours), Appeal: v.hours(AppealSlaHours)}
}

type Appeals struct {
	TextMaxLength int
	TokenLifetime time.Duration
}

func (v Values) Appeals() Appeals {
	return Appeals{TextMaxLength: v.value(AppealTextMaxLength), TokenLifetime: time.Duration(v.value(AppealTokenMinutes)) * time.Minute}
}

func (v Values) hours(name string) time.Duration {
	return time.Duration(v.value(name)) * time.Hour
}

func (v Values) value(name string) int {
	if number, set := v[name]; set {
		return number
	}
	key, _ := lookup(name)
	return key.Default
}

func lookup(name string) (Key, bool) {
	for _, key := range keys {
		if key.Name == name {
			return key, true
		}
	}
	return Key{}, false
}
