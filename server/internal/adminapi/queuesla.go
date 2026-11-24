package adminapi

import (
	"context"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/liveops"
)

type queueDeadline struct {
	DueAt   time.Time `json:"due_at"`
	Overdue bool      `json:"overdue"`
}

func deadlineOf(opened time.Time, sla time.Duration, now time.Time) queueDeadline {
	due := opened.Add(sla).UTC()
	return queueDeadline{DueAt: due, Overdue: now.After(due)}
}

func (h *Handler) queueSla(ctx context.Context) (liveops.QueueSla, error) {
	_, raw, err := latestSnapshot(ctx, h.pool, liveops.Domain)
	if err != nil || raw == nil {
		return liveops.Values{}.QueueSla(), err
	}
	values, err := liveops.Parse(raw)
	if err != nil {
		return liveops.QueueSla{}, err
	}
	return values.QueueSla(), nil
}
