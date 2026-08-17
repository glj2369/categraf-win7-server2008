package install

import (
	"github.com/kardianos/service"
)

const (
	ServiceName = "categraf"
)

var (
	serviceConfig = &service.Config{
		Name:        ServiceName,
		DisplayName: "categraf",
		Description: "Opensource telemetry collector",
		Option: service.KeyValue{
			"DelayedAutoStart": true,
		},
	}
)

func ServiceConfig() *service.Config {
	return serviceConfig
}
