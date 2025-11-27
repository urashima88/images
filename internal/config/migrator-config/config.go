package migrator_config

import (
	"flag"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Db
	MigrationsPath  string
	MigrationsTable string
}

type Db struct {
	Host         string
	Port         string
	User         string
	Password     string
	Name         string
	SSLMode      string
	ForceVersion int
	Steps        int
	DropFlag     bool
	VersionFlag  bool
	DownFlag     bool
}

func MustLoad() *Config {
	var (
		path, migrationsPath, migrationsTable string
		forceVersion, steps                   int
		dropFlag, versionFlag, downFlag       bool
	)

	flag.StringVar(&path, "config", "", "path to config file")
	flag.StringVar(&migrationsPath, "migrations-path", "", "path to migrations")
	flag.StringVar(&migrationsTable, "migrations-table", "migrations", "name of migrations table")

	flag.IntVar(&forceVersion, "force", -1, "force set database version")
	flag.IntVar(&steps, "steps", 1, "number of migrations to rollback (with -down)")
	flag.BoolVar(&dropFlag, "drop", false, "drop everything and start fresh")
	flag.BoolVar(&versionFlag, "version", false, "show current migration version")
	flag.BoolVar(&downFlag, "down", false, "rollback migrations")

	flag.Parse()

	err := godotenv.Load(path)
	if err != nil {
		panic("error loading .env: " + err.Error())
	}

	var cfg Config

	cfg.Db.Host = os.Getenv("DB_HOST")
	cfg.Db.Port = os.Getenv("DB_PORT")
	cfg.Db.User = os.Getenv("DB_USER")
	cfg.Db.Password = os.Getenv("DB_PASSWORD")
	cfg.Db.Name = os.Getenv("DB_NAME")
	cfg.Db.SSLMode = os.Getenv("SSL_MODE")

	if migrationsPath == "" {
		panic("migrations-path is required")
	}

	cfg.MigrationsPath = migrationsPath
	cfg.MigrationsTable = migrationsTable

	cfg.ForceVersion = forceVersion
	cfg.Steps = steps
	cfg.DropFlag = dropFlag
	cfg.VersionFlag = versionFlag
	cfg.DownFlag = downFlag

	return &cfg
}
