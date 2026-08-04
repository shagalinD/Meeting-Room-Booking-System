package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shdmitri/booking-service/internal/config"
)

type Config struct {
	Addr        string        `yaml:"addr"`
	Password    string        `yaml:"password"`
	User        string        `yaml:"user"`
	DB          int           `yaml:"db"`
	MaxRetries  int           `yaml:"max_retries"`
	DialTimeout time.Duration `yaml:"dial_timeout"`
	Timeout     time.Duration `yaml:"timeout"`
}

func ConnectRedis(config *config.RedisConfig) (*redis.Client, error) {
	dbid, err := strconv.Atoi(config.Index)

	if err != nil {
		return nil, fmt.Errorf("error on converting dbid: %w", err)
	}
	db := redis.NewClient(&redis.Options{
		Addr:         config.Host + ":" + config.Port,
		Password:     config.Password,
		DB:           dbid,
		Username:     config.User,
		MaxRetries:   5,
		DialTimeout:  time.Second * 15,
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 5,
	})

	pingContext, cancel := context.WithTimeout(context.Background(), time.Second * 15)
	defer cancel()

	if err := db.Ping(pingContext).Err(); err != nil {
		return nil, fmt.Errorf("error on connecting to redis: %w", err)
	}

	return db, nil
}