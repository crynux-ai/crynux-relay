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

		appConfig.Blockchain.Account.PrivateKey = GetPrivateKey()

		if err := checkPrivateKeyAndAddress(appConfig.Blockchain.Account.Address, appConfig.Blockchain.Account.PrivateKey); err != nil {
			return err
		}

		appConfig.Test.RootPrivateKey = GetTestPrivateKey()
		if err := checkPrivateKeyAndAddress(appConfig.Test.RootAddress, appConfig.Test.RootPrivateKey); err != nil {
			return err
		}
	}
	return nil
}

func checkPrivateKeyAndAddress(address, privateKey string) error {

	if address == "" {
		return errors.New("address not set")
	}

	if privateKey == "" {
		return errors.New("private key not set")
	}

	var testPk string
	if strings.HasPrefix(privateKey, "0x") {
		testPk = privateKey[2:]
	} else {
		testPk = privateKey
	}

	testRootPrivateKey, err := crypto.HexToECDSA(testPk)
	if err != nil {
		return err
	}

	testRootPublicKey := testRootPrivateKey.Public()

	testRootPublicKeyECDSA, ok := testRootPublicKey.(*ecdsa.PublicKey)
	if !ok {
		return errors.New("error casting test root public key to ECDSA")
	}

	testRootAddress := crypto.PubkeyToAddress(*testRootPublicKeyECDSA).Hex()

	if testRootAddress != address {
		return errors.New("account address and private key mismatch")
	}

	return nil
}

func GetConfig() *AppConfig {
	return appConfig
}
