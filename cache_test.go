package stablecache

import (
	"bytes"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCache(t *testing.T) {
	cache := New()
	cache.WithCallback(getmessage)

	Convey(fmt.Sprintf("key %v expect %v", "123", "value_123"), t, func() {
		// So(reverse1(item.from, item.to), ShouldResemble, item.result)
		// So(reverse2(item.from, item.to), ShouldResemble, item.result)
		key := "123"
		value, err := cache.Get(key)
		So(err, ShouldResemble, nil)
		So(value, ShouldResemble, message)
	})
}

func blob(char byte, len int) []byte {
	return bytes.Repeat([]byte{char}, len)
}

var message = blob('a', 256)

func getmessage(key string) (m interface{}, err error) {
	return message, nil
}

func BenchmarkGet(b *testing.B) {
	cache := New()
	cache.WithCallback(getmessage)
	for i := 0; i < b.N; i++ {
		cache.Get("123")
	}
}

func BenchmarkWriteToCache(b *testing.B) {
	b.Run("_size2^6", func(b *testing.B) {
		m := blob('a', 1<<6)
		writeToCache(b, m)
	})
	b.Run("_size2^8", func(b *testing.B) {
		m := blob('a', 1<<8)
		writeToCache(b, m)
	})
	b.Run("_size2^10", func(b *testing.B) {
		m := blob('a', 1<<10)
		writeToCache(b, m)
	})
	b.Run("_size2^15", func(b *testing.B) {
		m := blob('a', 1<<15)
		writeToCache(b, m)
	})
	b.Run("_size2^20", func(b *testing.B) {
		m := blob('a', 1<<20)
		writeToCache(b, m)
	})
}

func writeToCache(b *testing.B, data []byte) {
	cache := New()
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

func BenchmarkReadFromCache(b *testing.B) {
	readFromCache(b)
}

func readFromCache(b *testing.B) {
	cache := New()
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

func BenchmarkReadFromCacheNonExistentKeys(b *testing.B) {
	readFromCacheNonExistentKeys(b)
}

func readFromCacheNonExistentKeys(b *testing.B) {
	cache := New()
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
// go test -bench=BenchmarkRead1Min -benchtime=6s
func BenchmarkRead1Min(b *testing.B) {
	lens := []int{1, 10, 100, 1000, 10000}
	for _, l := range lens {
		b.Run(fmt.Sprintf("request %d", l), func(b *testing.B) {
			readFromCacheKKeys(b, l)
		})
	}
}

func readFromCacheKKeys(b *testing.B, l int) {
	keys := make([]int, l)
	for i := 0; i < l; i++ {
		keys[i] = i
	}
	cache := New()
	count := 0
	cache.WithCallback(func(key string) (m interface{}, err error) {
		count++
		return message, nil
	})
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		b.ReportAllocs()

		for pb.Next() {
			cache.Get(strconv.Itoa(keys[b.N%len(keys)]))
		}
		return
	})
	fmt.Println(count)
}
