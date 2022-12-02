package stablecache

import (
	"errors"
	"math/rand"
	"sync"
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

type Method int8

const (
	LRU Method = iota
	LFU        = iota
	ARC        = iota
)

var (
	NotFound = errors.New("not found")
	Timeout  = errors.New("timeout")
	Disuse   = errors.New("disuse")
)

// Cache
type Cache struct {
	noCopy
	defaultDuration time.Duration
	mu              sync.RWMutex
	method          Method
	items           map[string]Item
	janitor         *janitor
	// barrier bool
	randfunc func(int64, int64) bool
	caller   func(string) (interface{}, error)
	size     uint32
	// order Order
}

func randfunc(t, d int64) bool {
	id := rand.Int63n(d * d * d)
	if id < t*t*t {
		return true
	}
	return false
}

// New new a cache
func New() *Cache {
	items := make(map[string]Item)
	return &Cache{
		items:           items,
		defaultDuration: 10 * time.Second,
		randfunc:        randfunc,
	}
}

// WithCallback set callback
func (c *Cache) WithCallback(call func(string) (interface{}, error)) *Cache {
	c.caller = call
	return c
}

// WithRandfunc set rand func
func (c *Cache) WithRandfunc(call func(int64, int64) bool) *Cache {
	c.randfunc = call
	return c
}

// Get cache value
// error maybe not found, timeout
func (c *Cache) Get(k string) (r any, err error) {
	c.mu.RLock()
	v, ok := c.items[k]
	if !ok {
		if c.caller == nil {
			return nil, NotFound
		}
		c.mu.RUnlock()
		v, err := c.caller(k)
		if err != nil {
			return nil, NotFound
		}
		c.SetWithExp(k, v, c.defaultDuration)
		return v, nil
	}
	c.mu.RUnlock()
	if v.Expired() {
		c.Refresh(k, v)
		return v.obj, Timeout
	}
	if v.Disuse() {
		c.Refresh(k, v)
		return v.obj, Disuse
	}
	c.Refresh(k, v)
	return v.obj, nil
}

// Set set cache
func (c *Cache) Set(k string, v any) {
	c.SetWithExp(k, v, c.defaultDuration)
}

// SetWithExp actively set cache value
func (c *Cache) SetWithExp(k string, v any, dur time.Duration) {
	c.mu.Lock()
	i, ok := c.items[k]
	if ok {
		i.obj = v
		i.expiration = time.Now().Add(dur).UnixNano()
		i.duration = int64(dur)
		// c.items[k] = i
		c.mu.Unlock()
		return
	}
	c.items[k] = Item{
		obj:        v,
		expiration: time.Now().Add(dur).UnixNano(),
		duration:   int64(dur),
		color:      black,
	}
	c.mu.Unlock()
}

func (c *Cache) Refresh(k string, i Item) {
	if c.caller == nil {
		return
	}
	t := i.expiration - time.Now().UnixNano()
	if t > 0 && t*100/i.duration < 30 {
		if c.randfunc != nil && !c.randfunc(t, i.duration) {
			return
		}
		v, err := c.caller(k)
		if err == nil {
			c.SetWithExp(k, v, c.defaultDuration)
		}
	}
}

// Load load keys avoid concurrent large traffic penetration
func (c *Cache) Load(ks []string) {
	if c.caller == nil {
		return
	}
	for _, k := range ks {
		v, err := c.caller(k)
		if err == nil {
			c.SetWithExp(k, v, c.defaultDuration)
		}
	}
}

func (c *Cache) DeleteExpired() {
	// TODO:
}

type janitor struct {
	Interval time.Duration
	stop     chan bool
}

func (j *janitor) Run(c *Cache) {
	ticker := time.NewTicker(j.Interval)
	for {
		select {
		case <-ticker.C:
			c.DeleteExpired()
		case <-j.stop:
			ticker.Stop()
			return
		}
	}
}

func stopJanitor(c *Cache) {
	c.janitor.stop <- true
}

func runJanitor(c *Cache, ci time.Duration) {
	j := &janitor{
		Interval: ci,
		stop:     make(chan bool),
	}
	c.janitor = j
	go j.Run(c)
}
