package config

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Postgres PostgresConfig `mapstructure:"postgres"`
	Redis 	 RedisConfig `mapstructure:"redis"`
	Server   ServerConfig `mapstructure:"server"`
}

var AppConfig = &Config{}

func LoadConfig() error {
	err := godotenv.Load()

	if err != nil {
		return err
	}

	v := viper.New()
	setDefaults(v)

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")   


		// 2. Читаем YAML файл
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

	v.BindEnv("redis.password", "REDIS_PASSWORD")
	v.BindEnv("redis.user", "REDIS_USER")
	v.BindEnv("redis.port", "REDIS_PORT")
	v.BindEnv("redis.host", "REDIS_HOST")
	v.BindEnv("redis.index", "REDIS_INDEX")

	v.BindEnv("server.log_level", "LOG_LEVEL")
	v.BindEnv("server.jwt_secret", "JWT_SECRET")
	v.BindEnv("server.port", "SERVER_PORT")
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("postgres.user", "")
	v.SetDefault("postgres.password", "")
	v.SetDefault("postgres.name", "")
	v.SetDefault("postgres.port", "")
	v.SetDefault("postgres.host", "")
	v.SetDefault("postgres.connect_timeout", "5s")
	v.SetDefault("postgres.query_timeout", "10s")
	v.SetDefault("postgres.max_open_conns", 25)
	v.SetDefault("postgres.max_idle_conns", 25)
	v.SetDefault("postgres.conn_max_lifetime", "5m")
	v.SetDefault("postgres.conn_max_idle_time", "1m")

	v.SetDefault("redis.password", "")
	v.SetDefault("redis.user", "")
	v.SetDefault("redis.port", "")
	v.SetDefault("redis.host", "")
	v.SetDefault("redis.index", "")

	v.SetDefault("server.log_level", "")
	v.SetDefault("server.jwt_secret", "")
	v.SetDefault("server.server_port", "")
}
