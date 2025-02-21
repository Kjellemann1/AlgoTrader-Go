package main

import (
  "testing"
  "time"
  "github.com/stretchr/testify/assert"
  "github.com/Kjellemann1/AlgoTrader-Go/constant"
  "github.com/qdm12/reprint"
)

func newAssetTesting() (a *Asset) {
  a = &Asset{
    Symbol: "Foo",
    O: make([]float64, constant.WINDOW_SIZE),
    H: make([]float64, constant.WINDOW_SIZE),
    L: make([]float64, constant.WINDOW_SIZE),
    C: make([]float64, constant.WINDOW_SIZE),
  }
  return
}

func TestIndexingMethods(t *testing.T) {
  a := newAssetTesting()
  pos := 0
  for i := 0; i < constant.WINDOW_SIZE; i++ {
    a.C[i] = float64(i)
  }

  t.Run("i", func(t *testing.T) {
    arr := a.C[a.i(pos)]
    assert.Equal(t, a.C[len(a.C) - 1 - pos], arr)
  })

  t.Run("s", func(t *testing.T) {
    from := 2
    to := 11
    arr := a.s(&a.C, from, to)
    assert.Equal(t, 10, len(arr))
    for i := 0; i < len(arr); i++ {
      assert.Equal(t, a.C[len(a.C) - 1 - i - from], arr[len(arr) - 1 - i])
    }
  })
}

func TestWindowUpdate(t *testing.T) {
  t.Run("onBar", func(t *testing.T) {
    test_size := constant.WINDOW_SIZE * 2
    base_array := make([]float64, constant.WINDOW_SIZE)
    x := float64(test_size) - 1
    for i := constant.WINDOW_SIZE - 1; i >= 0; i-- {
      base_array[i] = x
      x--
    }
    a := newAssetTesting()
    for i := 0; i < test_size; i++ {
      j := float64(i)
      a.updateWindowOnBar(j, j, j, j, time.Now(), time.Now())
    }
    assert.Equal(t, base_array, a.O)
    assert.Equal(t, base_array, a.H)
    assert.Equal(t, base_array, a.L)
    assert.Equal(t, base_array, a.C)
  })
}

func TestPrepAssetsMap(t *testing.T) {
  assets := prepAssetsMap()
  assert.NotEmpty(t, assets)
  assert.Equal(t, len(constant.STOCK_LIST), len(assets["stock"]))
  assert.Equal(t, len(constant.CRYPTO_LIST), len(assets["crypto"]))
}

func TestFillMissingMinutes(t *testing.T) {
  asset := newAssetTesting()
  for i := constant.WINDOW_SIZE - 1; i >= 0; i-- {
    asset.C[i] = float64(i)
    asset.O[i] = float64(i)
    asset.H[i] = float64(i)
    asset.L[i] = float64(i)
  }

  baseArray := make([]float64, constant.WINDOW_SIZE)
  for i := constant.WINDOW_SIZE - 1; i >= 0; i-- {
    baseArray[i] = float64(i)
  }

	t.Run("Different day for stock", func(t *testing.T) {
    b := make([]float64, constant.WINDOW_SIZE)
    copy(b, baseArray)
    a, _ := reprint.This(asset).(*Asset)
		a.AssetClass = "stock"
		a.Time = time.Date(2001, 1, 1, 0, 1, 0, 0, time.UTC)
		newTime := time.Date(2001, 1, 2, 1, 4, 0, 0, time.UTC)
		a.fillMissingMinutes(newTime)
		assert.Equal(t, b, a.C)
		assert.Equal(t, b, a.O)
		assert.Equal(t, b, a.H)
		assert.Equal(t, b, a.L)
	})

  for i := 0; i < 3; i++ {
    rollFloat(&baseArray, asset.C[constant.WINDOW_SIZE-1])
  }

	t.Run("Different day for crypto", func(t *testing.T) {
    b := make([]float64, constant.WINDOW_SIZE)
    copy(b, baseArray)
    a, _ := reprint.This(asset).(*Asset)
		a.Time = time.Date(2001, 1, 1, 23, 59, 0, 0, time.UTC)
		a.AssetClass = "crypto"
		newTime := time.Date(2001, 1, 2, 0, 3, 0, 0, time.UTC)
		a.fillMissingMinutes(newTime)
		assert.Equal(t, b, a.C)
		assert.Equal(t, b, a.O)
		assert.Equal(t, b, a.H)
		assert.Equal(t, b, a.L)
	})

	t.Run("Same day", func(t *testing.T) {
    b := make([]float64, constant.WINDOW_SIZE)
    copy(b, baseArray)
    a, _ := reprint.This(asset).(*Asset)
		a.Time = time.Date(2001, 1, 1, 0, 1, 0, 0, time.UTC)
		newTime := time.Date(2001, 1, 1, 0, 5, 0, 0, time.UTC)
		a.fillMissingMinutes(newTime)
		assert.Equal(t, b, a.C)
		assert.Equal(t, b, a.O)
		assert.Equal(t, b, a.H)
		assert.Equal(t, b, a.L)
	})
}
