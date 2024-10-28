package resolver

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bufbuild/connect-go"
	calendarv1 "github.com/tierklinik-dobersberg/apis/gen/go/tkd/calendar/v1"
	v1 "github.com/tierklinik-dobersberg/apis/gen/go/tkd/office_hours/v1"
	"github.com/tierklinik-dobersberg/apis/pkg/discovery"
	"github.com/tierklinik-dobersberg/apis/pkg/discovery/wellknown"
	"github.com/tierklinik-dobersberg/office-hours-service/internal/repo"
)

type Resolver struct {
	repo    *repo.Repo
	catalog discovery.Discoverer
}

func NewResolver(repo *repo.Repo, catalog discovery.Discoverer) *Resolver {
	return &Resolver{
		repo:    repo,
		catalog: catalog,
	}
}

func (r *Resolver) ResolveOfficeHours(ctx context.Context, t time.Time) ([]*v1.OfficeHour, error) {
	hours, err := r.repo.FindByTime(ctx, t)
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
	isHoliday, err := r.isHoliday(ctx, t)
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

func (r *Resolver) isHoliday(ctx context.Context, t time.Time) (bool, error) {
	holidayClient, err := wellknown.HolidayService.Create(ctx, r.catalog)
	if err != nil {
		return false, fmt.Errorf("failed to get holiday client using service catalog: %w", err)
	}

	holidayResponse, err := holidayClient.GetHoliday(ctx, connect.NewRequest(&calendarv1.GetHolidayRequest{
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
