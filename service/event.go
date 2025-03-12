package service

import (
	"context"
	"crynux_relay/models"

	"gorm.io/gorm"
)

func emitEvent(ctx context.Context, db *gorm.DB, e models.ToEventType) error {
	event, err := e.ToEvent()
	if err != nil {
		return err
	}

	if err := event.Save(ctx, db); err != nil {
		return err
	}
	return nil
}
