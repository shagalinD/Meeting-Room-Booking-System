package config

import "time"

type ServerConfig struct {
	JWTAccessSecret   string        `mapstructure:"jwt_access_secret"`
	JWTRefreshSecret  string        `mapstructure:"jwt_refresh_secret"`
	LogLevel          string        `mapstructure:"log_level"`
	Port              string        `mapstructure:"port"`
	ReadTimeout       time.Duration `mapstructure:"read_timeout"`
	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"`
	WriteTimeout      time.Duration `mapstructure:"write_timeout"`
	IdleTimeout       time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout   time.Duration `mapstructure:"shutdown_timeout"`
}
