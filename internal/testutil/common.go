package testutil

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	CtxType = reflect.TypeOf((*context.Context)(nil)).Elem()
)

func AssertEventually(t *testing.T, done <-chan struct{}) {
	t.Helper()
	assert.Eventually(t, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}, 10*time.Second, 100*time.Millisecond)
}
