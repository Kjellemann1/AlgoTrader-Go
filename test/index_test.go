
package test

import (
  "testing"
  "github.com/stretchr/testify/assert"
  "github.com/Kjellemann1/AlgoTrader-Go/src/constant"
)


func TestIndexSingle(t *testing.T) {
  a := newAsset()
  pos := 0
  for i := 0; i < constant.WINDOW_SIZE; i++ {
    a.C[i] = float64(i)
  }
  arr := a.IndexSingle(&a.C, pos)
  assert.Equal(t, a.C[len(a.C) - 1 - pos], arr)
}


func TestIndexArray(t *testing.T) {
  a := newAsset()
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
