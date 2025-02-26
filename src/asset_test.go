package main

import (
  "testing"
  "time"
  "github.com/shopspring/decimal"
  "github.com/stretchr/testify/assert"
  "github.com/qdm12/reprint"
  "github.com/Kjellemann1/AlgoTrader-Go/constant"
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
  for i := range constant.WINDOW_SIZE {
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
    for i := range len(arr) {
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
    for i := range test_size {
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
  assert.Equal(t, len(constant.STOCK_SYMBOLS), len(assets["stock"]))
  assert.Equal(t, len(constant.CRYPTO_SYMBOLS), len(assets["crypto"]))
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
		a.Class = "stock"
		a.Time = time.Date(2001, 1, 1, 0, 1, 0, 0, time.UTC)
		newTime := time.Date(2001, 1, 2, 1, 4, 0, 0, time.UTC)
		a.fillMissingMinutes(newTime)
		assert.Equal(t, b, a.C)
		assert.Equal(t, b, a.O)
		assert.Equal(t, b, a.H)
		assert.Equal(t, b, a.L)
	})

  for range 3 {
    rollFloat(&baseArray, asset.C[constant.WINDOW_SIZE-1])
  }

	t.Run("Different day for crypto", func(t *testing.T) {
    b := make([]float64, constant.WINDOW_SIZE)
    copy(b, baseArray)
    a, _ := reprint.This(asset).(*Asset)
		a.Time = time.Date(2001, 1, 1, 23, 59, 0, 0, time.UTC)
		a.Class = "crypto"
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

func TestPendingOrders(t *testing.T) {
  a := newAssetTesting()
  a.Positions = make(map[string]*Position)
  a.Positions["1"] = &Position{OpenOrderPending: true, CloseOrderPending: false, Symbol: "Foo"}
  a.Positions["2"] = &Position{OpenOrderPending: false, CloseOrderPending: true, Symbol: "Foo"}
  a.Positions["3"] = &Position{OpenOrderPending: false, CloseOrderPending: false, Symbol: "Foo"}
  b := newAssetTesting()
  b.Positions = make(map[string]*Position)
  b.Positions["1"] = &Position{OpenOrderPending: true, CloseOrderPending: false, Symbol: "Bar"}
  b.Positions["2"] = &Position{OpenOrderPending: false, CloseOrderPending: true, Symbol: "Bar"}
  b.Positions["3"] = &Position{OpenOrderPending: false, CloseOrderPending: false, Symbol: "Bar"}
  assets := make(map[string]map[string]*Asset)
  assets["x"], assets["y"] = make(map[string]*Asset), make(map[string]*Asset)
  assets["x"]["a"], assets["y"]["b"] = a, b
  pending := pendingOrders(assets)
  assert.Equal(t, 2, len(pending))
  assert.Equal(t, 2, len(pending["Foo"]))
  assert.Equal(t, 2, len(pending["Bar"]))
}

func TestPositionsSymbols(t *testing.T) {
  positions := make(map[string][]*Position)
  positions["Foo"] = []*Position{
    {StratName: "1", AssetClass: "stock"}, 
    {StratName: "2", AssetClass: "stock"},
  }
  positions["Bar"] = []*Position{
    {StratName: "1", AssetClass: "crypto"}, 
    {StratName: "2", AssetClass: "crypto"},
  }
  symbols := positionsSymbols(positions)
  assert.Equal(t, 2, len(symbols))
  assert.Contains(t, symbols, "stock")
  assert.Contains(t, symbols, "crypto")
  assert.Equal(t, 1, len(symbols["stock"]))
  assert.Equal(t, 1, len(symbols["crypto"]))
  assert.Contains(t, symbols["stock"], "Foo")
  assert.Contains(t, symbols["crypto"], "Bar")
}

func TestSumPosQtysEqAssetQty(t *testing.T) {
  a := newAssetTesting()
  a.Qty = decimal.NewFromInt(6)
  a.Positions = make(map[string]*Position)
  a.Positions["1"] = &Position{Qty: decimal.NewFromInt(1)}
  a.Positions["2"] = &Position{Qty: decimal.NewFromInt(1)}
  assert.False(t, a.sumPosQtysEqAssetQty())
  a.Positions["3"] = &Position{Qty: decimal.NewFromInt(2)}
  a.Positions["4"] = &Position{Qty: decimal.NewFromInt(2)}
  assert.True(t, a.sumPosQtysEqAssetQty())
}

func TestSumNoPendingPosQtys(t *testing.T) {
  a := newAssetTesting()
  a.Qty = decimal.NewFromInt(6)
  a.Positions = make(map[string]*Position)
  a.Positions["1"] = &Position{Qty: decimal.NewFromInt(10), OpenOrderPending: true, CloseOrderPending: false}
  a.Positions["2"] = &Position{Qty: decimal.NewFromInt(10), OpenOrderPending: false, CloseOrderPending: true}
  a.Positions["3"] = &Position{Qty: decimal.NewFromInt(1), OpenOrderPending: false, CloseOrderPending: false}
  a.Positions["4"] = &Position{Qty: decimal.NewFromInt(2), OpenOrderPending: false, CloseOrderPending: false}
  assert.Equal(t, decimal.NewFromInt(3), a.sumNoPendingPosQtys())
}
