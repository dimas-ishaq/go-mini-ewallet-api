package database

import (
	"database/sql"
	"embed"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// Menyematkan folder migrations ke dalam binary Go
//
//go:embed migrations
var migrationFS embed.FS

func RunMigrations(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("gagal membuat driver migrasi: %w", err)
	}

	sourceDriver, err := iofs.New(migrationFS, "migrations")
	if err != nil {
		return fmt.Errorf("gagal membaca folder migrasi: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", driver)
	if err != nil {
		return fmt.Errorf("gagal inisialisasi instance migrasi: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("gagal menjalankan migrasi: %w", err)
	}

	log.Println("Migrasi database berhasil dijalankan!")
	return nil
}
