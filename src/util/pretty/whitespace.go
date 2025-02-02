
package pretty

func AddWhitespace(s *string, n int) {
  for len(*s) < n {
    *s += " "
  }
}
