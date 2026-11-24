package db

import (
	"bytes"
	"io"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/metrics"
)

type PoolGauges struct {
	pool *pgxpool.Pool
}

func Gauges(pool *pgxpool.Pool) PoolGauges {
	return PoolGauges{pool: pool}
}

func (g PoolGauges) WriteTo(w io.Writer) (int64, error) {
	stat := g.pool.Stat()
	var out bytes.Buffer
	metrics.WriteGauge(&out, "hd_db_connections_max", "Largest size of the PostgreSQL pool.", float64(stat.MaxConns()))
	metrics.WriteGauge(&out, "hd_db_connections_open", "PostgreSQL connections open in the pool.", float64(stat.TotalConns()))
	metrics.WriteGauge(&out, "hd_db_connections_acquired", "PostgreSQL connections in use.", float64(stat.AcquiredConns()))
	metrics.WriteGauge(&out, "hd_db_acquire_waits", "Acquires since start that waited for a connection because none was idle.", float64(stat.EmptyAcquireCount()))
	metrics.WriteGauge(&out, "hd_db_acquire_canceled", "Acquires since start canceled before a connection was free.", float64(stat.CanceledAcquireCount()))
	return out.WriteTo(w)
}
