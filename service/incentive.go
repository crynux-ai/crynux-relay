package service

import (
	"context"
	"crynux_relay/models"
	"time"

	"gorm.io/gorm"
)

func addNodeIncentive(ctx context.Context, db *gorm.DB, nodeAddress string, incentive float64) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	t := time.Now().UTC().Truncate(24 * time.Hour)
	nodeIncentive := models.NodeIncentive{Time: t, NodeAddress: nodeAddress}
	if err := db.WithContext(ctx).Model(&nodeIncentive).Where(&nodeIncentive).First(&nodeIncentive).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
	}
	if nodeIncentive.ID > 0 {
		nodeIncentive.Incentive += incentive
		nodeIncentive.TaskCount += 1
		if err := db.WithContext(dbCtx).Save(&nodeIncentive).Error; err != nil {
			return err
		}
	} else {
		nodeIncentive.Incentive = incentive
		nodeIncentive.TaskCount = 1
		if err := db.WithContext(dbCtx).Create(&nodeIncentive).Error; err != nil {
			return err
		}
	}
	return nil
}
