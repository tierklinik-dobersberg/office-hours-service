package watcher

import (
	"context"
	"log/slog"
	"time"

	"github.com/bufbuild/connect-go"
	eventsv1 "github.com/tierklinik-dobersberg/apis/gen/go/tkd/events/v1"
	"github.com/tierklinik-dobersberg/apis/gen/go/tkd/events/v1/eventsv1connect"
	office_hoursv1 "github.com/tierklinik-dobersberg/apis/gen/go/tkd/office_hours/v1"
	"github.com/tierklinik-dobersberg/office-hours-service/internal/resolver"
	"google.golang.org/protobuf/types/known/anypb"
)

type Watcher struct {
	resolver    *resolver.Resolver
	eventClient eventsv1connect.EventServiceClient

	trigger chan struct{}
}

func New(r *resolver.Resolver, eventClient eventsv1connect.EventServiceClient) *Watcher {
	w := &Watcher{
		resolver:    r,
		eventClient: eventClient,
		trigger:     make(chan struct{}),
	}

	return w
}

func (w *Watcher) Start(ctx context.Context) {
	var wasOpen *bool
	go func() {
		for {
			interval := time.Minute

			now := time.Now()

			// fetch all office hours for today
			hours, err := w.resolver.ResolveOfficeHours(ctx, now)
			if err == nil {
				var (
					min         time.Time
					isOpen      bool
					appliedHour *office_hoursv1.OfficeHour
				)

				for _, h := range hours {
					for _, tr := range h.TimeRanges {
						start := tr.Start.At(now)
						end := tr.End.At(now)

						if start.After(now) && (start.Before(min) || min.IsZero()) {
							min = start
						}

						if end.After(now) && (end.Before(min) || min.IsZero()) {
							min = end
						}

						// check if the office hour currntly applies
						if tr.At(now).Includes(now) {
							isOpen = true
							appliedHour = h
						}
					}
				}

				if !min.IsZero() {
					// get the expect interval at which the office-hour state will change
					interval = time.Until(min)
					slog.Info("waiting for office-hour change", "expectedChange", min.Format(time.RFC3339))
				}

				// Publish an OpenChangeEvent
				if wasOpen == nil || *wasOpen != isOpen {
					wasOpen = &isOpen

					pb, err := anypb.New(&office_hoursv1.OpenChangeEvent{
						IsOpen:     isOpen,
						OfficeHour: appliedHour,
					})
					if err == nil {
						_, err = w.eventClient.Publish(ctx, connect.NewRequest(&eventsv1.Event{
							Event: pb,
						}))
					}

					if err != nil {
						slog.Error("failed to publish OpenChangeEvent", "error", err)
					}
				}

			} else {
				slog.Error("failed to resolve office hours", "error", err)
			}

			select {
			case <-time.After(interval):
				slog.Info("checking current office-hour state", "trigger", "interval")

			case <-w.trigger:
				slog.Info("checking current office-hour state", "trigger", "service")

			case <-ctx.Done():
				slog.Info("exiting watcher loop, context cancelled")
				return
			}
		}
	}()
}

func (w *Watcher) Trigger() {
	if w == nil {
		return
	}

	select {
	case w.trigger <- struct{}{}:
	default:
	}
}
