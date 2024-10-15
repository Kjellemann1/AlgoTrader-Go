
package util

import "log"

func AssertTrue(x bool, message string) {
  if x != true {
    log.Fatalf("Assertion failed: %v != true", x)
  }
}

func AssertNotNil(err error, message string) {
  if err != nil {
    log.Fatalf("Assertion failed: %v is nil", err)
  }
}

func AssertEqualInt(x int, y int, message string) {
  if x != y {
    log.Fatalf("Assertion failed: %v != %v", x, y)
  }
}
