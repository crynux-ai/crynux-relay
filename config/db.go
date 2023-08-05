package config

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"time"
)

var db *gorm.DB

func InitDB(appConfig *AppConfig) error {

	if appConfig.Environment == EnvTest && appConfig.Db.Driver == "" {
		return nil
	}

	var dial gorm.Dialector

	if appConfig.Db.Driver == "mysql" {
		dial = mysql.Open(appConfig.Db.ConnectionString)
	} else if appConfig.Db.Driver == "postgres" {
		dial = postgres.Open(appConfig.Db.ConnectionString)
	} else if appConfig.Db.Driver == "sqlite" {
		dial = sqlite.Open(appConfig.Db.ConnectionString)
	} else {
		return errors.New("DB not supported")
	}

	db, err := gorm.Open(dial, &gorm.Config{})
	if err != nil {
		log.Error("InitDB Failed:" + err.Error())
		return err
	}

	sqlDB, err := db.DB()

	if err != nil {
		log.Error("InitDB Failed:" + err.Error())
		return err
	}

	sqlDB.SetMaxIdleConns(20)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(5 * time.Second)

	return nil
}

func GetDB() *gorm.DB {
	return db
}
