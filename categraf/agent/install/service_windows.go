package install

import (
	"github.com/kardianos/service"
)

const (
	ServiceName = "categraf"
)

var (
	serviceConfig = &service.Config{
		Name:         ServiceName,
		DisplayName:  "categraf",
		Description:  "Opensource telemetry collector",
		Dependencies: []string{"Tcpip"},
		Option: service.KeyValue{
			"DelayedAutoStart":       true,
			"OnFailure":              "restart",
			"OnFailureDelayDuration": "10s",
			"OnFailureResetPeriod":   120,
		},
	}
)

func ServiceConfig() *service.Config {
	return serviceConfig
}
