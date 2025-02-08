
package test

import (
  "testing"
  "github.com/Kjellemann1/AlgoTrader-Go/src"

  "github.com/stretchr/testify/assert"
)

var NNP = src.NewNoNewPositions()

func TestNoNewPositions(t *testing.T) {
  NNP.NoNewPositionsTrue("ID1")
  assert.True(t, NNP.Flag == true)

  NNP.NoNewPositionsTrue("ID2")
  assert.True(t, NNP.Flag == true)

  NNP.NoNewPositionsFalse("ID1")
  assert.True(t, NNP.Flag == true)

  NNP.NoNewPositionsFalse("ID2")
  assert.True(t, NNP.Flag == false)
}
