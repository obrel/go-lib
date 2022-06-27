package rr

import (
  "fmt"
  "reflect"
  "testing"
)

func TestRoundRobin(t *testing.T) {
  tests := []struct {
    list     []string
    iserr    bool
    expected []string
    want     []string
  }{
    {
      list: []string{
        "192.168.33.10",
        "192.168.33.11",
        "192.168.33.12",
      },
      iserr: false,
      want: []string{
        "192.168.33.10",
        "192.168.33.11",
        "192.168.33.12",
        "192.168.33.10",
      },
    },
    {
      list: []string{},
      iserr: true,
      want: []string{},
    },
  }

  for i, test := range tests {
    list := make([]interface{}, 0, len(test.list))

    for _, l := range test.list {
      list = append(list, l)
    }

    rr, err := New(list...)

    if got, want := !(err == nil), test.iserr; got != want {
      t.Errorf("tests[%d] - RoundRobin iserr is wrong. want: %v, but got: %v", i, test.want, got)
    }

    list = make([]interface{}, 0, len(test.want))
    for j := 0; j < len(test.want); j++ {
      list = append(list, rr.Next())
    }

    gots := make([]string, 0, len(list))

    for _, l := range list {
      gots = append(gots, fmt.Sprintf("%v", l))
    }

    if got, want := gots, test.want; !reflect.DeepEqual(got, want) {
      t.Errorf("tests[%d] - RoundRobin is wrong. want: %v, got: %v", i, want, got)
    }
  }
}
