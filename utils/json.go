package utils

import "encoding/json"

func jsonRemarshal(bytes []byte) ([]byte, error) {
	var v interface{}
	if err := json.Unmarshal(bytes, &v); err != nil {
		return nil, err
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func JSONMarshalWithSortedKeys(v interface{}) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	data, err = jsonRemarshal(data)
	if err != nil {
		return nil, err
	}
	return data, nil
}
