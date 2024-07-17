package migrate

import (
	"crynux_relay/migrate/migrations"

	"github.com/go-gormigrate/gormigrate/v2"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var migrationScripts []*gormigrate.Gormigrate

func Migrate() error {
	for _, migrationScript := range migrationScripts {
		if err := migrationScript.Migrate(); err != nil {
			log.Errorf("Migrate: %v", err)
			return err
		}
	}

	return nil
}

func Rollback() error {
	lastMigration := migrationScripts[len(migrationScripts)-1]

	if err := lastMigration.RollbackLast(); err != nil {
		log.Errorf("Migrate: %v", err)
		return err
	}

	return nil
}

func InitMigration(db *gorm.DB) {
	db.Set("gorm:table_options", "CHARSET=utf8mb4")

	// Add new migrations here
	migrationScripts = append(migrationScripts, migrations.M20230810(db))
	migrationScripts = append(migrationScripts, migrations.M20230824(db))
	migrationScripts = append(migrationScripts, migrations.M20240115(db))
	migrationScripts = append(migrationScripts, migrations.M20240518(db))
	migrationScripts = append(migrationScripts, migrations.M20240522(db))
	migrationScripts = append(migrationScripts, migrations.M20240530(db))
	migrationScripts = append(migrationScripts, migrations.M20240613(db))
	migrationScripts = append(migrationScripts, migrations.M20240717(db))
}
