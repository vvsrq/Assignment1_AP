package db

import (
	_ "github.com/lib/pq"
)

//func Connect() (*sql.DB, error) {
//	cfg := config.LoadConfig()
//
//	db, err := sql.Open("postgres", cfg.DatabaseURL)
//	if err != nil {
//		// НЕ ВЫЗЫВАЕМ Fatal, ВОЗВРАЩАЕМ ошибку
//		return nil, fmt.Errorf("failed to open database connection: %w", err)
//	}
//
//	err = db.Ping()
//	if err != nil {
//		db.Close()
//		return nil, fmt.Errorf("database ping failed: %w", err)
//	}
//
//	return db, nil
//}
