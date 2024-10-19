
package backoff

import "time"

func BackoffWithMax(backoff_sec *int, backoff_max_sec int) {
  time.Sleep(time.Duration(*backoff_sec) * time.Second)
  if *backoff_sec >= backoff_max_sec {
    *backoff_sec = backoff_max_sec
  } else{
    *backoff_sec = *backoff_sec * 2
  }
}

func Backoff(backoff_sec *int) {
  time.Sleep(time.Duration(*backoff_sec) * time.Second)
  *backoff_sec = *backoff_sec * 2
}
