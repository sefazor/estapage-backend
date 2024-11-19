package database

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB(dsn string) {
	var err error

	// PostgreSQL spesifik konfigürasyon
	pgConfig := postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true, // Prepared statement sorununu çözmek için
	}

	// GORM konfigürasyonu
	gormConfig := &gorm.Config{
		Logger:      logger.Default.LogMode(logger.Error),
		PrepareStmt: false, // Statement cache'i devre dışı bırak
	}

	DB, err = gorm.Open(postgres.New(pgConfig), gormConfig)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Connection pool ayarları
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatal("Failed to get database instance:", err)
	}

	// Connection pool limitleri
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	log.Println("Database connected successfully!")
}

func GetDB() *gorm.DB {
	return DB
}

func MigrateDatabase(models ...interface{}) error {
	for _, model := range models {
		if !DB.Migrator().HasTable(model) {
			if err := DB.Migrator().CreateTable(model); err != nil {
				return err
			}
			log.Printf("Created table for %T\n", model)
		} else {
			if err := DB.Migrator().AutoMigrate(model); err != nil {
				return err
			}
			log.Printf("Updated table for %T\n", model)
		}
	}
	return nil
}
