
package test

import (
  "time"
  "testing"
  "github.com/stretchr/testify/assert"

  "github.com/Kjellemann1/AlgoTrader-Go/src"
  "github.com/Kjellemann1/AlgoTrader-Go/src/constant"
)


func TestUpdateWindowOnBar(t *testing.T) {
  const TEST_SIZE int = 10000
  var base_array [constant.WINDOW_SIZE]float64 = [constant.WINDOW_SIZE]float64{}

  x := float64(TEST_SIZE) - 1
  for i := constant.WINDOW_SIZE - 1; i >= 0; i-- {
    base_array[i] = x
    x--
  }

  a := src.Asset{Symbol: "AAPL"}
  for i := 0; i < TEST_SIZE; i++ {
    j := float64(i)
    a.UpdateWindowOnBar(j, j, j, j, time.Now(), time.Now())
  }


  assert.Equal(t, base_array, a.Open)
  assert.Equal(t, base_array, a.High)
  assert.Equal(t, base_array, a.Low)
  assert.Equal(t, base_array, a.Close)
}
