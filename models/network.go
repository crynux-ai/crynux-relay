package models

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"math/big"

	"gorm.io/gorm"
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

type NetworkNodeNumber struct {
	gorm.Model
	AllNodes    uint64 `json:"all_nodes"`
	BusyNodes   uint64 `json:"busy_nodes"`
	ActiveNodes uint64 `json:"active_nodes"`
}

type NetworkTaskNumber struct {
	gorm.Model
	TotalTasks   uint64 `json:"total_tasks"`
	RunningTasks uint64 `json:"running_tasks"`
	QueuedTasks  uint64 `json:"queued_tasks"`
}

type NetworkNodeData struct {
	gorm.Model
	Address   string `json:"address" gorm:"index"`
	CardModel string `json:"card_model"`
	VRam      int    `json:"v_ram"`
	Balance   BigInt `json:"balance" gorm:"type:string;size:255"`
	QoS       int64  `json:"qos"`
}
