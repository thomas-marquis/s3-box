package utils_test

import (
	"fmt"
	"github.com/thomas-marquis/s3-box/internal/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_FormatSizeBytes(t *testing.T) {
	// Given
	cases := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{1024, "1.00 KB"},
		{1024 * 1024, "1.00 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
		{2 * 1024 * 1024 * 1024, "2.00 GB"},
		{1500 * 1024 * 1024, "1.46 GB"},
		{1024 * 1024 * 1024 * 1024, "1.00 TB"},
		{1024 * 1024 * 1024 * 1024 * 1024, "1.00 PB"},
		{1025, "1.00 KB"},
		{1024*1024 + 512, "1.00 MB"},
		{1024*1024*1024 + 1024*1024*500, "1.49 GB"},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("FormatSizeBytes %d -> %s", c.input, c.expected), func(t *testing.T) {
			// When
			res := utils.FormatSizeBytes(c.input)

			// Then
			assert.Equal(t, c.expected, res)
		})
	}
}
