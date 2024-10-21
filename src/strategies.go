
package src

var OpenCount int = 0
func (a *Asset) testingStrategy() {
  if OpenCount < 2 {
    a.OpenPosition("long", "IOC", "test")
  }
  OpenCount++
}
