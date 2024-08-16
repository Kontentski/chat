package storage

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
)

var DB *pgxpool.Pool

func Init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	dsn := os.Getenv("DATABASE_URL")

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("Unable to parse database URL: %v\n", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}

	DB = pool
}

func RunMigrations() {
	dsn := os.Getenv("DATABASE_URL")
	db, err := goose.OpenDBWithDriver("pgx", dsn)
	if err != nil {
		log.Fatalf("Could not open db: %v\n", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("Could not set dialect: %v\n", err)
	}

	if err := goose.Up(db, "./migrations"); err != nil {
		log.Fatalf("Could not run up migrations: %v\n", err)
	}
}