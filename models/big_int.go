package models

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"math/big"
)

type BigInt struct {
	big.Int
}

func (i *BigInt) Scan(val interface{}) error {
	var intStr string
	switch v := val.(type) {
	case string:
		intStr = v
	case []byte:
		intStr = string(v)
	case nil:
		return nil
	default:
		return errors.New(fmt.Sprint("Unable to convert BigInt value to string: ", val))
	}

	_, success := i.SetString(intStr, 10)
	if !success {
		return errors.New("Unable to parse BigInt string: " + intStr)
	}
	return nil
}

func (i BigInt) Value() (driver.Value, error) {
	return i.String(), nil
}
