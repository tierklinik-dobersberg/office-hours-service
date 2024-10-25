package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/bufbuild/connect-go"
	calendarv1 "github.com/tierklinik-dobersberg/apis/gen/go/tkd/calendar/v1"
	commonv1 "github.com/tierklinik-dobersberg/apis/gen/go/tkd/common/v1"
	v1 "github.com/tierklinik-dobersberg/apis/gen/go/tkd/office_hours/v1"
	"github.com/tierklinik-dobersberg/apis/gen/go/tkd/office_hours/v1/office_hoursv1connect"
	"github.com/tierklinik-dobersberg/office-hours-service/internal/config"
	"github.com/tierklinik-dobersberg/office-hours-service/internal/repo"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Service struct {
	office_hoursv1connect.UnimplementedOfficeHourServiceHandler

	repo      *repo.Repo
	providers *config.Providers
}

func New(repo *repo.Repo, providers *config.Providers) *Service {
	return &Service{
		repo:      repo,
		providers: providers,
	}
}

func (svc *Service) ListHours(ctx context.Context, req *connect.Request[v1.ListHoursRequest]) (*connect.Response[v1.ListHoursResponse], error) {
	hours, err := svc.repo.ListOfficeHours(ctx)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&v1.ListHoursResponse{
		OfficeHours: hours,
	}), nil
}

func (svc *Service) UpsertOfficeHour(ctx context.Context, req *connect.Request[v1.OfficeHour]) (*connect.Response[v1.OfficeHour], error) {
	hour, err := svc.repo.UpsertOfficeHours(ctx, req.Msg)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(hour), nil
}

func (svc *Service) DeleteOfficeHour(ctx context.Context, req *connect.Request[v1.DeleteOfficeHourRequest]) (*connect.Response[emptypb.Empty], error) {
	if err := svc.repo.DeleteOfficeHour(ctx, req.Msg.Name); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}

		return nil, err
	}

	return connect.NewResponse(new(emptypb.Empty)), nil
}

func (svc *Service) OfficeHourRanges(ctx context.Context, req *connect.Request[v1.OfficeHourRangesRequest]) (*connect.Response[v1.OfficeHourRangesResponse], error) {
	t := time.Now().Local()

	if req.Msg.Date != nil {
		t = req.Msg.Date.AsTimeInLocation(time.Local)
	}

	hours, err := svc.resolveOfficeHours(ctx, t)
	if err != nil {
		return nil, err
	}
	if len(hours) == 0 {
		return connect.NewResponse(new(v1.OfficeHourRangesResponse)), nil
	}

	if len(hours) > 1 {
		slog.Warn("found multiple office hours, only considering the first one", "time", t.Format(time.RFC3339))
	}

	res := &v1.OfficeHourRangesResponse{
		OfficeHour: hours[0],
		OpenRanges: make([]*commonv1.TimeRange, len(hours[0].TimeRanges)),
	}

	for idx, tr := range hours[0].TimeRanges {
		res.OpenRanges[idx] = tr.At(t)
	}

	return connect.NewResponse(res), nil
}

func (svc *Service) IsOpen(ctx context.Context, req *connect.Request[v1.IsOpenRequest]) (*connect.Response[v1.IsOpenResponse], error) {
	t := time.Now()

	if req.Msg.Timestamp.IsValid() {
		t = req.Msg.Timestamp.AsTime()
	}

	// switch t to local time
	t = t.Local()

	hours, err := svc.resolveOfficeHours(ctx, t)
	if err != nil {
		return nil, err
	}

	isOpen := false
	var appliedHours *v1.OfficeHour
	for _, h := range hours {
		for _, tr := range h.TimeRanges {
			if tr.At(t).Includes(t) {
				isOpen = true
				appliedHours = h
				break
			}
		}
	}

	return connect.NewResponse(&v1.IsOpenResponse{
		Open:       isOpen,
		OfficeHour: appliedHours,
	}), nil
}

func (svc *Service) resolveOfficeHours(ctx context.Context, t time.Time) ([]*v1.OfficeHour, error) {
	hours, err := svc.repo.FindByTime(ctx, t)
	if err != nil {
		// If it's a NotFound error there are not office hours for the given date,
		// thus, just return a normal response.
		if errors.Is(err, repo.ErrNotFound) {
			return nil, nil
		}

		// otherwise, return the error to the caller
		return nil, err
	}

	// now, check if t is a public holiday
	isHoliday, err := svc.isHoliday(ctx, t)
	if err != nil {
		return nil, err
	}

	// we actually expect to only find one
	validHours := make([]*v1.OfficeHour, 0, 1)
	for _, h := range hours {
		switch {
		case isHoliday && h.HolidayCondition != v1.HolidayCondition_HOLIDAY_CONDITION_UNSPECIFIED:
			validHours = append(validHours, h)

		case !isHoliday && h.HolidayCondition != v1.HolidayCondition_EXCLUSIVE:
			validHours = append(validHours, h)
		}
	}

	return validHours, nil
}

func (svc *Service) isHoliday(ctx context.Context, t time.Time) (bool, error) {
	holidayResponse, err := svc.providers.GetHoliday(ctx, connect.NewRequest(&calendarv1.GetHolidayRequest{
		Year:  uint64(t.Year()),
		Month: uint64(t.Month()),
	}))

	if err != nil {
		return false, fmt.Errorf("failed to fetch holidays: %w", err)
	}

	dateKey := t.Format("2006-01-02")
	for _, holiday := range holidayResponse.Msg.Holidays {
		if holiday.Date == dateKey {
			if holiday.Type == calendarv1.HolidayType_PUBLIC {
				return true, nil
			}

			break
		}
	}

	return false, nil
}
