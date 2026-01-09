package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"go-api-starter/core/constants"
	"go-api-starter/core/logger"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type IDatabase interface {
	ExecContext(ctx context.Context, query string, args ...any) error
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	NamedQueryContext(ctx context.Context, query string, arg any) (*sqlx.Rows, error)
	NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error)
	SQLx() *sqlx.DB
}

type Database struct {
	db   *sql.DB
	sqlx *sqlx.DB
}

type DatabaseConfig struct {
	Host                   string
	Port                   int
	User                   string
	Password               string
	DBName                 string
	MaxOpenConns           int
	MaxIdleConns           int
	ConnMaxLifetime        int    // in minutes
	SSLMode                string // disable, require, verify-ca, verify-full
	ConnectTimeout         int    // in seconds
	StatementTimeout       int    // in seconds
	IdleInTxSessionTimeout int    // in seconds
}

var (
	instance *Database
)

func GetDB() IDatabase {
	return instance
}

func InitDB(config DatabaseConfig) (Database, error) {
	logger.Info("Initializing database...")
	var db Database
	var err error

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, constants.DatabaseSSLMode)

	sqlxDB, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		return Database{}, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB := sqlxDB.DB
	sqlDB.SetMaxOpenConns(constants.DatabaseMaxOpenConns)
	sqlDB.SetMaxIdleConns(constants.DatabaseMaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(constants.DatabaseConnMaxLifetime) * time.Minute)

	if err = sqlDB.Ping(); err != nil {
		logger.Error("Failed to ping database", "error", err)
		return Database{}, fmt.Errorf("failed to ping database: %w", err)
	}

	db = Database{
		db:   sqlDB,
		sqlx: sqlxDB,
	}

	logger.Info("Database initialized successfully",
		"host", config.Host,
		"port", config.Port,
		"database", config.DBName,
		"user", config.User,
		"maxOpenConns", constants.DatabaseMaxOpenConns,
		"maxIdleConns", constants.DatabaseMaxIdleConns,
		"connMaxLifetime", constants.DatabaseConnMaxLifetime,
	)

	// Check if users table exists
	var tableExists bool
	checkTableQuery := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'users'
		)
	`
	err = sqlDB.QueryRow(checkTableQuery).Scan(&tableExists)
	if err != nil {
		logger.Error("Failed to check users table", "error", err)
	} else {
		if tableExists {
			logger.Info("Users table exists")
			// Check columns in users table
			var columns []string
			checkColumnsQuery := `
				SELECT column_name 
				FROM information_schema.columns 
				WHERE table_schema = 'public' 
				AND table_name = 'users'
				ORDER BY ordinal_position
			`
			rows, err := sqlDB.Query(checkColumnsQuery)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var colName string
					if err := rows.Scan(&colName); err == nil {
						columns = append(columns, colName)
					}
				}
				logger.Info("Users table columns", "columns", columns)
			}
		} else {
			logger.Warn("Users table does not exist")
		}
	}

	return db, nil
}

func (d *Database) ExecContext(ctx context.Context, query string, args ...any) error {
	_, err := d.sqlx.ExecContext(ctx, query, args...)
	return err
}

func (d *Database) GetContext(ctx context.Context, dest any, query string, args ...any) error {
	return d.sqlx.GetContext(ctx, dest, query, args...)
}

func (d *Database) SelectContext(ctx context.Context, dest any, query string, args ...any) error {
	return d.sqlx.SelectContext(ctx, dest, query, args...)
}

func (d *Database) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return d.db.QueryRowContext(ctx, query, args...)
}

func (d *Database) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return d.db.QueryContext(ctx, query, args...)
}

func (d *Database) NamedQueryContext(ctx context.Context, query string, arg any) (*sqlx.Rows, error) {
	return d.sqlx.NamedQueryContext(ctx, query, arg)
}

func (d *Database) NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error) {
	return d.sqlx.NamedExecContext(ctx, query, arg)
}

func (d *Database) SQLx() *sqlx.DB {
	return d.sqlx
}
