package stablecache

import (
	"errors"
	"fmt"
	"math/rand"
	"stablecache/basic"
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

type Cache[K comparable, V any] interface {
	WithCallback(func(K) (V, error))
	WithRandfunc(func(int64, int64) bool)
	Get(K) (V, error)
	Set(K, V)
	SetWithExp(K, V, time.Duration)
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

func New[K comparable, V any](t string) Cache[K, V] {
	switch t {
	case "normal":
		return basic.NewPartitionCache[K, V]()
	case "lru":
		// return NewLRUCache(defaultSize)
		return basic.NewPartitionCache[K, V]()
	default:
		return basic.NewPartitionCache[K, V]()
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
