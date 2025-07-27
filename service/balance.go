package service

import (
	"context"
	"crynux_relay/config"
	"crynux_relay/models"
	"crynux_relay/utils"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

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

func getPendingTransferEvents(ctx context.Context, db *gorm.DB, limit int) ([]models.TransferEvent, error) {
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

	balanceCache.mu.Lock()
	defer balanceCache.mu.Unlock()

	if balance, exists := balanceCache.balances[address]; exists {
		return balance, nil
	}

	var dbBalance models.Balance
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.WithContext(dbCtx).Where("address = ?", address).Attrs(models.Balance{Balance: models.BigInt{Int: *big.NewInt(0)}}).FirstOrInit(&dbBalance).Error; err != nil {
		return nil, err
	}

	balanceCache.balances[address] = &dbBalance.Balance.Int

	return balanceCache.balances[address], nil
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
				log.Errorf("Failed to sync balances: %v", err)
			}
		}
	}
}

func mergeTransferEvents(events []models.TransferEvent) map[string]*big.Int {
	mergedEvents := make(map[string]*big.Int)
	for _, event := range events {
		if _, exists := mergedEvents[event.FromAddress]; !exists {
			mergedEvents[event.FromAddress] = big.NewInt(0).Sub(big.NewInt(0), &event.Amount.Int)
		} else {
			mergedEvents[event.FromAddress].Sub(mergedEvents[event.FromAddress], &event.Amount.Int)
		}
		if _, exists := mergedEvents[event.ToAddress]; !exists {
			mergedEvents[event.ToAddress] = big.NewInt(0).Set(&event.Amount.Int)
		} else {
			mergedEvents[event.ToAddress].Add(mergedEvents[event.ToAddress], &event.Amount.Int)
		}
	}
	return mergedEvents
}

func processPendingTransferEvents(ctx context.Context, db *gorm.DB, events []models.TransferEvent) error {
	mergedEvents := mergeTransferEvents(events)

	var eventIDs []uint
	for _, event := range events {
		eventIDs = append(eventIDs, event.ID)
	}

	var addresses []string
	for address := range mergedEvents {
		addresses = append(addresses, address)
	}

	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return db.WithContext(dbCtx).Transaction(func(tx *gorm.DB) error {
		var existedBalances []models.Balance
		if err := tx.Model(&models.Balance{}).Where("address IN (?)", addresses).Find(&existedBalances).Error; err != nil {
			return err
		}

		existedBalancesMap := make(map[string]*models.Balance)
		for _, balance := range existedBalances {
			existedBalancesMap[balance.Address] = &balance
		}

		var newBalances []models.Balance
		for address, amount := range mergedEvents {
			if balance, exists := existedBalancesMap[address]; !exists {
				newBalances = append(newBalances, models.Balance{Address: address, Balance: models.BigInt{Int: *amount}})
			} else {
				balance.Balance = models.BigInt{Int: *new(big.Int).Add(&balance.Balance.Int, amount)}
			}
		}

		if len(newBalances) > 0 {
			if err := tx.CreateInBatches(&newBalances, 100).Error; err != nil {
				return err
			}
		}

		if len(existedBalancesMap) > 0 {
			var cases string
			for _, balance := range existedBalancesMap {
				cases += fmt.Sprintf(" WHEN address = '%s' THEN '%s'", balance.Address, balance.Balance.String())
			}
			if err := tx.Model(&models.Balance{}).Where("address IN (?)", addresses).
				Update("balance", gorm.Expr("CASE"+cases+" END")).Error; err != nil {
				return err
			}
		}

		if err := tx.Model(&models.TransferEvent{}).Where("id IN (?)", eventIDs).Where("status = ?", models.TransferEventStatusPending).Update("status", models.TransferEventStatusProcessed).Error; err != nil {
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
					events, err := getPendingTransferEvents(ctx, db, 100)
					if err != nil {
						return err
					}

					if len(events) == 0 {
						break
					}

					if err := processPendingTransferEvents(ctx, db, events); err != nil {
						return err
					}
				}
				return nil
			}()

			if err != nil {
				log.Errorf("Failed to sync balances: %v", err)
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
		Balance: models.BigInt{Int: *new(big.Int).Set(amount)},
	}).Error
}

func Transfer(ctx context.Context, db *gorm.DB, from, to string, amount *big.Int) (func (), error) {
	fromBalance, err := getBalanceFromCache(ctx, db, from)
	if err != nil {
		return nil, err
	}
	if fromBalance.Cmp(amount) < 0 {
		return nil, errors.New("insufficient balance")
	}
	toBalance, err := getBalanceFromCache(ctx, db, to)
	if err != nil {
		return nil, err
	}

	event := &models.TransferEvent{
		FromAddress: from,
		ToAddress:   to,
		Amount:      models.BigInt{Int: *new(big.Int).Set(amount)},
		CreatedAt:   time.Now(),
		Status:      models.TransferEventStatusPending,
	}

	if err := db.Create(event).Error; err != nil {
		return nil, err
	}

	amountCopy := new(big.Int).Set(amount)
	commitFunc := func() {
		balanceCache.mu.Lock()
		defer balanceCache.mu.Unlock()
		fromBalance.Sub(fromBalance, amountCopy)
		toBalance.Add(toBalance, amountCopy)
	}

	return commitFunc, nil
}

func GetBalance(ctx context.Context, db *gorm.DB, address string) (*big.Int, error) {
	return getBalanceFromCache(ctx, db, address)
}
