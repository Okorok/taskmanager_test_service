package config

import (
	"os"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type HTTPConfig struct {
	Addr         string        `yaml:"addr"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

type DBConfig struct {
	DSN             string        `yaml:"dsn"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

type RedisConfig struct {
	Addr        string        `yaml:"addr"`
	Password    string        `yaml:"password"`
	DB          int           `yaml:"db"`
	TaskListTTL time.Duration `yaml:"task_list_ttl"`
}

type AuthConfig struct {
	JWTSecret string        `yaml:"jwt_secret"`
	TokenTTL  time.Duration `yaml:"token_ttl"`
}

type Config struct {
	HTTP  HTTPConfig  `yaml:"http"`
	DB    DBConfig    `yaml:"db"`
	Redis RedisConfig `yaml:"redis"`
	Auth  AuthConfig  `yaml:"auth"`
}

func LoadConfig(configFile string) (*Config, error) {
	configData, err := os.ReadFile(configFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read config file")
	}

	conf := &Config{}
	if err = yaml.Unmarshal(configData, conf); err != nil {
		return nil, errors.Wrap(err, "failed to parse config")
	}

	return conf, nil
}
