package metrics

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type dbTracerCtxKey struct{}

type dbTraceState struct {
	start     time.Time
	operation string
}

// DBTracer is a pgx.QueryTracer that records query duration as a metric.
type DBTracer struct{}

func NewDBTracer() *DBTracer { return &DBTracer{} }

func (DBTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	return context.WithValue(ctx, dbTracerCtxKey{}, dbTraceState{
		start:     time.Now(),
		operation: queryOperation(data.SQL),
	})
}

func (DBTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	state, ok := ctx.Value(dbTracerCtxKey{}).(dbTraceState)
	if !ok {
		return
	}
	RecordDBQuery(ctx, state.operation, time.Since(state.start), data.Err)
}

// queryOperation extracts the leading SQL keyword (SELECT, INSERT, ...) as a
// low-cardinality label, instead of the full query text.
func queryOperation(sql string) string {
	sql = strings.TrimSpace(sql)
	if i := strings.IndexAny(sql, " \n\t"); i > 0 {
		return strings.ToUpper(sql[:i])
	}
	return strings.ToUpper(sql)
}
