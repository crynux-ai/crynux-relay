package config

import (
	"crypto/ecdsa"
	"errors"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/viper"
	"strings"
)

var appConfig *AppConfig

// InitConfig Init is an exported method that takes the config from the config file
// and unmarshal it into AppConfig struct
func InitConfig(configPath string) error {
	v := viper.New()
	v.SetConfigType("yml")
	v.SetConfigName("config")

	if configPath != "" {
		v.AddConfigPath(configPath)
	} else {
		v.AddConfigPath("/app/config")
		v.AddConfigPath("config")
	}

	if err := v.ReadInConfig(); err != nil {
		return err
	}

	appConfig = &AppConfig{}

	if err := v.Unmarshal(appConfig); err != nil {
		return err
	}

	if appConfig.Environment == EnvTest {
		appConfig.Blockchain.Account.PrivateKey = GetTestPrivateKey()
	} else {
		// Load hard-coded private key
		appConfig.Blockchain.Account.PrivateKey = GetPrivateKey(appConfig.Blockchain.Account.PrivateKeyFile)
	}
	if err := checkBlockchainAccount(); err != nil {
		return err
	}

	return nil
}

func checkBlockchainAccount() error {

	if appConfig.Blockchain.Account.PrivateKey == "" {
		return errors.New("blockchain account private key not set")
	}

	if appConfig.Blockchain.Account.Address == "" {
		return errors.New("blockchain account address not set")
	}

	var pk string
	if strings.HasPrefix(appConfig.Blockchain.Account.PrivateKey, "0x") {
		pk = appConfig.Blockchain.Account.PrivateKey[2:]
	} else {
		pk = appConfig.Blockchain.Account.PrivateKey
	}

	// Check private key and address
	privateKey, err := crypto.HexToECDSA(pk)
	if err != nil {
		return err
	}

	publicKey := privateKey.Public()

	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return errors.New("error casting public key to ECDSA")
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()

	if address != appConfig.Blockchain.Account.Address {
		return errors.New("account address and private key mismatch")
	}

	return nil
}

func GetConfig() *AppConfig {
	return appConfig
}
