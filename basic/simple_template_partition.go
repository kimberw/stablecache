package basic

import (
	"sync"
	"time"
	"unsafe"
)

const (
	InitialSize = 1 << 4
)

// PartitionCache
type PartitionCache[K comparable, V any] struct {
	noCopy
	defaultDuration time.Duration
	mask            uintptr
	buckets         []bucket[K, V]
	randfunc        func(int64, int64) bool
	caller          func(K) (V, error)
}

// bucket
type bucket[K comparable, V any] struct {
	noCopy
	defaultDuration time.Duration
	mu              sync.RWMutex
	items           map[uintptr]TemplateItem[K, V]
}

func (b *bucket[K, V]) clean() {
}

func (b *bucket[K, V]) initBucket() {
	b.items = make(map[uintptr]TemplateItem[K, V])
}

// NewPartitionCache new cache
func NewPartitionCache[K comparable, V any]() *PartitionCache[K, V] {
	c := &PartitionCache[K, V]{
		mask:            uintptr(InitialSize - 1),
		buckets:         make([]bucket[K, V], InitialSize),
		defaultDuration: 10 * time.Second,
		randfunc:        randfunc,
	}
	c.initBucket()
	return c
}

func (c *PartitionCache[K, V]) initBucket() {
	for i := range c.buckets {
		c.buckets[i].initBucket()
	}
}

// WithCallback set callback
func (c *PartitionCache[K, V]) WithCallback(call func(K) (V, error)) {
	c.caller = call
}

// WithRandfunc set rand func
func (c *PartitionCache[K, V]) WithRandfunc(call func(int64, int64) bool) {
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

func (c *PartitionCache[K, V]) Get(k K) (r V, err error) {
	hash := ehash(k)
	i := hash & c.mask
	b := &(c.buckets[i])
	return b.Get(c, k, hash)
}

func (c *PartitionCache[K, V]) Set(k K, v V) {
	c.SetWithExp(k, v, c.defaultDuration)
}

// SetWithExp actively set bucket value
func (c *PartitionCache[K, V]) SetWithExp(k K, v V, dur time.Duration) {
	hash := ehash(k)
	i := hash & c.mask
	b := &(c.buckets[i])
	b.SetWithExp(k, v, hash, dur)
}

func (c *PartitionCache[K, V]) deleteExpired() {
	// TODO:
}

// Get bucket value
// error maybe not found, timeout
func (b *bucket[K, V]) Get(p *PartitionCache[K, V], k K, h uintptr) (r V, err error) {
	b.mu.RLock()
	item, ok := b.items[h]
	if !ok || item.k != k {
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
	if item.Expired() {
		b.refresh(p, k, h, item)
		return item.obj, Timeout
	}
	if item.Disuse() {
		b.refresh(p, k, h, item)
		return item.obj, Disuse
	}
	b.refresh(p, k, h, item)
	return item.obj, nil
}

// SetWithExp actively set bucket value
func (b *bucket[K, V]) SetWithExp(k K, v V, h uintptr, dur time.Duration) {
	b.mu.Lock()
	i, ok := b.items[h]
	if ok {
		i.k = k
		i.obj = v
		i.expiration = time.Now().Add(dur).UnixNano()
		i.duration = int64(dur)
		// b.items[k] = i
		b.mu.Unlock()
		return
	}
	b.items[h] = TemplateItem[K, V]{
		k:          k,
		obj:        v,
		expiration: time.Now().Add(dur).UnixNano(),
		duration:   int64(dur),
		color:      black,
	}
	b.mu.Unlock()
}

func (b *bucket[K, V]) refresh(p *PartitionCache[K, V], k K, h uintptr, tItem TemplateItem[K, V]) {
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

func (b *bucket[K, V]) deleteExpired() {
	// TODO:
}
