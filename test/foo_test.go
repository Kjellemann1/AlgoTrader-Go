
package test

import (
  "testing"
  "github.com/Kjellemann1/AlgoTrader-Go/src"
)



func TestFoo(t *testing.T) {
  var x bool = src.Foo()
  if !x {
    t.Errorf("Error")
  }
}
