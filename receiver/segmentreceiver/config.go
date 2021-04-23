package segmentreceiver

import (
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/confighttp"
)

type Config struct {
	config.ReceiverSettings       `mapstructure:",squash"`
	confighttp.HTTPServerSettings `mapstructure:",squash"`
}

var _ config.Receiver = (*Config)(nil)

func (cfg *Config) Validate() error {
	return nil
}
