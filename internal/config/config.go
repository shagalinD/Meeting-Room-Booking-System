package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Postgres PostgresConfig `mapstructure:"postgres"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Server   ServerConfig   `mapstructure:"server"`
}

var AppConfig = &Config{}

func LoadConfig() error {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("error on loading .env file: %w", err)
	}

	v := viper.New()
	setDefaults(v)

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath(filepath.Join(".", "internal", "config"))

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error on reading config file: %w", err)
		}
	}

	bindEnvs(v)

	if err := v.Unmarshal(&AppConfig); err != nil {
		return fmt.Errorf("error on unmarshaling config: %w", err)
	}

	return nil
}

func bindEnvs(v *viper.Viper) {
	v.BindEnv("postgres.user", "POSTGRES_USER")
	v.BindEnv("postgres.password", "POSTGRES_PASSWORD")
	v.BindEnv("postgres.name", "POSTGRES_DB")
	v.BindEnv("postgres.port", "POSTGRES_PORT")
	v.BindEnv("postgres.host", "POSTGRES_HOST")
	v.BindEnv("postgres.connect_timeout", "POSTGRES_CONNECT_TIMEOUT")
	v.BindEnv("postgres.query_timeout", "POSTGRES_QUERY_TIMEOUT")
	v.BindEnv("postgres.max_open_conns", "POSTGRES_MAX_OPEN_CONNS")
	v.BindEnv("postgres.max_idle_conns", "POSTGRES_MAX_IDLE_CONNS")
	v.BindEnv("postgres.conn_max_lifetime", "POSTGRES_CONN_MAX_LIFETIME")
	v.BindEnv("postgres.conn_max_idle_time", "POSTGRES_CONN_MAX_IDLE_TIME")

	v.BindEnv("redis.user", "REDIS_USER")
	v.BindEnv("redis.password", "REDIS_PASSWORD")
	v.BindEnv("redis.port", "REDIS_PORT")
	v.BindEnv("redis.host", "REDIS_HOST")
	v.BindEnv("redis.index", "REDIS_INDEX")
	v.BindEnv("redis.dial_timeout", "REDIS_DIAL_TIMEOUT")
	v.BindEnv("redis.read_timeout", "REDIS_READ_TIMEOUT")
	v.BindEnv("redis.write_timeout", "REDIS_WRITE_TIMEOUT")

	v.BindEnv("server.log_level", "LOG_LEVEL")
	v.BindEnv("server.jwt_access_secret", "JWT_ACCESS_SECRET")
	v.BindEnv("server.jwt_refresh_secret", "JWT_REFRESH_SECRET")
	v.BindEnv("server.port", "SERVER_PORT")
	v.BindEnv("server.read_timeout", "SERVER_READ_TIMEOUT")
	v.BindEnv("server.read_header_timeout", "SERVER_READ_HEADER_TIMEOUT")
	v.BindEnv("server.write_timeout", "SERVER_WRITE_TIMEOUT")
	v.BindEnv("server.idle_timeout", "SERVER_IDLE_TIMEOUT")
	v.BindEnv("server.shutdown_timeout", "SERVER_SHUTDOWN_TIMEOUT")
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("postgres.user", "")
	v.SetDefault("postgres.password", "")
	v.SetDefault("postgres.name", "")
	v.SetDefault("postgres.port", "5432")
	v.SetDefault("postgres.host", "localhost")
	v.SetDefault("postgres.connect_timeout", "5s")
	v.SetDefault("postgres.query_timeout", "10s")
	v.SetDefault("postgres.max_open_conns", 25)
	v.SetDefault("postgres.max_idle_conns", 25)
	v.SetDefault("postgres.conn_max_lifetime", "5m")
	v.SetDefault("postgres.conn_max_idle_time", "1m")

	v.SetDefault("redis.user", "default")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.port", "6379")
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.index", 0)
	v.SetDefault("redis.dial_timeout", "5s")
	v.SetDefault("redis.read_timeout", "3s")
	v.SetDefault("redis.write_timeout", "3s")

	v.SetDefault("server.log_level", "debug")
	v.SetDefault("server.jwt_access_secret", "my_jwt_access_secret")
	v.SetDefault("server.jwt_refresh_secret", "my_jwt_refresh_secret")
	v.SetDefault("server.port", "8081")
	v.SetDefault("server.read_timeout", "5s")
	v.SetDefault("server.read_header_timeout", "2s")
	v.SetDefault("server.write_timeout", "10s")
	v.SetDefault("server.idle_timeout", "60s")
	v.SetDefault("server.shutdown_timeout", "15s")
}
