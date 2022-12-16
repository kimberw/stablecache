package basic

import (
	"sync"
	"time"
)

type TemplateItem[K comparable, V any] struct {
	obj        V
	k          K
	expiration int64
	duration   int64
	color      Color
}

// Expired is expired data
func (i TemplateItem[K, V]) Expired() bool {
	if i.expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > i.expiration
}

// Disuse is disuse data
func (i TemplateItem[K, V]) Disuse() bool {
	if i.color != black {
		return false
	}
	return true
}

// TemplateCache
type TemplateCache[K comparable, V any] struct {
	noCopy
	defaultDuration time.Duration
	mu              sync.RWMutex
	items           map[K]TemplateItem[K, V]
	randfunc        func(int64, int64) bool
	caller          func(K) (V, error)
	size            uint32
	// order Order
}

func (c *TemplateCache[K, V]) clean() {
}

// NewTemplateCache new cache
func NewTemplateCache[K comparable, V any]() *TemplateCache[K, V] {
	return &TemplateCache[K, V]{
		items:           make(map[K]TemplateItem[K, V]),
		defaultDuration: 10 * time.Second,
		randfunc:        randfunc,
	}
}

// WithCallback set callback
func (c *TemplateCache[K, V]) WithCallback(call func(K) (V, error)) {
	c.caller = call
}

// WithRandfunc set rand func
func (c *TemplateCache[K, V]) WithRandfunc(call func(int64, int64) bool) {
	c.randfunc = call
}

// Get TemplateCache value
// error maybe not found, timeout
func (c *TemplateCache[K, V]) Get(k K) (r V, err error) {
	c.mu.RLock()
	item, ok := c.items[k]
	if !ok {
		if c.caller == nil {
			return r, NotFound
		}
		c.mu.RUnlock()
		v, err := c.caller(k)
		if err != nil {
			return r, NotFound
		}
		c.SetWithExp(k, v, c.defaultDuration)
		return v, nil
	}
	c.mu.RUnlock()
	if item.Expired() {
		c.refresh(k, item)
		return item.obj, Timeout
	}
	if item.Disuse() {
		c.refresh(k, item)
		return item.obj, Disuse
	}
	c.refresh(k, item)
	return item.obj, nil
}

// Set set TemplateCache
func (c *TemplateCache[K, V]) Set(k K, v V) {
	c.SetWithExp(k, v, c.defaultDuration)
}

// SetWithExp actively set TemplateCache value
func (c *TemplateCache[K, V]) SetWithExp(k K, v V, dur time.Duration) {
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
	c.items[k] = TemplateItem[K, V]{
		obj:        v,
		expiration: time.Now().Add(dur).UnixNano(),
		duration:   int64(dur),
		color:      black,
	}
	c.mu.Unlock()
}

func (c *TemplateCache[K, V]) refresh(k K, tItem TemplateItem[K, V]) {
	if c.caller == nil {
		return
	}
	t := tItem.expiration - time.Now().UnixNano()
	if t > 0 && t*100/tItem.duration < 30 {
		if c.randfunc != nil && !c.randfunc(t, tItem.duration) {
			return
		}
		v, err := c.caller(k)
		if err == nil {
			c.SetWithExp(k, v, c.defaultDuration)
		}
	}
}

// Load load keys avoid concurrent large traffic penetration
func (c *TemplateCache[K, V]) Load(ks []K) {
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

func (c *TemplateCache[K, V]) deleteExpired() {
	// TODO:
}
