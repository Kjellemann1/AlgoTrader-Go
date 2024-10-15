
package test

import (
  "github.com/Kjellemann1/AlgoTrader-Go/src"

  "testing"
  "github.com/stretchr/testify/assert"
)


func TestUpdate(t *testing.T) {
  const TEST_SIZE int = 10000
  var base_array [src.BUFFER_SIZE]float64 = [src.BUFFER_SIZE]float64{}

  base_array[0] = float64(TEST_SIZE - 1)
  for i := 1; i < src.BUFFER_SIZE; i++ {
    base_array[i] = base_array[i-1] - 1.0
  }
  
  assert.Equal(t, float64(TEST_SIZE - 1), base_array[0])
  assert.Equal(t, float64(TEST_SIZE - src.BUFFER_SIZE), base_array[src.BUFFER_SIZE - 1])

  a := src.Asset{Symbol: "AAPL"}
  for i := 0; i < TEST_SIZE; i++ {
    j := float64(i)
    a.Update(j, j, j, j)
  }

  assert.Equal(t, base_array, a.O)
  assert.Equal(t, base_array, a.H)
  assert.Equal(t, base_array, a.L)
  assert.Equal(t, base_array, a.C)
}
