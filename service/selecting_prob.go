package service

import (
	"context"
	"crynux_relay/config"
	"crynux_relay/models"
	"crynux_relay/utils"
	"math/big"
	"sync"
	"time"

	"gorm.io/gorm"
)

var (
	globalMaxStaking  = &MaxStaking{staking: big.NewInt(0)}
	globalMaxQosScore = float64(TASK_SCORE_REWARDS[0])
)

func InitSelectingProb(ctx context.Context, db *gorm.DB) error {
	return RefreshMaxStaking(ctx, db)
}

func CalculateSelectingProb(staking, maxStaking *big.Int, qosScore, maxQosScore float64) (float64, float64, float64) {
	stakingProb := CalculateStakingScore(staking, maxStaking)
	qosProb := CalculateQosScore(qosScore, maxQosScore)
	if qosProb == 0 {
		qosProb = 0.5
	}
	var prob float64
	if stakingProb == 0 || qosProb == 0 {
		prob = 0
	} else {
		prob = stakingProb * qosProb / (stakingProb + qosProb)
	}
	return stakingProb, qosProb, prob
}

func CalculateStakingScore(staking, maxStaking *big.Int) float64 {
	if maxStaking.Sign() == 0 {
		return 0
	}
	stakingProb, _ := big.NewFloat(0).Quo(big.NewFloat(0).SetInt(staking), big.NewFloat(0).SetInt(maxStaking)).Float64()
	return stakingProb
}

func CalculateQosScore(qosScore, maxQosScore float64) float64 {
	if maxQosScore == 0 {
		return 0
	}
	return qosScore / maxQosScore
}

func GetMaxStaking() *big.Int {
	return globalMaxStaking.get()
}

func GetMaxQosScore() float64 {
	return globalMaxQosScore
}

func UpdateMaxStaking(staking *big.Int) {
	globalMaxStaking.update(staking)
}

func RefreshMaxStaking(ctx context.Context, db *gorm.DB) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	type result struct {
		StakeAmount models.BigInt `json:"stake_amount"`
	}

	var res result
	if err := db.WithContext(dbCtx).Model(&models.Node{}).Select("MAX(CAST(stake_amount as DECIMAL(65,0))) as stake_amount").First(&res).Error; err != nil {
		return err
	}

	if res.StakeAmount.Int.Sign() > 0 {
		globalMaxStaking.update(&res.StakeAmount.Int)
	} else {
		appConfig := config.GetConfig()
		globalMaxStaking.update(utils.EtherToWei(big.NewInt(int64(appConfig.Task.StakeAmount))))
	}

	return nil
}


type MaxStaking struct {
	sync.RWMutex
	staking *big.Int
}

func (g *MaxStaking) update(staking *big.Int) {
	g.RLock()
	if staking.Cmp(g.staking) > 0 {
		g.RUnlock()
		g.Lock()
		g.staking.Set(staking)
		g.Unlock()
	} else {
		g.RUnlock()
	}
}

func (g *MaxStaking) get() *big.Int {
	g.RLock()
	defer g.RUnlock()
	return g.staking
}