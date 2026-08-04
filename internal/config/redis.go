package config

import "time"

type RedisConfig struct {
	Host        string `mapstructure:"host"`
	Port        string `mapstructure:"port"`
	User        string `mapstructure:"user"`
	Password    string `mapstructure:"password"`
	Index       string `mapstructure:"index"`
	DialTimeout time.Duration `mapstructure:"dial_timeout"`
	

}
