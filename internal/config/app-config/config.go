package app_config

import (
	"flag"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
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
	MaxImageSize        int
	MaxMemory           int64
	PostMaxNumberImages int
	ImageDirectory      string
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

	if cfg.MaxImageSize, err = strconv.Atoi(os.Getenv("MAX_IMAGE_SIZE")); err != nil {
		panic("error conversion max image size")
	}

	if cfg.ImageMeta.MaxMemory, err = strconv.ParseInt(os.Getenv("MAX_MEMORY"), 10, 64); err != nil {
		panic("error parsing max memory")
	}
	if cfg.ImageMeta.PostMaxNumberImages, err = strconv.Atoi(os.Getenv("POST_MAX_NUMBER_IMAGES")); err != nil {
		panic("error conversion post max number images")
	}

	cfg.ImageMeta.ImageDirectory = os.Getenv("IMAGE_DIRECTORY")

	cfg.FileServer.Host = os.Getenv("FILE_SERVER_HOST")
	cfg.FileServer.CacheMaxAge = os.Getenv("CACHE_MAX_AGE")

	if cfg.TagMeta.MaxTagLength, err = strconv.Atoi(os.Getenv("MAX_TAG_LENGTH")); err != nil {
		panic("error conversion max tag length")
	}

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
		panic("error parsing idle timeout duration")
	}

}
