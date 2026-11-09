package penalties

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

const Domain = "penalties"

var ErrInvalidConfig = errors.New("penalties: invalid config")

type Config struct {
	AfkReturnedWeight           int     `json:"afkReturnedWeight"`
	AfkWeight                   int     `json:"afkWeight"`
	LeaveReturnedWeight         int     `json:"leaveReturnedWeight"`
	LeaveWeight                 int     `json:"leaveWeight"`
	OffenseWindowDays           int     `json:"offenseWindowDays"`
	CleanMatchesToForgive       int     `json:"cleanMatchesToForgive"`
	WarningLevel                int     `json:"warningLevel"`
	QueueDelayLevel             int     `json:"queueDelayLevel"`
	BanLevel                    int     `json:"banLevel"`
	ReviewLevel                 int     `json:"reviewLevel"`
	QueueDelaySeconds           int     `json:"queueDelaySeconds"`
	QueueDelayMatches           int     `json:"queueDelayMatches"`
	BanHours                    int     `json:"banHours"`
	BanEscalationFactor         float64 `json:"banEscalationFactor"`
	OffenseRatingPenalty        int     `json:"offenseRatingPenalty"`
	PartnerLossReductionPercent int     `json:"partnerLossReductionPercent"`
	CasualOffenseWindowDays     int     `json:"casualOffenseWindowDays"`
	CasualCleanMatchesToForgive int     `json:"casualCleanMatchesToForgive"`
	CasualWarningLevel          int     `json:"casualWarningLevel"`
	CasualQueueDelayLevels      []int   `json:"casualQueueDelayLevels"`
	CasualQueueDelaySeconds     []int   `json:"casualQueueDelaySeconds"`
	CasualQueueDelayMatches     int     `json:"casualQueueDelayMatches"`
}

func DefaultConfig() Config {
	return Config{
		AfkReturnedWeight:           1,
		AfkWeight:                   2,
		LeaveReturnedWeight:         2,
		LeaveWeight:                 3,
		OffenseWindowDays:           30,
		CleanMatchesToForgive:       10,
		WarningLevel:                1,
		QueueDelayLevel:             3,
		BanLevel:                    6,
		ReviewLevel:                 12,
		QueueDelaySeconds:           300,
		QueueDelayMatches:           5,
		BanHours:                    24,
		BanEscalationFactor:         2,
		OffenseRatingPenalty:        10,
		PartnerLossReductionPercent: 50,
		CasualOffenseWindowDays:     14,
		CasualCleanMatchesToForgive: 5,
		CasualWarningLevel:          2,
		CasualQueueDelayLevels:      []int{4, 8},
		CasualQueueDelaySeconds:     []int{60, 180},
		CasualQueueDelayMatches:     3,
	}
}

func Validate(c Config) error {
	var errs []error
	fail := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf("%w: "+format, append([]any{ErrInvalidConfig}, args...)...))
	}
	for name, value := range map[string]int{
		"AfkReturnedWeight": c.AfkReturnedWeight, "AfkWeight": c.AfkWeight,
		"LeaveReturnedWeight": c.LeaveReturnedWeight, "LeaveWeight": c.LeaveWeight,
		"OffenseWindowDays": c.OffenseWindowDays, "CasualOffenseWindowDays": c.CasualOffenseWindowDays,
		"BanHours": c.BanHours,
	} {
		if value <= 0 {
			fail("%s must be positive", name)
		}
	}
	for name, value := range map[string]int{
		"CleanMatchesToForgive": c.CleanMatchesToForgive, "CasualCleanMatchesToForgive": c.CasualCleanMatchesToForgive,
		"QueueDelaySeconds": c.QueueDelaySeconds, "QueueDelayMatches": c.QueueDelayMatches,
		"CasualQueueDelayMatches": c.CasualQueueDelayMatches, "OffenseRatingPenalty": c.OffenseRatingPenalty,
	} {
		if value < 0 {
			fail("%s must not be negative", name)
		}
	}
	if c.WarningLevel > c.QueueDelayLevel || c.QueueDelayLevel > c.BanLevel || c.BanLevel > c.ReviewLevel {
		fail("ladder levels must not decrease")
	}
	if c.BanEscalationFactor < 1 {
		fail("BanEscalationFactor must be at least 1")
	}
	if c.PartnerLossReductionPercent < 0 || c.PartnerLossReductionPercent > 100 {
		fail("PartnerLossReductionPercent must be within 0 and 100")
	}
	if len(c.CasualQueueDelayLevels) != len(c.CasualQueueDelaySeconds) {
		fail("CasualQueueDelayLevels and CasualQueueDelaySeconds must have equal length")
	}
	for index, level := range c.CasualQueueDelayLevels {
		if index > 0 && level <= c.CasualQueueDelayLevels[index-1] {
			fail("CasualQueueDelayLevels must ascend")
		}
		if index < len(c.CasualQueueDelaySeconds) && c.CasualQueueDelaySeconds[index] > c.QueueDelaySeconds {
			fail("CasualQueueDelaySeconds must not exceed QueueDelaySeconds")
		}
	}
	return errors.Join(errs...)
}

func Parse(raw []byte) (Config, error) {
	config := DefaultConfig()
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	return config, Validate(config)
}
