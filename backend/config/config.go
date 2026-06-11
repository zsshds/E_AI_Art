package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	MongoDB  MongoDBConfig  `yaml:"mongodb"`
	RabbitMQ RabbitMQConfig `yaml:"rabbitmq"`
	OpenAI   OpenAIConfig   `yaml:"openai"`
	OSS      OSSConfig      `yaml:"oss"`
	Worker   WorkerConfig   `yaml:"worker"`
	Auth     AuthConfig     `yaml:"auth"`
}

type AuthConfig struct {
	JWTSecret string `yaml:"jwt_secret"`
	AdminUser string `yaml:"admin_user"`
	AdminPass string `yaml:"admin_pass"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type MongoDBConfig struct {
	URI      string `yaml:"uri"`
	Database string `yaml:"database"`
}

type RabbitMQConfig struct {
	URI string `yaml:"uri"`
}

type OpenAIConfig struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
}

type OSSConfig struct {
	Bucket    string `yaml:"bucket"`
	Endpoint  string `yaml:"endpoint"`
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
}

type WorkerConfig struct {
	Concurrency            int `yaml:"concurrency"`
	TimeoutSec             int `yaml:"timeout_sec"` // backward compat
	TaskTimeoutSec         int `yaml:"task_timeout_sec"`
	ImageRequestTimeoutSec int `yaml:"image_request_timeout_sec"`
	DownloadTimeoutSec     int `yaml:"download_timeout_sec"`
	MaxRetry               int `yaml:"max_retry"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		MongoDB: MongoDBConfig{
			URI:      "mongodb://localhost:27017",
			Database: "imagegen",
		},
		Worker: WorkerConfig{
			Concurrency:            5,
			TimeoutSec:             300,
			TaskTimeoutSec:         300,
			ImageRequestTimeoutSec: 600,
			DownloadTimeoutSec:     120,
			MaxRetry:               2,
		},
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	if cfg.Worker.TaskTimeoutSec <= 0 {
		if cfg.Worker.TimeoutSec > 0 {
			cfg.Worker.TaskTimeoutSec = cfg.Worker.TimeoutSec
		} else {
			cfg.Worker.TaskTimeoutSec = 300
		}
	}
	if cfg.Worker.ImageRequestTimeoutSec <= 0 {
		cfg.Worker.ImageRequestTimeoutSec = cfg.Worker.TaskTimeoutSec
	}
	if cfg.Worker.DownloadTimeoutSec <= 0 {
		cfg.Worker.DownloadTimeoutSec = 120
	}
	cfg.Worker.TimeoutSec = cfg.Worker.TaskTimeoutSec

	// 环境变量覆盖
	if v := os.Getenv("OPENAI_API_KEY"); v != "" {
		cfg.OpenAI.APIKey = v
	}
	if v := os.Getenv("MONGO_URI"); v != "" {
		cfg.MongoDB.URI = v
	}
	if v := os.Getenv("RABBITMQ_URI"); v != "" {
		cfg.RabbitMQ.URI = v
	}
	if v := os.Getenv("OSS_BUCKET"); v != "" {
		cfg.OSS.Bucket = v
	}
	if v := os.Getenv("OSS_ENDPOINT"); v != "" {
		cfg.OSS.Endpoint = v
	}
	if v := os.Getenv("OSS_ACCESS_KEY"); v != "" {
		cfg.OSS.AccessKey = v
	}
	if v := os.Getenv("OSS_SECRET_KEY"); v != "" {
		cfg.OSS.SecretKey = v
	}
	if v := os.Getenv("SERVER_PORT"); v != "" {
		// port override handled by main.go
	}
	if v := os.Getenv("OPENAI_BASE_URL"); v != "" {
		cfg.OpenAI.BaseURL = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.Auth.JWTSecret = v
	}
	if v := os.Getenv("ADMIN_USER"); v != "" {
		cfg.Auth.AdminUser = v
	}
	if v := os.Getenv("ADMIN_PASS"); v != "" {
		cfg.Auth.AdminPass = v
	}

	return cfg, nil
}
