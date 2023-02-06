package stablecache

import (
	"container/list"
	"fmt"
	"runtime"
	"stablecache/basic"
	"sync"
	"time"
	"unsafe"
)

type LRUItem[K comparable, V any] struct {
	obj        V
	key        K
	expiration int64
	duration   int64
	p          *list.Element
}

// Expired is expired data
func (i *LRUItem[K, V]) Expired() bool {
	if i.expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > i.expiration
}

// LRUBucket
type LRUBucket[K comparable, V any] struct {
	noCopy
	defaultDuration time.Duration
	mu              sync.RWMutex
	items           map[uintptr]LRUItem[K, V]
	order           *list.List
	size            uint64
}

func (b *LRUBucket[K, V]) clean() {
	b.order = nil
	b.items = nil
}

func (b *LRUBucket[K, V]) initBucket(size uint64) {
	b.items = make(map[uintptr]LRUItem[K, V])
	b.order = list.New()
	b.size = size
}

// Get LRUBucket value
// error maybe not found, timeout
func (b *LRUBucket[K, V]) Get(p *LRUCache[K, V], k K, h uintptr) (r V, err error) {
	b.mu.RLock()
	item, ok := b.items[h]
	if !ok || item.key != k {
		if p.caller == nil {
			return r, NotFound
		}
		b.mu.RUnlock()
		v, err := p.caller(k)
		if err != nil {
			return r, NotFound
		}
		b.SetWithExp(k, v, h, p.defaultDuration)
		return v, nil
	}
	b.mu.RUnlock()
	b.mu.Lock()
	b.move(item.p)
	b.mu.Unlock()
	if item.Expired() {
		b.refresh(p, k, h, item)
		return item.obj, Timeout
	}
	b.refresh(p, k, h, item)
	return item.obj, nil
}

// SetWithExp actively set LRUBucket value
func (b *LRUBucket[K, V]) SetWithExp(k K, v V, h uintptr, dur time.Duration) (*list.Element, bool) {
	b.mu.Lock()
	i, ok := b.items[h]
	if ok {
		i.key = k
		i.obj = v
		i.expiration = time.Now().Add(dur).UnixNano()
		i.duration = int64(dur)
		b.mu.Unlock()
		return i.p, true
	}
	p := b.add(k)
	b.items[h] = LRUItem[K, V]{
		key:        k,
		obj:        v,
		expiration: time.Now().Add(dur).UnixNano(),
		duration:   int64(dur),
		p:          p,
	}
	b.mu.Unlock()
	return p, true
}

func (b *LRUBucket[K, V]) refresh(p *LRUCache[K, V], k K, h uintptr, tItem LRUItem[K, V]) {
	if p.caller == nil {
		return
	}
	t := tItem.expiration - time.Now().UnixNano()
	if t > 0 && t*100/tItem.duration < 30 {
		if p.randfunc != nil && !p.randfunc(t, tItem.duration) {
			return
		}
		v, err := p.caller(k)
		if err == nil {
			b.SetWithExp(k, v, h, p.defaultDuration)
		}
	}
}

func (b *LRUBucket[K, V]) deleteExpired() {
	now := time.Now().UnixNano()
	i := 0
	b.mu.Lock()
	for k, item := range b.items {
		if i >= basic.DeleteNums {
			break
		}
		if item.expiration < now {
			i++
			b.remove(item.p)
			delete(b.items, k)
		}
	}
	b.mu.Unlock()
}

func (b *LRUBucket[K, V]) move(e *list.Element) {
	if e != nil {
		b.order.MoveToFront(e)
	}
}

func (b *LRUBucket[K, V]) add(key K) *list.Element {
	return b.order.PushFront(key)
}

func (b *LRUBucket[K, V]) remove(e *list.Element) {
	b.order.Remove(e)
}

// LRUCache
type LRUCache[K comparable, V any] struct {
	noCopy
	defaultDuration time.Duration
	mask            uintptr
	buckets         []LRUBucket[K, V]
	randfunc        func(int64, int64) bool
	caller          func(K) (V, error)
	janitor         *Janitor
}

// NewLRUCache new cache
func NewLRUCache[K comparable, V any](size uint64) *LRUCache[K, V] {
	c := &LRUCache[K, V]{
		mask:            uintptr(basic.InitialSize - 1),
		buckets:         make([]LRUBucket[K, V], basic.InitialSize),
		defaultDuration: 10 * time.Second,
		randfunc:        randfunc,
	}
	c.initBucket(size)
	j := NewJanitor(1*time.Second, c.deleteExpired)
	runtime.SetFinalizer(c, (*LRUCache[K, V]).clean)
	c.janitor = j
	return c
}

func (c *LRUCache[K, V]) initBucket(size uint64) {
	for i := range c.buckets {
		c.buckets[i].initBucket(size/basic.InitialSize + 1)
	}
}
func (c *LRUCache[K, V]) clean() {
	fmt.Println("lru stop")
	if c.janitor != nil {
		c.janitor.Stop()
		c.janitor = nil
	}
	for i := range c.buckets {
		c.buckets[i].clean()
	}
}

// WithCallback set callback
func (c *LRUCache[K, V]) WithCallback(call func(K) (V, error)) {
	c.caller = call
}

// WithRandfunc set rand func
func (c *LRUCache[K, V]) WithRandfunc(call func(int64, int64) bool) {
	c.randfunc = call
}
func ehash(i interface{}) uintptr {
	return nilinterhash(noescape(unsafe.Pointer(&i)), 0xdeadbeef)
}

//go:linkname nilinterhash runtime.nilinterhash
func nilinterhash(p unsafe.Pointer, h uintptr) uintptr

//go:nocheckptr
//go:nosplit
func noescape(p unsafe.Pointer) unsafe.Pointer {
	x := uintptr(p)
	return unsafe.Pointer(x ^ 0)
}

func (c *LRUCache[K, V]) Get(k K) (r V, err error) {
	hash := ehash(k)
	i := hash & c.mask
	b := &(c.buckets[i])
	r, err = b.Get(c, k, hash)
	return
}

func (c *LRUCache[K, V]) Set(k K, v V) {
	c.SetWithExp(k, v, c.defaultDuration)
}

// SetWithExp actively set LRUBucket value
func (c *LRUCache[K, V]) SetWithExp(k K, v V, dur time.Duration) {
	hash := ehash(k)
	i := hash & c.mask
	b := &(c.buckets[i])
	p, exsit := b.SetWithExp(k, v, hash, dur)
	if exsit {
		b.move(p)
	}
}

func (c *LRUCache[K, V]) deleteExpired() {
	for i := range c.buckets {
		c.buckets[i].deleteExpired()
	}
}
