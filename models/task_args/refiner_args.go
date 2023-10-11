package task_args

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type RefinerArgs struct {
	DenoisingCutoff int    `json:"denoising_cutoff" description:"Noise cutoff ratio between base model and refiner" validate:"required,max=100,min=1"`
	Model           string `json:"model" description:"The refiner model name" validate:"required"`
	Steps           int    `json:"steps" description:"Running steps for the refiner" validate:"required,min=10,max=100"`
}

func (*RefinerArgs) GormDBDataType(_ *gorm.DB, _ *schema.Field) string {
	return "text"
}

func (refinerArgs *RefinerArgs) Scan(value interface{}) error {

	if refinerArgs == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSON value:", value))
	}

	return json.Unmarshal(bytes, refinerArgs)
}

func (refinerArgs *RefinerArgs) Value() (driver.Value, error) {

	if refinerArgs == nil {
		return nil, nil
	}

	return json.Marshal(refinerArgs)
}
