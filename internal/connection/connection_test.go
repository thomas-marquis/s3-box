package connection_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/connection"
)

func Test_ConnectionType_NewConnectionTypeFromString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected connection.ConnectionType
	}{
		{
			name:     "AWSConnectionType",
			input:    "aws",
			expected: connection.AWSConnectionType,
		},
		{
			name:     "AWSConnectionType upper case",
			input:    "AWS",
			expected: connection.AWSConnectionType,
		},
		{
			name:     "S3LikeConnectionType",
			input:    "s3-like",
			expected: connection.S3LikeConnectionType,
		},
		{
			name:     "DefaultConnectionType",
			input:    "",
			expected: connection.DefaultConnectionType,
		},
		{
			name:     "DefaultConnectionType with random value",
			input:    "fhziufh",
			expected: connection.DefaultConnectionType,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := connection.NewConnectionTypeFromString(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}
