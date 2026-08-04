package config

import "time"

type PostgresConfig struct {
	Name            string        `mapstructure:"name"`
	Port            string        `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Host            string        `mapstructure:"host"`
	ConnectTimeout  time.Duration `mapstructure:"connect_timeout"`
	QueryTimeout    time.Duration `mapstructure:"query_timeout"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
}