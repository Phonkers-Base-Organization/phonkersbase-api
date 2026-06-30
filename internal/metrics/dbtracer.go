package metrics

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// slowQueryThreshold is the duration above which a query is logged as a
// warning, so a Grafana latency alert can be traced back to the offending
// query via logs instead of just a bare "SELECT was slow" metric.
const slowQueryThreshold = 500 * time.Millisecond

// unnamedQuery is the db.query.name value for queries missing a "-- name:"
// leading comment.
const unnamedQuery = "unknown"

type dbTracerCtxKey struct{}

type dbTraceState struct {
	start     time.Time
	name      string
	operation string
}

// DBTracer is a pgx.QueryTracer that records query duration as a metric and
// logs queries that exceed slowQueryThreshold.
type DBTracer struct{}

func NewDBTracer() *DBTracer { return &DBTracer{} }

func (DBTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	return context.WithValue(ctx, dbTracerCtxKey{}, dbTraceState{
		start:     time.Now(),
		name:      queryName(data.SQL),
		operation: queryOperation(data.SQL),
	})
}

func (DBTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	state, ok := ctx.Value(dbTracerCtxKey{}).(dbTraceState)
	if !ok {
		return
	}
	duration := time.Since(state.start)
	RecordDBQuery(ctx, state.name, state.operation, duration, data.Err)

	if duration >= slowQueryThreshold {
		log.Warn().
			Str("query.name", state.name).
			Str("db.operation.name", state.operation).
			Dur("duration", duration).
			Err(data.Err).
			Msg("slow query")
	}
}

// queryName extracts the query's low-cardinality identifier from a leading
// "-- name: <name>" comment, e.g. "organisation.get_all.items". Repository
// queries are expected to carry this tag; queries without one report as
// unnamedQuery, which still groups by operation but loses per-call-site
// resolution in Grafana.
func queryName(sql string) string {
	for line := range strings.Lines(sql) {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if name, ok := strings.CutPrefix(line, "-- name:"); ok {
			return strings.TrimSpace(name)
		}
		if strings.HasPrefix(line, "--") {
			continue
		}
		break
	}
	return unnamedQuery
}

// queryOperation extracts the leading SQL keyword (SELECT, INSERT, ...) as a
// low-cardinality label, instead of the full query text. Leading comment
// lines (e.g. the "-- name: ..." tag) are skipped.
func queryOperation(sql string) string {
	for line := range strings.Lines(sql) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}
		if i := strings.IndexAny(line, " \n\t"); i > 0 {
			return strings.ToUpper(line[:i])
		}
		return strings.ToUpper(line)
	}
	return ""
}
