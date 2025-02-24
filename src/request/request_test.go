package request

import (
  "fmt"
  "testing"
  "net/http"
  "io"
  "strings"
  "errors"
  "github.com/stretchr/testify/assert"
  "github.com/Kjellemann1/AlgoTrader-Go/push"
  "github.com/Kjellemann1/AlgoTrader-Go/constant"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
  return f(req)
}

func init() {
  HttpClient = nil
  if HttpClient != nil {
    panic("HttpClient is not nil")
  }

  push.DisablePush()
}

func TestGetPositions(t *testing.T) {
  iter := constant.GET_POSITIONS_RETRIES
  HttpClient = &http.Client{
    Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
      iter++
      if iter == 2 * constant.GET_POSITIONS_RETRIES - 1 {
        return &http.Response{ StatusCode: 200, Body: io.NopCloser(strings.NewReader(`[{"id":1}]`)) }, nil
      }
      return nil, errors.New("error")
    }),
  }

  t.Run("Success on retry", func(t *testing.T) {
    arr, err := GetPositions(0, 0)
    assert.Nil(t, err)
    assert.NotNil(t, arr)
  })

  iter = 0

  t.Run("Error on retry", func(t *testing.T) {
    arr, err := GetPositions(0, 0)
    assert.NotNil(t, err)
    assert.Nil(t, arr)
  })
}


func TestGetClosedOrders(t *testing.T) {
  iter := constant.GET_POSITIONS_RETRIES
  HttpClient = &http.Client{
    Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
      iter++
      if iter == 2 * constant.GET_POSITIONS_RETRIES - 1 {
        return &http.Response{ StatusCode: 200, Body: io.NopCloser(strings.NewReader(`[{"id":1}]`)) }, nil
      }
      return nil, errors.New("error")
    }),
  }

  t.Run("Success on retry", func(t *testing.T) {
    arr, err := GetPositions(0, 0)
    assert.Nil(t, err)
    assert.NotNil(t, arr)
  })

  iter = 0

  t.Run("Error on retry", func(t *testing.T) {
    arr, err := GetPositions(0, 0)
    assert.NotNil(t, err)
    assert.Nil(t, arr)
  })

  t.Run("Creating url", func(t *testing.T) {
    testMap := map[string]map[string]int{
      "stock": {
        "foo": 1,
      },
      "crypto": {
        "foo/bar": 1,
      },
    }
    url := urlGetClosedOrders(testMap)
    baseUrl := fmt.Sprintf(
      "%s/orders?status=closed&limit=500&direction=desc&symbols=%s",  // Max limit is 500
      constant.ENDPOINT, "foo%2Cfoo%2Fbar",
    )
    assert.Equal(t, baseUrl, url)
  })
}
