package service

import (
	"context"
	"crynux_relay/models"
	"errors"
	"math/big"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func Transfer(ctx context.Context, db *gorm.DB, from, to string, amount *big.Int) error {
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return db.Transaction(func(tx *gorm.DB) error {
		var fromBalance models.Balance
		if err := tx.WithContext(dbCtx).Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("address = ?", from).First(&fromBalance).Error; err != nil {
			return err
		}
		if fromBalance.Balance.Cmp(amount) == -1 {
			return errors.New("insufficient balance")
		}

		if err := tx.WithContext(dbCtx).Model(&fromBalance).Update("balance", big.NewInt(0).Sub(&fromBalance.Balance.Int, amount)).Error; err != nil {
			return err
		}

		var toBalance models.Balance
		if err := tx.WithContext(dbCtx).Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("address = ?", to).First(&toBalance).Error; err != nil {
			return err
		}
		if err := tx.WithContext(dbCtx).Model(&toBalance).Update("balance", big.NewInt(0).Add(&toBalance.Balance.Int, amount)).Error; err != nil {
			return err
		}

		return nil
	})
}

func GetBalance(ctx context.Context, db *gorm.DB, address string) (*big.Int, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var balance models.Balance
	if err := db.WithContext(dbCtx).Where("address = ?", address).First(&balance).Error; err != nil {
		return nil, err
	}
	return &balance.Balance.Int, nil
}