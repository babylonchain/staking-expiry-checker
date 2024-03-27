package config

import (
	"fmt"
)

type QueueConfig struct {
	User              string `mapstructure:"user"`
	Pass              string `mapstructure:"pass"`
	Url               string `mapstructure:"url"`
	ProcessingTimeout int    `mapstructure:"processing_timeout"`
}

func (cfg *QueueConfig) Validate() error {
	if cfg.User == "" {
		return fmt.Errorf("missing queue user")
	}

	if cfg.Pass == "" {
		return fmt.Errorf("missing queue password")
	}

	if cfg.Url == "" {
		return fmt.Errorf("missing queue url")
	}

	if cfg.ProcessingTimeout <= 0 {
		return fmt.Errorf("invalid queue processing timeout")
	}

	return nil
}
