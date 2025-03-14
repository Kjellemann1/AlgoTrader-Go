package main

import (
  "sync"
  "time"
)

var NNP = NewNoNewPositions()

type NoNewPositions struct {
  Flag bool
  m map[string]bool
  rwm sync.RWMutex
}

func (n *NoNewPositions) NoNewPositionsTrue(id string) {
  n.rwm.Lock()
  defer n.rwm.Unlock()
  n.m[id] = true
  n.Flag = true
}

func (n *NoNewPositions) NoNewPositionsFalse(id string) {
  n.rwm.Lock()
  defer n.rwm.Unlock()
  n.m[id] = false
  for _, val := range n.m {
    if val {
      return
    }
  }
  n.Flag = false
}

func NewNoNewPositions() (n *NoNewPositions) {
  n = &NoNewPositions{
    Flag: false,
    m: make(map[string]bool),
  }
  return
}

func (n *NoNewPositions) RateLimitSleep() {
  go func() {
    n.NoNewPositionsTrue("RateLimitSleep")
    time.Sleep(30 * time.Second)
    n.NoNewPositionsFalse("RateLimitSleep")
  }()
}
