package task_args

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type ControlnetArgs struct {
	ImageDataURL string          `json:"image_dataurl" description:"The reference image DataURL" validate:"required"`
	Model        string          `json:"model" description:"The controlnet model name" validate:"required"`
	Preprocess   *PreprocessArgs `json:"preprocess" description:"Preprocess the image"`
	Weight       int             `json:"weight" validate:"max=100,min=1" description:"Weight of the controlnet model" validate:"required"`
}

func (*ControlnetArgs) GormDBDataType(_ *gorm.DB, _ *schema.Field) string {
	return "text"
}

func (controlnetArgs *ControlnetArgs) Scan(value interface{}) error {

	if controlnetArgs == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSON value:", value))
	}

	return json.Unmarshal(bytes, controlnetArgs)
}

func (controlnetArgs *ControlnetArgs) Value() (driver.Value, error) {

	if controlnetArgs == nil {
		return nil, nil
	}

	return json.Marshal(controlnetArgs)
}
