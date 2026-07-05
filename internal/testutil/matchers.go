package testutil

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type jsonEqMatcher struct {
	expectedJson string
	t            *testing.T
}

func JsonEqMatcher(t *testing.T, expectedJson string) gomock.Matcher {
	return jsonEqMatcher{expectedJson: expectedJson, t: t}
}

var _ gomock.Matcher = (*jsonEqMatcher)(nil)

func (m jsonEqMatcher) Matches(val any) bool {
	return assert.JSONEq(m.t, m.expectedJson, val.(string))
}

func (m jsonEqMatcher) String() string {
	return fmt.Sprintf("JSON is equal to %s", m.expectedJson)
}
