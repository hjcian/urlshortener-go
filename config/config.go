package config

import (
	"errors"

	"github.com/kelseyhightower/envconfig"
)

const (
	InMemory = "inmemory"
	Redis    = "redis"
)

type Env struct {
	AppPort        int    `envconfig:"APP_PORT"    default:"8080"`
	DBHost         string `envconfig:"DB_HOST"     default:"localhost"`
	DBPort         int    `envconfig:"DB_PORT"     default:"5555"`
	DBName         string `envconfig:"DB_NAME"     default:"test"`
	DBUser         string `envconfig:"DB_USER"     default:"test"`
	DBPassword     string `envconfig:"DB_PASSWORD" default:"test"`
	CacheMode      string `envconfig:"CACHE_MODE"  default:"inmemory"`
	CacheHost      string `envconfig:"CACHE_HOST"  default:"localhost"`
	CachePort      int    `envconfig:"CACHE_PORT"  default:"6679"`
	RedirectOrigin string `envconfig:"REDIRECT_ORIGIN"  default:"http://localhost:8080"`
}

func Process() (env Env, err error) {
	err = envconfig.Process("", &env)
	if err != nil {
		return env, err
	}
	err = validate(env)
	return env, err
}

func validate(env Env) error {
	switch env.CacheMode {
	case InMemory:
	case Redis:
		if env.CacheHost == "" || env.CachePort == 0 {
			return errors.New("redis cache mode need host and port")
		}
	default:
		return errors.New("undefined cache mode: " + env.CacheMode)
	}
	return nil
}
