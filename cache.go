package stablecache

import (
	"errors"
	"math/rand"
	"time"
)

type Color int8

const (
	black Color = iota
	white Color = iota
)

type Item struct {
	obj        interface{}
	expiration int64
	duration   int64
	color      Color
}

// Expired is expired data
func (i Item) Expired() bool {
	if i.expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > i.expiration
}

// Disuse is disuse data
func (i Item) Disuse() bool {
	if i.color != black {
		return false
	}
	return true
}

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
	refresh(k string, i Item)
	Load(ks []string)
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
		return &SimpleCache{
			items:           make(map[string]Item),
			defaultDuration: 10 * time.Second,
			randfunc:        randfunc,
		}
	default:
		return &SimpleCache{
			items:           make(map[string]Item),
			defaultDuration: 10 * time.Second,
			randfunc:        randfunc,
		}
	}
}

type janitor struct {
	Interval time.Duration
	stop     chan bool
}

func (j *janitor) Run(c *SimpleCache) {
	ticker := time.NewTicker(j.Interval)
	for {
		select {
		case <-ticker.C:
			c.deleteExpired()
		case <-j.stop:
			ticker.Stop()
			return
		}
	}
}

func stopJanitor(c *SimpleCache) {
	c.janitor.stop <- true
}

func runJanitor(c *SimpleCache, ci time.Duration) {
	j := &janitor{
		Interval: ci,
		stop:     make(chan bool),
	}
	c.janitor = j
	go j.Run(c)
}
