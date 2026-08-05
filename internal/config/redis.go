package config

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	Host         string        `mapstructure:"host"`
	Port         string        `mapstructure:"port"`
	User         string        `mapstructure:"user"`
	Password     string        `mapstructure:"password"`
	Index        int           `mapstructure:"index"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}


func ConnectRedis(config *RedisConfig) (*redis.Client, error) {
	db := redis.NewClient(&redis.Options{
		Addr:         config.Host + ":" + config.Port,
		Password:     config.Password,
		DB:           config.Index,
		Username:     config.User,
		MaxRetries:   5,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	})

	pingContext, cancel := context.WithTimeout(context.Background(), config.DialTimeout)
	defer cancel()

	if err := db.Ping(pingContext).Err(); err != nil {
		return nil, fmt.Errorf("error on connecting to redis: %w", err)
	}

	return db, nil
}
