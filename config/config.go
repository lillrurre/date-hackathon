package config

import (
	"os"
	"time"
)

type Config struct {
	ApiKey         string
	Url            string
	SystemPrompt   string
	RequestTimeout time.Duration
}

func LoadConfig() *Config {
	return &Config{
		ApiKey:         os.Getenv("BOT_API_KEY"),
		Url:            os.Getenv("BOT_URL"),
		SystemPrompt:   os.Getenv("SYSTEM_PROMPT"),
		RequestTimeout: time.Second * 10,
	}
}
