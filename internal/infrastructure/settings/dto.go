package settings

import (
	"errors"
	"time"

	"github.com/thomas-marquis/s3-box/internal/domain/settings"
)

var (
	errDtoInvalidType = errors.New("the type asked is different from the stored one")
)

type settingDTO struct {
	Name         string  `json:"name"`
	StringValue  *string `json:"strValue,omitempty"`
	Uint64Value  *uint64 `json:"u64Value,omitempty"`
	NanoSecValue *int64  `json:"nsValue,omitempty"`
}

func newDto(name string, value any) (settingDTO, error) {
	switch val := value.(type) {
	case string:
		return newDtoFromString(name, val), nil
	case uint64:
		return newDtoFromUint64(name, val), nil
	case time.Duration:
		return newDtoFromDuration(name, val), nil
	default:
		return settingDTO{}, errDtoInvalidType
	}
}

func newDtoFromString(name, value string) settingDTO {
	return settingDTO{
		Name:        name,
		StringValue: &value,
	}
}

func newDtoFromUint64(name string, value uint64) settingDTO {
	return settingDTO{
		Name:        name,
		Uint64Value: &value,
	}
}

func newDtoFromDuration(name string, value time.Duration) settingDTO {
	ns := value.Nanoseconds()
	return settingDTO{
		Name:         name,
		NanoSecValue: &ns,
	}
}

func (d settingDTO) Read() (any, settings.Type) {
	if d.StringValue != nil {
		return *d.StringValue, settings.StringType
	}
	if d.Uint64Value != nil {
		return *d.Uint64Value, settings.Uint64Type
	}
	if d.NanoSecValue != nil {
		return *d.NanoSecValue, settings.DurationType
	}

	return nil, ""
}
