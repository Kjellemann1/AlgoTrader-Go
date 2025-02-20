package main

import (
  "testing"
  "time"
  "github.com/stretchr/testify/assert"

  "github.com/Kjellemann1/AlgoTrader-Go/constant"
)

func newAsset(test_window_size int) (a *Asset) {
  a = &Asset{
    Symbol: "Foo",
    O: make([]float64, test_window_size),
    H: make([]float64, test_window_size),
    L: make([]float64, test_window_size),
    C: make([]float64, test_window_size),
  }
  return
}

func TestIndexSingle(t *testing.T) {
  a := newAsset(constant.WINDOW_SIZE)
  pos := 0
  for i := 0; i < constant.WINDOW_SIZE; i++ {
    a.C[i] = float64(i)
  }
  arr := a.C[a.I(pos)]
  assert.Equal(t, a.C[len(a.C) - 1 - pos], arr)
}


func TestIndexArray(t *testing.T) {
  a := newAsset(constant.WINDOW_SIZE)
  from := 2
  to := 11
  for i := 0; i < constant.WINDOW_SIZE; i++ {
    a.C[i] = float64(i)
  }
  arr := a.IndexArray(&a.C, from, to)
  assert.Equal(t, 10, len(arr))
  for i := 0; i < len(arr); i++ {
    assert.Equal(t, a.C[len(a.C) - 1 - i - from], arr[len(arr) - 1 - i])
  }
}

func TestUpdateWindowOnBar(t *testing.T) {
  test_size := constant.WINDOW_SIZE * 2
  base_array := make([]float64, constant.WINDOW_SIZE)
  x := float64(test_size) - 1
  for i := constant.WINDOW_SIZE - 1; i >= 0; i-- {
    base_array[i] = x
    x--
  }
  a := newAsset(constant.WINDOW_SIZE)
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

func TestFillMissingMinutes(t *testing.T) {
  a := newAsset(constant.WINDOW_SIZE)
  a.Time = time.Date(2001, 1, 1, 0, 1, 0, 0, time.UTC)
  for i := constant.WINDOW_SIZE - 1; i >= 0; i-- {
    a.C[i] = float64(i)
    a.O[i] = float64(i)
    a.H[i] = float64(i)
    a.L[i] = float64(i)
  }
  base_array := make([]float64, constant.WINDOW_SIZE)
  for i := constant.WINDOW_SIZE - 1; i >= 0; i-- {
    base_array[i] = float64(i)
  }

  // Different day for stock
  a.AssetClass = "stock"
  new_time := time.Date(2001, 1, 2, 1, 4, 0, 0, time.UTC)
  a.fillMissingMinutes(new_time)
  assert.Equal(t, base_array, a.C)

  // Different day for crypto
  a.AssetClass = "crypto"
  for i := 0; i < 2; i++ {
    rollFloat(&base_array, a.C[constant.WINDOW_SIZE - 1])
  }
  a.Time = time.Date(2001, 1, 1, 23, 59, 0, 0, time.UTC)
  new_time = time.Date(2001, 1, 2, 0, 2, 0, 0, time.UTC)
  a.fillMissingMinutes(new_time)
  assert.Equal(t, base_array, a.C)
  
  // Same day missing minutes
  for i := constant.WINDOW_SIZE - 1; i >= 0; i-- {
    a.C[i] = float64(i)
    a.O[i] = float64(i)
    a.H[i] = float64(i)
    a.L[i] = float64(i)
  }
  for i := constant.WINDOW_SIZE - 1; i >= 0; i-- {
    base_array[i] = float64(i)
  }
  for i := 0; i < 3; i++ {
    rollFloat(&base_array, a.C[constant.WINDOW_SIZE - 1])
  }
  a.Time = time.Date(2001, 1, 1, 0, 1, 0, 0, time.UTC)
  new_time = time.Date(2001, 1, 1, 0, 5, 0, 0, time.UTC)
  a.fillMissingMinutes(new_time)
  assert.Equal(t, base_array, a.C)
}
