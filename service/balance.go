package service

import (
	"context"
	"crynux_relay/config"
	"crynux_relay/models"
	"crynux_relay/utils"
	"errors"
	"log"
	"math/big"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BalanceCache struct {
	balances map[string]*big.Int
	mu       sync.RWMutex
}

var balanceCache = &BalanceCache{
	balances: make(map[string]*big.Int),
}

func InitBalanceCache(ctx context.Context, db *gorm.DB) error {
	var events []models.TransferEvent
	if err := db.Where("status = ?", models.TransferEventStatusPending).Find(&events).Error; err != nil {
		return err
	}

	for _, event := range events {
		fromAmount, err := getBalanceFromCache(ctx, db, event.FromAddress)
		if err != nil {
			return err
		}
		toAmount, err := getBalanceFromCache(ctx, db, event.ToAddress)
		if err != nil {
			return err
		}

		fromAmount.Sub(fromAmount, &event.Amount.Int)
		toAmount.Add(toAmount, &event.Amount.Int)
	}
	return nil
}

func getPendingTransferEvents(ctx context.Context, db *gorm.DB, limit, offset int) ([]models.TransferEvent, error) {
	var events []models.TransferEvent
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err := db.WithContext(dbCtx).Where("status = ?", models.TransferEventStatusPending).Order("id").Limit(limit).Find(&events).Error
	if err != nil {
		return nil, err
	}
	return events, nil
}

func getBalanceFromCache(ctx context.Context, db *gorm.DB, address string) (*big.Int, error) {
	balanceCache.mu.RLock()
	balance, exists := balanceCache.balances[address]
	balanceCache.mu.RUnlock()

	if exists {
		return balance, nil
	}

	var dbBalance models.Balance
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.WithContext(dbCtx).Where("address = ?", address).Attrs(models.Balance{Balance: models.BigInt{Int: *big.NewInt(0)}}).FirstOrInit(&dbBalance).Error; err != nil {
		return nil, err
	}

	balanceCache.mu.Lock()
	balanceCache.balances[address] = &dbBalance.Balance.Int
	balanceCache.mu.Unlock()

	return &dbBalance.Balance.Int, nil
}

func StartBalanceSync(ctx context.Context, db *gorm.DB) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := syncBalancesToDB(ctx, db); err != nil {
				log.Printf("Failed to sync balances: %v", err)
			}
		}
	}
}

func processTransferEvent(ctx context.Context, db *gorm.DB, event *models.TransferEvent) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return db.WithContext(dbCtx).Transaction(func(tx *gorm.DB) error {
		fromBalance := &models.Balance{Address: event.FromAddress}
		toBalance := &models.Balance{Address: event.ToAddress}
		if err := tx.Model(fromBalance).Where(fromBalance).First(fromBalance).Error; err != nil {
			return err
		}
		fromBalance.Balance.Sub(&fromBalance.Balance.Int, &event.Amount.Int)

		if fromBalance.Balance.Int.Cmp(big.NewInt(0)) < 0 {
			return errors.New("insufficient balance")
		}
		if err := tx.Save(fromBalance).Error; err != nil {
			return err
		}

		if err := tx.Model(toBalance).Where(toBalance).Attrs(models.Balance{Balance: models.BigInt{Int: *big.NewInt(0)}}).FirstOrInit(toBalance).Error; err != nil {
			return err
		}
		toBalance.Balance.Add(&toBalance.Balance.Int, &event.Amount.Int)
		if err := tx.Save(toBalance).Error; err != nil {
			return err
		}

		if err := tx.Model(event).Update("status", models.TransferEventStatusProcessed).Error; err != nil {
			return err
		}

		return nil
	})
}

func syncBalancesToDB(ctx context.Context, db *gorm.DB) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			err := func() error {
				for {
					events, err := getPendingTransferEvents(ctx, db, 1000, 0)
					if err != nil {
						return err
					}

					if len(events) == 0 {
						break
					}

					for _, event := range events {
						if err := processTransferEvent(ctx, db, &event); err != nil {
							return err
						}
					}
				}
				return nil
			}()

			if err != nil {
				log.Printf("Failed to sync balances: %v", err)
			}

			// Wait for 2 seconds before next iteration
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(2 * time.Second):
				continue
			}
		}
	}
}

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
	fromBalance, err := getBalanceFromCache(ctx, db, from)
	if err != nil {
		return err
	}
	if fromBalance.Cmp(amount) < 0 {
		return errors.New("insufficient balance")
	}
	toBalance, err := getBalanceFromCache(ctx, db, to)
	if err != nil {
		return err
	}

	event := &models.TransferEvent{
		FromAddress: from,
		ToAddress:   to,
		Amount:      models.BigInt{Int: *amount},
		CreatedAt:   time.Now(),
		Status:      models.TransferEventStatusPending,
	}

	if err := db.Create(event).Error; err != nil {
		return err
	}

	balanceCache.mu.Lock()
	defer balanceCache.mu.Unlock()

	fromBalance.Sub(fromBalance, amount)
	toBalance.Add(toBalance, amount)

	return nil
}

func GetBalance(ctx context.Context, db *gorm.DB, address string) (*big.Int, error) {
	return getBalanceFromCache(ctx, db, address)
}
