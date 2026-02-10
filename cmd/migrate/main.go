package main

import (
	"log"
	"user_crud_jwt/internal/pkg/config"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	config.LoadConfig()
	cfg := config.GlobalConfig.Database
	dsn := "postgres://" + cfg.User + ":" + cfg.Password + "@" + cfg.Host + ":" + cfg.Port + "/" + cfg.DBName + "?sslmode=" + cfg.SSLMode

	m, err := migrate.New(
		"file://migrations",
		dsn,
	)
	if err != nil {
		log.Fatal(err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		// 如果数据库处于 dirty 状态，尝试强制修复到上一版本，然后重试
		if err.Error() == "Dirty database version 1. Fix and force version." {
			log.Println("Database is dirty, forcing version 1...")
			if err := m.Force(1); err != nil {
				log.Fatal("Failed to force version:", err)
			}
			// 重试 Up
			if err := m.Up(); err != nil && err != migrate.ErrNoChange {
				log.Fatal(err)
			}
		} else {
			log.Fatal(err)
		}
	}

	log.Println("Migration successful")
}
