package config

type ServerConfig struct {
	JWTSecret string `mapstructure:"jwt_secret"`
	LogLevel  string `mapstructure:"log_level"`
	Port      string `mapstructure:"port"`
}