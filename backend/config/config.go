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
	APIKey string `yaml:"api_key"`
}

type OSSConfig struct {
	Bucket    string `yaml:"bucket"`
	Endpoint  string `yaml:"endpoint"`
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
}

type WorkerConfig struct {
	Concurrency int `yaml:"concurrency"`
	TimeoutSec  int `yaml:"timeout_sec"`
	MaxRetry    int `yaml:"max_retry"`
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
			Concurrency: 5,
			TimeoutSec:  150,
			MaxRetry:    2,
		},
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

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
	if v := os.Getenv("WORKER_CONCURRENCY"); v != "" {
		// concurrency override handled by main.go
	}

	return cfg, nil
}
