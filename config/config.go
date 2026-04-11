package config

import (
	"fmt"

	"github.com/wb-go/wbf/config"
)

type Config struct {
	AppPort int

	Postgres PostgresConfig
	RabbitMQ RabbitMQConfig
	Redis    RedisConfig

	Telegram TelegramConfig
}

type PostgresConfig struct {
	Host     string
	Port     int
	DB       string
	User     string
	Password string
}

type RabbitMQConfig struct {
	URL string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type TelegramConfig struct {
	BotToken string
}

func Load() (*Config, error) {
	cfg := config.New()

	if err := cfg.LoadEnvFiles(".env"); err != nil {
		return nil, fmt.Errorf("load .env: %w", err)
	}

	cfg.EnableEnv("")

	cfg.SetDefault("APP_PORT", 8080)
	cfg.SetDefault("POSTGRES_PORT", 5432)
	cfg.SetDefault("REDIS_DB", 0)

	res := &Config{
		AppPort: cfg.GetInt("APP_PORT"),
		Postgres: PostgresConfig{
			Host:     cfg.GetString("POSTGRES_HOST"),
			Port:     cfg.GetInt("POSTGRES_PORT"),
			DB:       cfg.GetString("POSTGRES_DB"),
			User:     cfg.GetString("POSTGRES_USER"),
			Password: cfg.GetString("POSTGRES_PASSWORD"),
		},
		RabbitMQ: RabbitMQConfig{
			URL: cfg.GetString("RABBITMQ_URL"),
		},
		Redis: RedisConfig{
			Addr:     cfg.GetString("REDIS_ADDR"),
			Password: cfg.GetString("REDIS_PASSWORD"),
			DB:       cfg.GetInt("REDIS_DB"),
		},
		Telegram: TelegramConfig{
			BotToken: cfg.GetString("TG_BOT_TOKEN"),
		},
	}

	return res, nil
}

func (p PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		p.User,
		p.Password,
		p.Host,
		p.Port,
		p.DB,
	)
}
