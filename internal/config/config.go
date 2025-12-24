package config

import "time"

// Config описывает все параметры приложения.
type Config struct {
	App      AppConfig      `toml:"app"`
	HTTP     HTTPConfig     `toml:"http"`
	Postgres PostgresConfig `toml:"postgres"`
	Logger   LoggerConfig   `toml:"logger"`
	Rabbit   RabbitConfig   `toml:"rabbit"`
	JWT      JWTConfig      `toml:"jwt"`
}

type JWTConfig struct {
	Secret string        `toml:"secret"`
	TTL    time.Duration `toml:"ttl"`
}

type RabbitConfig struct {
	AMPQURL string `toml:"ampq_url"`
}

// AppConfig общие сведения о приложении (имя, окружение).
type AppConfig struct {
	Name        string `toml:"name"`
	Environment string `toml:"environment"` // dev / prod / test
}

// HTTPConfig настройки HTTP-сервера.
type HTTPConfig struct {
	Host string `toml:"host"` // "0.0.0.0"
	Port int    `toml:"port"` // 8080
}

// PostgresConfig параметры подключения к БД.
type PostgresConfig struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	User     string `toml:"user"`
	Password string `toml:"password"`
	Database string `toml:"database"`
	SSLMode  string `toml:"sslmode"` // disable / require
}

// LoggerConfig параметры логгера.
type LoggerConfig struct {
	Level string `toml:"level"` // debug / info / warn / error
}
