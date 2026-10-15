package config

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

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

func (cfg *PostgresConfig) ConnectionString() string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   net.JoinHostPort(cfg.Host, cfg.Port),
		Path:   cfg.Name,
	}
	u.RawQuery = url.Values{"sslmode": {"disable"}}.Encode()
	return u.String()
}

func ConnectDB(cfg *PostgresConfig, dsn string) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("error on parsing postgres config: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	poolConfig.MinConns = int32(cfg.MaxIdleConns)
	poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime
	poolConfig.MaxConnIdleTime = cfg.ConnMaxIdleTime
	poolConfig.HealthCheckPeriod = cfg.ConnectTimeout

	dbpool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("error on creating postgres pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer cancel()

	if err := dbpool.Ping(pingCtx); err != nil {
		return nil, fmt.Errorf("timeout on pinging postgres: %w", err)
	}

	return dbpool, nil
}
