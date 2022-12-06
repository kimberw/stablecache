package stablecache

import (
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestLRUCache(t *testing.T) {
	cache := New("lru")
	cache.WithCallback(getmessage)

	Convey(fmt.Sprintf("key %v expect %v", "123", "value_123"), t, func() {
		// So(reverse1(item.from, item.to), ShouldResemble, item.result)
		// So(reverse2(item.from, item.to), ShouldResemble, item.result)
		key := "123"
		value, err := cache.Get(key)
		So(err, ShouldResemble, nil)
		So(value, ShouldResemble, message)
	})
	cache = nil
	runtime.GC()
}

func BenchmarkGetLRUCache(b *testing.B) {
	cache := New("lru")
	cache.WithCallback(getmessage)
	for i := 0; i < b.N; i++ {
		cache.Get("123")
	}
	cache = nil
	runtime.GC()
}

func BenchmarkWriteToLRUCache(b *testing.B) {
	b.Run("_size2^6", func(b *testing.B) {
		m := blob('a', 1<<6)
		writeToLRUCache(b, m)
	})
	b.Run("_size2^8", func(b *testing.B) {
		m := blob('a', 1<<8)
		writeToLRUCache(b, m)
	})
	b.Run("_size2^10", func(b *testing.B) {
		m := blob('a', 1<<10)
		writeToLRUCache(b, m)
	})
	b.Run("_size2^15", func(b *testing.B) {
		m := blob('a', 1<<15)
		writeToLRUCache(b, m)
	})
	b.Run("_size2^20", func(b *testing.B) {
		m := blob('a', 1<<20)
		writeToLRUCache(b, m)
	})
}

func writeToLRUCache(b *testing.B, data []byte) {
	cache := New("lru")
	rand.Seed(time.Now().Unix())

	b.RunParallel(func(pb *testing.PB) {
		id := rand.Int()
		counter := 0

		b.ReportAllocs()
		for pb.Next() {
			cache.SetWithExp(fmt.Sprintf("key-%d-%d", id, counter), data, 100*time.Second)
			counter = counter + 1
		}
	})
}

func BenchmarkReadFromLRUCache(b *testing.B) {
	readFromLRUCache(b)
}

func readFromLRUCache(b *testing.B) {
	cache := New("lru")
	cache.WithCallback(getmessage)
	for i := 0; i < b.N; i++ {
		cache.SetWithExp(strconv.Itoa(i), message, 100*time.Second)
	}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		b.ReportAllocs()

		for pb.Next() {
			cache.Get(strconv.Itoa(rand.Intn(b.N)))
		}
	})
}

func BenchmarkReadFromLRUCacheNonExistentKeys(b *testing.B) {
	readFromLRUCacheNonExistentKeys(b)
}

func readFromLRUCacheNonExistentKeys(b *testing.B) {
	cache := New("lru")
	cache.WithCallback(getmessage)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		b.ReportAllocs()

		for pb.Next() {
			cache.Get(strconv.Itoa(rand.Intn(b.N)))
		}
	})
}

// 运行 1分钟 1个key 请求 穿透下游次数
// go test -bench=BenchmarkReadLRUCache -benchtime=6s
func BenchmarkReadLRUCache(b *testing.B) {
	lens := []int{1, 10, 100, 1000, 10000}
	for _, l := range lens {
		b.Run(fmt.Sprintf("request %d", l), func(b *testing.B) {
			readFromLRUCacheKKeys(b, l)
		})
	}
}

func readFromLRUCacheKKeys(b *testing.B, l int) {
	keys := make([]int, l)
	for i := 0; i < l; i++ {
		keys[i] = i
	}
	cache := New("lru")
	cache.WithCallback(getmessage)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		b.ReportAllocs()

		for pb.Next() {
			cache.Get(strconv.Itoa(keys[b.N%len(keys)]))
		}
		return
	})
}
