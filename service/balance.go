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

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
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
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err := db.Transaction(func(tx *gorm.DB) error {
			var fromBalance models.Balance
			if err := tx.WithContext(dbCtx).
				Where("address = ?", from).First(&fromBalance).Error; err != nil {
				return err
			}
			if fromBalance.Balance.Cmp(amount) == -1 {
				return errors.New("insufficient balance")
			}

			result := tx.WithContext(dbCtx).Model(&fromBalance).
				Where("address = ? AND balance = ?", from, fromBalance.Balance).
				Update("balance", models.BigInt{Int: *big.NewInt(0).Sub(&fromBalance.Balance.Int, amount)})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return errors.New("concurrent modification detected")
			}

			var toBalance models.Balance
			err := tx.WithContext(dbCtx).
				Where("address = ?", to).First(&toBalance).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				toBalance = models.Balance{
					Address: to,
					Balance: models.BigInt{Int: *amount},
				}
				return tx.WithContext(dbCtx).Create(&toBalance).Error
			} else if err == nil {
				result := tx.WithContext(dbCtx).Model(&toBalance).
					Where("address = ? AND balance = ?", to, toBalance.Balance).
					Update("balance", models.BigInt{Int: *big.NewInt(0).Add(&toBalance.Balance.Int, amount)})
				if result.Error != nil {
					return result.Error
				}
				if result.RowsAffected == 0 {
					return errors.New("concurrent modification detected")
				}
				return nil
			} else {
				return err
			}
		})

		if err == nil {
			return nil
		}
		if err.Error() != "concurrent modification detected" {
			return err
		}
		time.Sleep(time.Millisecond * 100)
	}
	return errors.New("max retries exceeded")
}

func GetBalance(ctx context.Context, db *gorm.DB, address string) (*big.Int, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var balance models.Balance
	err := db.WithContext(dbCtx).Where("address = ?", address).First(&balance).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return big.NewInt(0), err
	}
	if err != nil {
		return nil, err
	}
	return &balance.Balance.Int, nil
}
