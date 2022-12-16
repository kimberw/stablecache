package basic

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestPartitionCache(t *testing.T) {
	cache := NewPartitionCache[string, []byte]()
	cache.WithCallback(getmessage2)

	Convey(fmt.Sprintf("key %v expect %v", "123", "value_123"), t, func() {
		// So(reverse1(item.from, item.to), ShouldResemble, item.result)
		// So(reverse2(item.from, item.to), ShouldResemble, item.result)
		key := "123"
		value, err := cache.Get(key)
		So(err, ShouldResemble, nil)
		So(value, ShouldResemble, message2)
	})
}

func BenchmarkGetPartitionCache(b *testing.B) {
	cache := NewPartitionCache[string, []byte]()
	cache.WithCallback(getmessage2)
	for i := 0; i < b.N; i++ {
		cache.Get("123")
	}
}

func BenchmarkWriteToPartitionCache(b *testing.B) {
	b.Run("_size2^6", func(b *testing.B) {
		m := blob2('a', 1<<6)
		writeToPartitionCache(b, m)
	})
	b.Run("_size2^8", func(b *testing.B) {
		m := blob2('a', 1<<8)
		writeToPartitionCache(b, m)
	})
	b.Run("_size2^10", func(b *testing.B) {
		m := blob2('a', 1<<10)
		writeToPartitionCache(b, m)
	})
	b.Run("_size2^15", func(b *testing.B) {
		m := blob2('a', 1<<15)
		writeToPartitionCache(b, m)
	})
	b.Run("_size2^20", func(b *testing.B) {
		m := blob2('a', 1<<20)
		writeToPartitionCache(b, m)
	})
}

func writeToPartitionCache(b *testing.B, data []byte) {
	cache := NewPartitionCache[string, []byte]()
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

func BenchmarkReadFromPartitionCache(b *testing.B) {
	readFromPartitionCache(b)
}

func readFromPartitionCache(b *testing.B) {
	cache := NewPartitionCache[string, []byte]()
	cache.WithCallback(getmessage2)
	for i := 0; i < b.N; i++ {
		cache.SetWithExp(strconv.Itoa(i), message2, 100*time.Second)
	}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		b.ReportAllocs()

		for pb.Next() {
			cache.Get(strconv.Itoa(rand.Intn(b.N)))
		}
	})
}

func BenchmarkReadFromPartitionCacheNonExistentKeys(b *testing.B) {
	readFromPartitionCacheNonExistentKeys(b)
}

func readFromPartitionCacheNonExistentKeys(b *testing.B) {
	cache := NewPartitionCache[string, []byte]()
	cache.WithCallback(getmessage2)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		b.ReportAllocs()

		for pb.Next() {
			cache.Get(strconv.Itoa(rand.Intn(b.N)))
		}
	})
}

// 运行 1分钟 1个key 请求 穿透下游次数
// go test -bench=BenchmarkReadPartitionCache -benchtime=6s
func BenchmarkReadPartitionCache(b *testing.B) {
	lens := []int{1, 10, 100, 1000, 10000}
	for _, l := range lens {
		b.Run(fmt.Sprintf("request %d", l), func(b *testing.B) {
			readFromPartitionCacheKKeys(b, l)
		})
	}
}

func readFromPartitionCacheKKeys(b *testing.B, l int) {
	keys := make([]int, l)
	for i := 0; i < l; i++ {
		keys[i] = i
	}
	cache := NewPartitionCache[string, []byte]()
	cache.WithCallback(getmessage2)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		b.ReportAllocs()

		for pb.Next() {
			cache.Get(strconv.Itoa(keys[b.N%len(keys)]))
		}
		return
	})
}
