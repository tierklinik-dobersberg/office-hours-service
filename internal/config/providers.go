package config

import (
	"github.com/tierklinik-dobersberg/apis/gen/go/tkd/calendar/v1/calendarv1connect"
	"github.com/tierklinik-dobersberg/apis/gen/go/tkd/idm/v1/idmv1connect"
)

type Providers struct {
	*Config

	calendarv1connect.HolidayServiceClient
	idmv1connect.UserServiceClient
	idmv1connect.RoleServiceClient
}
