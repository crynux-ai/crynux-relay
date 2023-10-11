package task_args

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type LoraArgs struct {
	Model  string `json:"model" description:"The LoRA model name" validate:"required"`
	Weight int    `json:"weight" description:"The LoRA weight" validate:"required,min=1,max=100"`
}

func (*LoraArgs) GormDBDataType(_ *gorm.DB, _ *schema.Field) string {
	return "text"
}

func (loraArgs *LoraArgs) Scan(value interface{}) error {

	if loraArgs == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSON value:", value))
	}

	return json.Unmarshal(bytes, loraArgs)
}

func (loraArgs *LoraArgs) Value() (driver.Value, error) {

	if loraArgs == nil {
		return nil, nil
	}

	return json.Marshal(loraArgs)
}
