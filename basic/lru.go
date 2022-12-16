package basic

import (
	"container/list"
	"fmt"
	"runtime"
	"sync"
	"time"
)

const (
	DeleteNums = 10
)

type LRUList struct {
	timestamp uint64
	key       string
}

type LRUItem struct {
	key        string
	obj        interface{}
	expiration int64
	duration   int64
	color      Color
	p          *list.Element
}

// Expired is expired data
func (i *LRUItem) Expired() bool {
	if i.expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > i.expiration
}

// Disuse is disuse data
func (i *LRUItem) Disuse() bool {
	if i.color != black {
		return false
	}
	return true
}

// LRUCache
type LRUCache struct {
	noCopy
	defaultDuration time.Duration
	mu              sync.RWMutex
	items           map[string]LRUItem
	janitor         *Janitor
	randfunc        func(int64, int64) bool
	caller          func(string) (interface{}, error)
	size            uint32
	order           *list.List
}

// NewLRUCache new cache
func NewLRUCache(size uint32) *LRUCache {
	c := &LRUCache{
		items:           make(map[string]LRUItem),
		defaultDuration: 10 * time.Second,
		randfunc:        randfunc,
		order:           list.New(),
		size:            size,
	}
	j := NewJanitor(1*time.Second, c.deleteExpired)
	runtime.SetFinalizer(c, (*LRUCache).clean)
	c.janitor = j
	return c
}

func (c *LRUCache) clean() {
	fmt.Println("lru stop")
	if c.janitor != nil {
		c.janitor.Stop()
		c.janitor = nil
	}
	c = nil
}

// WithCallback set callback
func (c *LRUCache) WithCallback(call func(string) (interface{}, error)) {
	c.caller = call
}

// WithRandfunc set rand func
func (c *LRUCache) WithRandfunc(call func(int64, int64) bool) {
	c.randfunc = call
}

// Get LRUCache value
// error maybe not found, timeout
func (c *LRUCache) Get(k string) (r any, err error) {
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

// Set set LRUCache
func (c *LRUCache) Set(k string, v any) {
	c.SetWithExp(k, v, c.defaultDuration)
}

// SetWithExp actively set LRUCache value
func (c *LRUCache) SetWithExp(k string, v any, dur time.Duration) {
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
	c.items[k] = LRUItem{
		obj:        v,
		expiration: time.Now().Add(dur).UnixNano(),
		duration:   int64(dur),
		color:      black,
		p:          c.add(k),
	}
	c.mu.Unlock()
}

func (c *LRUCache) move(item LRUItem) {
	if item.p != nil {
		c.order.MoveToFront(item.p)
	}
}

func (c *LRUCache) add(key string) *list.Element {
	return c.order.PushFront(key)
}

func (c *LRUCache) remove(item LRUItem) {
	c.order.Remove(item.p)
}

func (c *LRUCache) refresh(k string, i any) {
	item := i.(LRUItem)
	c.move(item)
	if c.caller == nil {
		return
	}
	t := item.expiration - time.Now().UnixNano()
	if t > 0 && t*100/item.duration < 30 {
		if c.randfunc != nil && !c.randfunc(t, item.duration) {
			return
		}
		v, err := c.caller(k)
		if err == nil {
			c.SetWithExp(k, v, c.defaultDuration)
		}
	}
}

// Load load keys avoid concurrent large traffic penetration
func (c *LRUCache) Load(ks []string) {
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

func (c *LRUCache) deleteExpired() {
	now := time.Now().UnixNano()
	i := 0
	c.mu.Lock()
	for k, item := range c.items {
		if i >= DeleteNums {
			break
		}
		if item.expiration < now {
			i++
			c.remove(c.items[k])
			delete(c.items, k)
		}
	}
	c.mu.Unlock()
}
