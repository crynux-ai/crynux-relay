package models

import (
	"database/sql/driver"
	"encoding/json"
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

func (b BigInt) MarshalJSON() ([]byte, error) {
	return json.Marshal((*big.Int)(&b.Int).String())
}

func (b *BigInt) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	_, success := (*big.Int)(&b.Int).SetString(s, 10)
	if !success {
		return fmt.Errorf("failed to parse big.Int from string: %s", s)
	}
	return nil
}