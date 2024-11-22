package blockchain

import (
	"crynux_relay/config"

	"golang.org/x/time/rate"
)

var limiter *rate.Limiter

func init() {
	appConfig := config.GetConfig()
	limiter = rate.NewLimiter(rate.Limit(appConfig.Blockchain.RPS), int(appConfig.Blockchain.RPS))
}
