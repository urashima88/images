package app_config

import (
	"flag"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Env string
	Db
	HTTPServer
}

type HTTPServer struct {
	Host        string
	Port        string
	Timeout     time.Duration
	IdleTimeout time.Duration
}

type Db struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

var envs = []string{"local", "dev", "prod"}

func MustLoad() *Config {
	var path, env string
	flag.StringVar(&path, "config", "", "path to config file")
	flag.StringVar(&env, "env", "", "environment")
	flag.Parse()

	if path == "" {
		panic("config file is empty")
	}

	err := godotenv.Load(path)
	if err != nil {
		panic("error loading .env: " + err.Error())
	}

	var cfg Config
	if slices.Contains(envs, env) {
		GetHTTPServerEnv(env, &cfg)
	} else {
		panic("env doesn't match one of the values: local, dev, prod")
	}

	cfg.Env = env

	cfg.Db.Host = os.Getenv("DB_HOST")
	cfg.Db.Port = os.Getenv("DB_PORT")
	cfg.Db.User = os.Getenv("DB_USER")
	cfg.Db.Password = os.Getenv("DB_PASSWORD")
	cfg.Db.Name = os.Getenv("DB_NAME")
	cfg.Db.SSLMode = os.Getenv("SSL_MODE")

	return &cfg
}

func GetHTTPServerEnv(env string, cfg *Config) {
	env = strings.ToUpper(env)
	cfg.HTTPServer.Host = os.Getenv(env + "_APP_HOST")
	cfg.HTTPServer.Port = os.Getenv(env + "_APP_PORT")
	var err error
	if cfg.HTTPServer.Timeout, err = time.ParseDuration(os.Getenv(env + "_TIMEOUT")); err != nil {
		panic("error parsing timeout duration")
	}
	if cfg.HTTPServer.IdleTimeout, err = time.ParseDuration(os.Getenv(env + "_IDLE_TIMEOUT")); err != nil {
		panic("error parsing idle timeout")
	}

}
