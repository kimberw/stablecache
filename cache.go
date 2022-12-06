package stablecache

import (
	"errors"
	"fmt"
	"math/rand"
	"time"
)

type Color int8

const (
	defaultSize = 1000
)

const (
	black Color = iota
	white       = iota
)

type noCopy struct{}

func (*noCopy) Lock() {}

type Type int8

const (
	LRU Type = iota
	LFU      = iota
	ARC      = iota
)

var (
	NotFound = errors.New("not found")
	Timeout  = errors.New("timeout")
	Disuse   = errors.New("disuse")
)

type Cache interface {
	WithCallback(func(string) (interface{}, error))
	WithRandfunc(func(int64, int64) bool)
	Get(k string) (r any, err error)
	Set(k string, v any)
	SetWithExp(k string, v any, dur time.Duration)
	Load(ks []string)
	deleteExpired()
	refresh(k string, i any)
}

func randfunc(t, d int64) bool {
	id := rand.Int63n(d * d * d)
	if id < t*t*t {
		return true
	}
	return false
}

// New new a Cache
// support type:
// - nolimit
// - limit_lru
// - limit_lfu
// - fragmented

func New(t string) Cache {
	switch t {
	case "nolimit":
		return NewSimpleCache()
	case "lru":
		return NewLRUCache(defaultSize)
	default:
		return NewSimpleCache()
	}
}

type Janitor struct {
	Interval time.Duration
	stop     chan bool
}

func (j *Janitor) Run(del func()) {
	ticker := time.NewTicker(j.Interval)
	for {
		select {
		case <-ticker.C:
			del()
		case <-j.stop:
			ticker.Stop()
			return
		}
	}
}

func (j *Janitor) Stop() {
	fmt.Println("janitor stop")
	j.stop <- true
}

func NewJanitor(ci time.Duration, del func()) *Janitor {
	j := &Janitor{
		Interval: ci,
		stop:     make(chan bool),
	}
	go j.Run(del)
	return j
}
