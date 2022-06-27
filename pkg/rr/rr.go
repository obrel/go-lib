package rr

import (
  "errors"
  "sync/atomic"
)

var ErrServersNotExists = errors.New("Data is not exists.")

type RoundRobin interface {
  Next() interface{}
}

type roundrobin struct {
  list []interface{}
  next uint32
}

func New(list ...interface{}) (RoundRobin, error) {
  if len(list) == 0 {
    return nil, ErrServersNotExists
  }

  return &roundrobin{
    list: list,
  }, nil
}

func (r *roundrobin) Next() interface{} {
  n := atomic.AddUint32(&r.next, 1)
  return r.list[(int(n)-1)%len(r.list)]
}
