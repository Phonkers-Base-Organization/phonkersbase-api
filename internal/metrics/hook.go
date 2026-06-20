package metrics

import (
	"context"

	"github.com/rs/zerolog"
)

// ErrorHook is a zerolog.Hook that increments the app.errors counter for
// every Error and Fatal level log event, keeping metrics consistent with
// logs without separate instrumentation at every error site.
type ErrorHook struct{}

func (ErrorHook) Run(_ *zerolog.Event, level zerolog.Level, _ string) {
	if level != zerolog.ErrorLevel && level != zerolog.FatalLevel {
		return
	}
	errorsTotal.Add(context.Background(), 1)
}
