package agent

import "context"

type progressReporterKey struct{}

type ProgressReporter func(delta string)

func withProgressReporter(ctx context.Context, fn ProgressReporter) context.Context {
	if fn == nil {
		return ctx
	}
	return context.WithValue(ctx, progressReporterKey{}, fn)
}

func reportProgress(ctx context.Context, delta string) {
	if delta == "" {
		return
	}
	fn, ok := ctx.Value(progressReporterKey{}).(ProgressReporter)
	if !ok || fn == nil {
		return
	}
	fn(delta)
}
