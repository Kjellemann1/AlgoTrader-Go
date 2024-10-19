
package src

type Strategy struct {
  *Asset
}

// Remember to handle NO_NEW_TRADES flag

func (s *Strategy) ClosePosition() {
  // TODO
}


func (s *Strategy) OpenPosition() {
  // TODO
}


func (s *Strategy) CheckForSignal() {
  // TODO
}


func (s *Strategy) RSI() {

}


func NewStrategy() *Strategy {
  return &Strategy{}
}
