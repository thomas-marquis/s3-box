package tests

import (
	"fmt"
	"reflect"

	"go.uber.org/mock/gomock"
)

type eqDeref struct {
	expected any
}

var _ gomock.Matcher = (*eqDeref)(nil)

func EqDeref(expected any) gomock.Matcher {
	return &eqDeref{expected}
}

func (m *eqDeref) Matches(x any) bool {
	if x == nil {
		return false
	}
	xVal := reflect.ValueOf(x)
	if xVal.Kind() != reflect.Ptr || xVal.IsNil() {
		return false
	}
	return reflect.DeepEqual(m.expected, xVal.Elem().Interface())
}

func (m *eqDeref) String() string {
	return fmt.Sprintf("%v", m.expected)
}
