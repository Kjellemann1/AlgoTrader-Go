package main

import (
  "testing"
  "time"
  "github.com/stretchr/testify/assert"

  "github.com/Kjellemann1/AlgoTrader-Go/constant"
)

func TestUpdateWindowOnBar(t *testing.T) {
  test_size := 10000
  base_array := make([]float64, constant.WINDOW_SIZE)
  x := float64(test_size) - 1
  for i := constant.WINDOW_SIZE - 1; i >= 0; i-- {
    base_array[i] = x
    x--
  }
  a := newAsset()
  for i := 0; i < test_size; i++ {
    j := float64(i)
    a.UpdateWindowOnBar(j, j, j, j, time.Now(), time.Now())
  }
  assert.Equal(t, base_array, a.O)
  assert.Equal(t, base_array, a.H)
  assert.Equal(t, base_array, a.L)
  assert.Equal(t, base_array, a.C)
}

func TestPrepAssetsMap(t *testing.T) {
  assets := prepAssetsMap()
  assert.NotEmpty(t, assets)
  assert.Equal(t, len(constant.STOCK_LIST), len(assets["stock"]))
  assert.Equal(t, len(constant.CRYPTO_LIST), len(assets["crypto"]))
}
