package database

import (
	"database/sql"
	"embed"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var EmbedMigrations embed.FS

// Config is the configuration for the database.
type Config struct {
	User     string `envconfig:"POSTGRES_USER" required:"true" default:"postgres"`
	Password string `envconfig:"POSTGRES_PASSWORD" required:"true" default:"postgres"`
	DBName   string `envconfig:"POSTGRES_DB" required:"true" default:"fileserver"`
	Host     string `envconfig:"POSTGRES_HOST" required:"true" default:"localhost"`
	Port     string `envconfig:"POSTGRES_PORT" required:"true" default:"5432"`
}

// NewDBConnection creates a new database connection.
func NewDBConnection(cfg Config) (*sql.DB, error) {
	connStr := fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable password=%s port=%s",
		cfg.Host, cfg.User, cfg.DBName, cfg.Password, cfg.Port)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("error pinging database: %w", err)
	}
	return db, nil
}

// MigrateDB migrates the database.
func MigrateDB(db *sql.DB, migrationsName string, fs embed.FS) error {
	goose.SetBaseFS(fs)
	if err := goose.Up(db, migrationsName); err != nil {
		return fmt.Errorf("error migrating database: %w", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("error setting dialect: %w", err)
	}

	return nil
}
