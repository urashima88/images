package app_config

import (
	"flag"
	"slices"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Env string
	Db
	HTTPServer
	ImageMeta
	FileServer
	TagMeta
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

type ImageMeta struct {
	MaxImageSize    int
	MaxMemory       int64
	MaxNumberImages int
	ImageDirectory  string
}

type FileServer struct {
	Host        string
	CacheMaxAge string
}

type TagMeta struct {
	MaxTagLength int
}

var envs = []string{"local", "dev", "prod"}

func MustLoad() *Config {
	var configPath, env string
	flag.StringVar(&configPath, "config", "", "path to config file")
	flag.StringVar(&env, "env", "", "environment (local, dev, prod)")
	flag.Parse()

	if configPath == "" {
		panic("config file is empty")
	}

	if !slices.Contains(envs, env) {
		panic("env doesn't match one of the values: local, dev, prod")
	}

	viper.SetConfigFile(configPath)
	viper.SetConfigType("env")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		panic("error reading config file: " + err.Error())
	}

	cfg := Config{
		Env: env,
		HTTPServer: HTTPServer{
			Host:        viper.GetString("APP_HOST"),
			Port:        viper.GetString("APP_PORT"),
			Timeout:     viper.GetDuration("APP_TIMEOUT"),
			IdleTimeout: viper.GetDuration("APP_IDLE_TIMEOUT"),
		},
		Db: Db{
			Host:     viper.GetString("DB_HOST"),
			Port:     viper.GetString("DB_PORT"),
			User:     viper.GetString("DB_USER"),
			Password: viper.GetString("DB_PASSWORD"),
			Name:     viper.GetString("DB_NAME"),
			SSLMode:  viper.GetString("SSL_MODE"),
		},
		ImageMeta: ImageMeta{
			MaxImageSize:    viper.GetInt("MAX_IMAGE_SIZE"),
			MaxMemory:       viper.GetInt64("MAX_MEMORY"),
			MaxNumberImages: viper.GetInt("MAX_NUMBER_IMAGES"),
			ImageDirectory:  viper.GetString("IMAGE_DIRECTORY"),
		},
		FileServer: FileServer{
			Host:        viper.GetString("FILE_SERVER_HOST"),
			CacheMaxAge: viper.GetString("CACHE_MAX_AGE"),
		},
		TagMeta: TagMeta{
			MaxTagLength: viper.GetInt("MAX_TAG_LENGTH"),
		},
	}

	return &cfg
}
