package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReverseString(t *testing.T) {
	tests := []struct {
		str      string
		expected string
	}{
		{
			str:      "abcdef",
			expected: "fedcba",
		},
		{
			str:      "abcdefghi",
			expected: "ihgfedcba",
		},
	}

	for _, test := range tests {
		res := ReverseString(test.str)
		assert.Equal(t, test.expected, res)
	}
}
