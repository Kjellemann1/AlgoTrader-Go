package main

import (
  "testing"
  "github.com/shopspring/decimal"
  "github.com/stretchr/testify/assert"
)

func TestSplit(t *testing.T) {
	tests := []struct {
		splits   int64
		numStr   string
		places   int32
		expected []string
	}{
		{splits: 3, numStr: "10.00", places: 2, expected: []string{"3.34", "3.33", "3.33"}},
		{splits: 4, numStr: "10.00", places: 2, expected: []string{"2.50", "2.50", "2.50", "2.50"}},
		{splits: 3, numStr: "10.05", places: 2, expected: []string{"3.35", "3.35", "3.35"}},
	}

	for _, tc := range tests {
		num, _ := decimal.NewFromString(tc.numStr)
		parts := split(tc.splits, num, tc.places)
		assert.Equal(t, int(tc.splits), len(parts), "Antall deler stemmer ikke for %s", tc.numStr)
		for i, expStr := range tc.expected {
			expectedDec, _ := decimal.NewFromString(expStr)
			assert.True(t, parts[i].Equal(expectedDec), "For %s, del %d: forventet %s, fikk %s", tc.numStr, i, expectedDec, parts[i])
		}
		sum := decimal.Zero
		for _, part := range parts {
			sum = sum.Add(part)
		}
		assert.True(t, sum.Equal(num), "For %s: summen av delene %s er ikke lik originaltallet", tc.numStr, sum)
	}
}
