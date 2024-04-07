package main

type Config struct {
	CRI      string `env:"CRI" envDefault:"cri-o"`
	LogLevel string `env:"LOG_LEVEL" envDefault:"INFO"`
}

func NewConfig() *Config {
	return &Config{}
}
