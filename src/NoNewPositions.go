
package src

import (
  "sync"
)


var NNP = NewNoNewPositions()


type NoNewPositions struct {
  flag bool
  m map[string]bool
  rwm sync.RWMutex
}


func (n *NoNewPositions) NoNewPositionsTrue(id string) {
  n.rwm.Lock()
  defer n.rwm.Unlock()
  n.m[id] = true
  n.flag = true
}


func (n *NoNewPositions) NoNewPositionsFalse(id string) {
  n.rwm.Lock()
  defer n.rwm.Unlock()
  n.m[id] = false
  for _, val := range n.m {
    if val == true {
      return
    }
  }
  n.flag = false
}


func NewNoNewPositions() (n *NoNewPositions) {
  n = &NoNewPositions{
    flag: false,
    m: make(map[string]bool),
  }
  n.m["Account.listen"] = false
  n.m["Market.listen"] = false
  n.m["Database"] = false
  return
}
