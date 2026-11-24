package cosmetics

import (
	"context"
	"errors"

	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
)

func Published(ctx context.Context, db snapshot.Querier) (*Catalog, error) {
	_, published, err := snapshot.Latest(ctx, db, Domain)
	if errors.Is(err, snapshot.ErrNone) {
		return Builtin(), nil
	}
	if err != nil {
		return nil, err
	}
	return Parse(published)
}
