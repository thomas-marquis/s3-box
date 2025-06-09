package tests_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/connection"
	"github.com/thomas-marquis/s3-box/internal/tests"
)

func Test_eqDeref_Matches_ShouldReturnTrueForEqualValues(t *testing.T) {
	// Given
	type fakeStruct struct {
		Field1 string
	}

	conn1 := connection.NewConnection(
		"connection 1",
		"AZERTY",
		"1234",
		"MyBucket",
		connection.AsAWSConnection("eu-west-1"),
	)
	conn2 := connection.NewConnection(
		"connection 1",
		"AZERTY",
		"1234",
		"MyBucket",
		connection.AsAWSConnection("eu-west-1"),
		connection.WithID(conn1.ID()), // Ensure the ID remains the same for comparison
	)

	testCases := []struct {
		actual   any
		expected any
	}{
		{actual: "test value", expected: "test value"},
		{actual: 42, expected: 42},
		{actual: 3.14, expected: 3.14},
		{
			actual:   fakeStruct{Field1: "test"},
			expected: fakeStruct{Field1: "test"},
		},
		{
			actual:   *conn1,
			expected: *conn2,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Testing %T", tc.actual), func(t *testing.T) {
			// When
			matcher := tests.EqDeref(tc.expected)
			result := matcher.Matches(&tc.actual)

			// Then
			assert.True(t, result, fmt.Sprintf("Matcher should return true for equal values: %v", tc.actual))
		})
	}
}
