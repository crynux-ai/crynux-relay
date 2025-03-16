package service

import (
	"context"
	"crynux_relay/config"
	"crynux_relay/models"
	"crynux_relay/utils"
	"errors"
	"math/big"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func CreateGenesisAccount(ctx context.Context, db *gorm.DB) error {
	appConfig := config.GetConfig()
	address := appConfig.Blockchain.Account.Address
	amount := utils.EtherToWei(big.NewInt(int64(appConfig.Blockchain.Account.GenesisTokenAmount)))

	dbCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	return db.WithContext(dbCtx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "address"}},
		DoNothing: true,
	}).Create(&models.Balance{
		Address: address,
		Balance: models.BigInt{Int: *amount},
	}).Error
}

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
		
		if err := tx.WithContext(dbCtx).Model(&fromBalance).Update("balance", models.BigInt{Int: *big.NewInt(0).Sub(&fromBalance.Balance.Int, amount)}).Error; err != nil {
			return err
		}

		var toBalance models.Balance
		err := tx.WithContext(dbCtx).Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("address = ?", to).First(&toBalance).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			toBalance = models.Balance{
				Address: to,
				Balance: models.BigInt{Int: *amount},
			}
			return tx.WithContext(dbCtx).Create(&toBalance).Error
		} else if err == nil {
			return tx.WithContext(dbCtx).Model(&toBalance).Update("balance", models.BigInt{Int: *big.NewInt(0).Add(&toBalance.Balance.Int, amount)}).Error
		} else {
			return err
		}
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
