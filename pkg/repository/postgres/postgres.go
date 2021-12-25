package postgres

import (
	"fmt"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx"
	"os"
	"strconv"
)

type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	DBName   string
	SSLMode  string
}

func (c *Config) GetDataSource() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.Username, c.Password, c.DBName, c.SSLMode)
}

func (c *Config) GetDriverConfig() *stdlib.DriverConfig {
	d := &stdlib.DriverConfig{
		ConnConfig: pgx.ConnConfig{
			RuntimeParams: map[string]string{
				"standard_conforming_strings": "on",
			},
			PreferSimpleProtocol: true,
		},
	}
	stdlib.RegisterDriverConfig(d)
	return d
}

func NewPostgresDb(cfg *Config) (*sqlx.DB, error) {
	driverConfig := cfg.GetDriverConfig()
	daaSource := cfg.GetDataSource()

	db, err := sqlx.Connect("pgx", driverConfig.ConnectionString(daaSource))

	if err != nil {
		return nil, err
	}

	maxOpenConnection, _ := strconv.Atoi(os.Getenv("DB_MAX_OPEN_CONNECTIONS"))
	db.SetMaxOpenConns(maxOpenConnection)

	return db, nil
}
