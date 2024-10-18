
package backoff

import "time"

func Backoff(duration time.Duration, max_duration time.Duration) time.Duration {
  time.Sleep(duration)
  if duration * 2 > max_duration {
    return max_duration
  }
  return duration * 2
}
