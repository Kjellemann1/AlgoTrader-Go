
package main

import (
  "testing"

  "github.com/stretchr/testify/assert"
)

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
