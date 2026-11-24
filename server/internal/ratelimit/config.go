package ratelimit

import (
	"errors"
	"time"
)

type Rule struct {
	PerMinute int
	Burst     int
}

func (r Rule) interval() time.Duration {
	return time.Minute / time.Duration(r.PerMinute)
}

type Config struct {
	IP          Rule
	Account     Rule
	SignIn      Rule
	Spend       Rule
	IPHeader    string
	ReadTimeout time.Duration
	IdleTimeout time.Duration
}

func DefaultConfig() Config {
	return Config{
		IP:          Rule{PerMinute: 60, Burst: 20},
		Account:     Rule{PerMinute: 30, Burst: 10},
		SignIn:      Rule{PerMinute: 5, Burst: 5},
		Spend:       Rule{PerMinute: 5, Burst: 5},
		ReadTimeout: 15 * time.Second,
		IdleTimeout: 60 * time.Second,
	}
}

func (c Config) Validate() error {
	for _, rule := range []Rule{c.IP, c.Account, c.SignIn, c.Spend} {
		if rule.PerMinute < 1 || rule.Burst < 1 {
			return errors.New("HD_RATE_*: every per-minute rate and burst must be at least 1")
		}
	}
	if c.ReadTimeout <= 0 || c.IdleTimeout <= 0 {
		return errors.New("HD_READ_TIMEOUT and HD_IDLE_TIMEOUT must be positive")
	}
	return nil
}
