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
var changedBalances map[string]struct{} = make(map[string]struct{})


func InitBalanceCache(ctx context.Context, db *gorm.DB) error {
	var events []models.TransferEvent
	if err := db.Where("status = ?", 0).Find(&events).Error; err != nil {
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
	go func() {
		ticker := time.NewTicker(5 * time.Second)
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
	}()
}

func syncBalancesToDB(ctx context.Context, db *gorm.DB) error {
	balanceCache.mu.RLock()
	defer balanceCache.mu.RUnlock()

	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return db.WithContext(dbCtx).Transaction(func(tx *gorm.DB) error {
		for address := range changedBalances {
			balance := balanceCache.balances[address]
			tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "address"}},
				DoUpdates: clause.Assignments(map[string]interface{}{"balance": models.BigInt{Int: *balance}}),
			}).Create(&models.Balance{
				Address: address,
				Balance: models.BigInt{Int: *balance},
			})
		}

		if err := tx.Model(&models.TransferEvent{}).
			Where("status = ?", 0).
			Update("status", 1).Error; err != nil {
			return err
		}

		return nil
	})
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

	changedBalances[from] = struct{}{}
	changedBalances[to] = struct{}{}

	return nil
}

func GetBalance(ctx context.Context, db *gorm.DB, address string) (*big.Int, error) {
	return getBalanceFromCache(ctx, db, address)
}
