package utils

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
)

func UnmarshalJson[T any](v any) (T, error) {
	data, err := jsoniter.Marshal(v)
	if err != nil {
		return *new(T), errors.WithMessage(err, "marshal json")
	}
	var result T
	if err := jsoniter.Unmarshal(data, &result); err != nil {
		return *new(T), errors.WithMessage(err, "unmarshal json")
	}
	return result, nil
}
