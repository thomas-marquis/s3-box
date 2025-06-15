package connections_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/domain/connections"
)

func Test_Provider_NewProviderFromString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected connections.Provider
	}{
		{
			name:     "AWSConnectionType",
			input:    "aws",
			expected: connections.AWSProvider,
		},
		{
			name:     "AWSConnectionType upper case",
			input:    "AWS",
			expected: connections.AWSProvider,
		},
		{
			name:     "S3LikeConnectionType",
			input:    "s3-like",
			expected: connections.S3LikeProvider,
		},
		{
			name:     "DefaultConnectionType",
			input:    "",
			expected: connections.DefaultProvider,
		},
		{
			name:     "DefaultConnectionType with random value",
			input:    "fhziufh",
			expected: connections.DefaultProvider,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := connections.NewProviderFromString(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}
