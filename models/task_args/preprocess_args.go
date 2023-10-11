package task_args

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type CannyPreprocessArgs struct {
	HighThreshold int `json:"high_threshold" validate:"required,max=255,min=1"`
	LowThreshold  int `json:"low_threshold" validate:"required,max=255,min=1"`
}

type PreprocessArgs struct {
	Args   any    `json:"args" description:"Args for different preprocess methods"`
	Method string `json:"method" validate:"required"`
}

const (
	PreprocessMethodCanny = "canny"
)

func (preprocessArgs *PreprocessArgs) MarshalJSON() ([]byte, error) {

	output := make(map[string]any)

	output["method"] = preprocessArgs.Method

	if preprocessArgs.Args != nil {

		var argsBytes []byte
		var err error

		if preprocessArgs.Method == PreprocessMethodCanny {
			cannyArgs := preprocessArgs.Args.(*CannyPreprocessArgs)

			argsBytes, err = json.Marshal(cannyArgs)
			if err != nil {
				return nil, err
			}
		}

		if len(argsBytes) != 0 {
			argsMap := make(map[string]any)

			err = json.Unmarshal(argsBytes, &argsMap)
			if err != nil {
				return nil, err
			}

			output["args"] = argsMap
		}
	}

	return json.Marshal(output)
}

func (preprocessArgs *PreprocessArgs) UnmarshalJSON(data []byte) error {

	resultMap := make(map[string]any)

	err := json.Unmarshal(data, &resultMap)
	if err != nil {
		return err
	}

	preprocessArgs.Method = resultMap["method"].(string)

	if resultMap["args"] != nil {

		argsBytes, err := json.Marshal(resultMap["args"])
		if err != nil {
			return err
		}

		if preprocessArgs.Method == PreprocessMethodCanny {
			args := &CannyPreprocessArgs{}

			err := json.Unmarshal(argsBytes, args)

			if err != nil {
				return err
			}

			preprocessArgs.Args = args

		} else {
			return errors.New(fmt.Sprint("Unknown preprocess type: ", preprocessArgs.Method))
		}
	}

	return nil
}

func (*PreprocessArgs) GormDBDataType(_ *gorm.DB, _ *schema.Field) string {
	return "text"
}

func (preprocessArgs *PreprocessArgs) Scan(value interface{}) error {

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSON value:", value))
	}

	return preprocessArgs.UnmarshalJSON(bytes)
}

func (preprocessArgs *PreprocessArgs) Value() (driver.Value, error) {
	return preprocessArgs.MarshalJSON()
}
