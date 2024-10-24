package repo

import (
	"fmt"
	"time"

	commonv1 "github.com/tierklinik-dobersberg/apis/gen/go/tkd/common/v1"
	office_hoursv1 "github.com/tierklinik-dobersberg/apis/gen/go/tkd/office_hours/v1"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TimeRange struct {
	From time.Time `bson:"from,omitempty"`
	To   time.Time `bson:"to,omitempty"`
}

type DayTime struct {
	Hours   int `bson:"hours"`
	Minutes int `bson:"minutes"`
	Seconds int `bson:"seconds"`
}

type DayTimeRange struct {
	Start DayTime `bson:"start"`
	End   DayTime `bson:"end"`
}

type OfficeHourModel struct {
	ID               primitive.ObjectID              `bson:"_id"`
	DayOfWeek        time.Weekday                    `bson:"dayOfWeek,omitempty"`
	Date             string                          `bson:"date,omitempty"`
	HolidayCondition office_hoursv1.HolidayCondition `bson:"holiday,omitempty"`
	TimeRanges       []DayTimeRange                  `bson:"timeRanges"` // no omitempty!
}

func (m OfficeHourModel) ToProto() *office_hoursv1.OfficeHour {
	res := &office_hoursv1.OfficeHour{
		Name:             m.ID.Hex(),
		HolidayCondition: m.HolidayCondition,
	}

	switch {
	case m.DayOfWeek != 0:
		res.Kind = &office_hoursv1.OfficeHour_DayOfWeek{
			DayOfWeek: commonv1.FromWeekday(m.DayOfWeek),
		}

	case m.Date != "":
		date, err := commonv1.ParseDate(m.Date)
		if err != nil {
			t, err := time.Parse(m.Date, "01-02")
			if err == nil {
				date = &commonv1.Date{
					Month: commonv1.FromMonth(t.Month()),
					Day:   int32(t.Day()),
				}
			}
		}

		res.Kind = &office_hoursv1.OfficeHour_Date{
			Date: date,
		}
	}

	res.TimeRanges = make([]*commonv1.DayTimeRange, len(m.TimeRanges))

	for idx, tr := range m.TimeRanges {
		res.TimeRanges[idx] = &commonv1.DayTimeRange{
			Start: &commonv1.DayTime{
				Hour:   int32(tr.Start.Hours),
				Minute: int32(tr.Start.Minutes),
				Second: int32(tr.Start.Seconds),
			},
			End: &commonv1.DayTime{
				Hour:   int32(tr.End.Hours),
				Minute: int32(tr.End.Minutes),
				Second: int32(tr.End.Seconds),
			},
		}
	}

	return res
}

func ModelFromProto(pb *office_hoursv1.OfficeHour) (*OfficeHourModel, error) {
	var oid primitive.ObjectID

	if pb.Name != "" {
		var err error
		oid, err = primitive.ObjectIDFromHex(pb.Name)
		if err != nil {
			return nil, fmt.Errorf("invalid office hour name: %w", err)
		}
	}

	res := &OfficeHourModel{
		ID:               oid,
		HolidayCondition: pb.HolidayCondition,
	}

	switch v := pb.Kind.(type) {
	case *office_hoursv1.OfficeHour_Date:
		if v.Date.Year != 0 {
			res.Date = v.Date.AsTime().Format("2006-01-02")
		} else {
			res.Date = v.Date.AsTime().Format("01-02")
		}

	case *office_hoursv1.OfficeHour_DayOfWeek:
		res.DayOfWeek = v.DayOfWeek.ToWeekday()

	default:
		return nil, fmt.Errorf("missing OfficeHour.kind")
	}

	res.TimeRanges = make([]DayTimeRange, len(pb.TimeRanges))
	if len(pb.TimeRanges) == 0 {
		return nil, fmt.Errorf("missing time ranges")
	}

	for idx, tr := range pb.TimeRanges {
		res.TimeRanges[idx] = DayTimeRange{
			Start: DayTime{
				Hours:   int(tr.GetStart().GetHour()),
				Minutes: int(tr.GetStart().GetMinute()),
				Seconds: int(tr.GetStart().GetSecond()),
			},
			End: DayTime{
				Hours:   int(tr.GetEnd().GetHour()),
				Minutes: int(tr.GetEnd().GetMinute()),
				Seconds: int(tr.GetEnd().GetSecond()),
			},
		}
	}

	return res, nil
}
