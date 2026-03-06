package testutil

import (
	"context"
	"reflect"
)

var (
	CtxType = reflect.TypeOf((*context.Context)(nil)).Elem()
)
