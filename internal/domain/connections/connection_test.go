package connections_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/domain/connections"
)

func Test_Connection_Compare(t *testing.T) {
	conns := connections.New()
	tests := []struct {
		name     string
		conn1    *connections.Connection
		conn2    *connections.Connection
		expected bool
	}{
		{
			name:     "Equal connections",
			conn1:    conns.NewConnection("Test Connection", "AccessKey1", "SecretKey1", "Bucket1"),
			conn2:    conns.NewConnection("Test Connection", "AccessKey1", "SecretKey1", "Bucket1"),
			expected: true,
		},
		{
			name:     "Different connections",
			conn1:    conns.NewConnection("Connection 1", "AccessKey1", "SecretKey1", "Bucket1"),
			conn2:    conns.NewConnection("Connection 2", "AccessKey2", "SecretKey2", "Bucket2"),
			expected: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.conn1.Compare(test.conn2)
			assert.Equal(t, test.expected, result)
		})
	}
}
