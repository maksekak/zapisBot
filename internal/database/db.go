package database

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"sort"
	"strings"

	_ "github.com/lib/pq"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB — обёртка над *sql.DB.
type DB struct {
	*sql.DB
}

// New подключается к PostgreSQL и применяет встроенные миграции.
func New(databaseURL string) (*DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть БД: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("не удалось подключиться к БД: %w", err)
	}

	log.Println("Подключение к PostgreSQL установлено")

	if err := applyMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("ошибка применения миграций: %w", err)
	}

	return &DB{db}, nil
}

// applyMigrations применяет встроенные SQL-миграции в алфавитном порядке.
func applyMigrations(db *sql.DB) error {
	// Создаём таблицу для отслеживания применённых миграций
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("создание таблицы миграций: %w", err)
	}

	// Читаем все .sql файлы из встроенной файловой системы
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("чтение встроенных миграций: %w", err)
	}

	var sqlFiles []string
	for _, f := range entries {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".sql") {
			sqlFiles = append(sqlFiles, f.Name())
		}
	}
	sort.Strings(sqlFiles)

	for _, file := range sqlFiles {
		// Проверяем, применена ли миграция
		var applied bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", file).Scan(&applied)
		if err != nil {
			return fmt.Errorf("проверка миграции %s: %w", file, err)
		}
		if applied {
			continue
		}

		// Читаем и выполняем SQL
		content, err := migrationsFS.ReadFile("migrations/" + file)
		if err != nil {
			return fmt.Errorf("чтение миграции %s: %w", file, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("начало транзакции для миграции %s: %w", file, err)
		}

		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("выполнение миграции %s: %w", file, err)
		}

		if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", file); err != nil {
			tx.Rollback()
			return fmt.Errorf("запись миграции %s: %w", file, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("коммит миграции %s: %w", file, err)
		}

		log.Printf("Миграция %s применена", file)
	}

	return nil
}

// Close закрывает соединение с БД.
func (d *DB) Close() error {
	return d.DB.Close()
}