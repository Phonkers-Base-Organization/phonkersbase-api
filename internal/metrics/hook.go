package metrics

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/rs/zerolog"
)

// ErrorHook increments the app.errors counter for every Error and Fatal event.
type ErrorHook struct{}

func (ErrorHook) Run(_ *zerolog.Event, level zerolog.Level, _ string) {
	if level != zerolog.ErrorLevel && level != zerolog.FatalLevel {
		return
	}
	errorsTotal.Add(context.Background(), 1)
}

// CallerHook attaches the source location (file:line) to Error and Fatal events.
// Skip=3: runtime.Caller → hook.Run(0) → (*Event).msg(1) → (*Event).Msg(2) → user code(3).
type CallerHook struct{}

func (CallerHook) Run(e *zerolog.Event, level zerolog.Level, _ string) {
	if level != zerolog.ErrorLevel && level != zerolog.FatalLevel {
		return
	}
	_, file, line, ok := runtime.Caller(3)
	if !ok {
		return
	}
	// Trim to the path starting from "internal/" or "cmd/" so the caller field
	// is identical in local dev and Docker (where the root differs).
	short := file
	for _, anchor := range []string{"/internal/", "/cmd/"} {
		if i := strings.Index(file, anchor); i >= 0 {
			short = file[i+1:]
			break
		}
	}
	e.Str("caller", fmt.Sprintf("%s:%d", short, line))
}
