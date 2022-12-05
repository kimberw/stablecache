package stablecache

import (
	"sync"
	"time"
)

// SimpleCache
type SimpleCache struct {
	noCopy
	defaultDuration time.Duration
	mu              sync.RWMutex
	items           map[string]Item
	janitor         *janitor
	// barrier bool
	randfunc func(int64, int64) bool
	caller   func(string) (interface{}, error)
	size     uint32
	// order Order
}

// WithCallback set callback
func (c *SimpleCache) WithCallback(call func(string) (interface{}, error)) {
	c.caller = call
}

// WithRandfunc set rand func
func (c *SimpleCache) WithRandfunc(call func(int64, int64) bool) {
	c.randfunc = call
}

// Get SimpleCache value
// error maybe not found, timeout
func (c *SimpleCache) Get(k string) (r any, err error) {
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
		c.refresh(k, v)
		return v.obj, Timeout
	}
	if v.Disuse() {
		c.refresh(k, v)
		return v.obj, Disuse
	}
	c.refresh(k, v)
	return v.obj, nil
}

// Set set SimpleCache
func (c *SimpleCache) Set(k string, v any) {
	c.SetWithExp(k, v, c.defaultDuration)
}

// SetWithExp actively set SimpleCache value
func (c *SimpleCache) SetWithExp(k string, v any, dur time.Duration) {
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

func (c *SimpleCache) refresh(k string, i Item) {
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
func (c *SimpleCache) Load(ks []string) {
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

func (c *SimpleCache) deleteExpired() {
	// TODO:
}
