package connection_deck_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
)

func Test_Provider_NewProviderFromString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected connection_deck.Provider
	}{
		{
			name:     "AWSConnectionType",
			input:    "aws",
			expected: connection_deck.AWSProvider,
		},
		{
			name:     "AWSConnectionType upper case",
			input:    "AWS",
			expected: connection_deck.AWSProvider,
		},
		{
			name:     "S3LikeConnectionType",
			input:    "s3-like",
			expected: connection_deck.S3LikeProvider,
		},
		{
			name:     "DefaultConnectionType",
			input:    "",
			expected: connection_deck.DefaultProvider,
		},
		{
			name:     "DefaultConnectionType with random value",
			input:    "fhziufh",
			expected: connection_deck.DefaultProvider,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := connection_deck.NewProviderFromString(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}
