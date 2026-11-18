package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/event"
)

func combineMonitors(monitors []*event.CommandMonitor) *event.CommandMonitor {
	active := make([]*event.CommandMonitor, 0, len(monitors))
	for _, monitor := range monitors {
		if monitor != nil {
			active = append(active, monitor)
		}
	}
	switch len(active) {
	case 0:
		return nil
	case 1:
		return active[0]
	}
	return &event.CommandMonitor{
		Started: fanOut(active, func(m *event.CommandMonitor) func(context.Context, *event.CommandStartedEvent) {
			return m.Started
		}),
		Succeeded: fanOut(active, func(m *event.CommandMonitor) func(context.Context, *event.CommandSucceededEvent) {
			return m.Succeeded
		}),
		Failed: fanOut(active, func(m *event.CommandMonitor) func(context.Context, *event.CommandFailedEvent) {
			return m.Failed
		}),
	}
}

func fanOut[E any](monitors []*event.CommandMonitor, pick func(*event.CommandMonitor) func(context.Context, E)) func(context.Context, E) {
	return func(ctx context.Context, e E) {
		for _, monitor := range monitors {
			if callback := pick(monitor); callback != nil {
				callback(ctx, e)
			}
		}
	}
}
