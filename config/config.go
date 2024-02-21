package config

import (
	"os"
	"time"
)

type Config struct {
	ApiKey         string
	Url            string
	RequestTimeout time.Duration
}

func LoadConfig() *Config {
	return &Config{
		ApiKey:         os.Getenv("BOT_API_KEY"),
		Url:            os.Getenv("BOT_URL"),
		RequestTimeout: time.Second * 10,
	}
}
