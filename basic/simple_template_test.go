package basic

import (
	"bytes"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTemplateCache(t *testing.T) {
	cache := NewTemplateCache[string, []byte]()
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

func blob2(char byte, len int) []byte {
	return bytes.Repeat([]byte{char}, len)
}

var message2 = blob2('a', 256)

func getmessage2(key string) (m []byte, err error) {
	return message2, nil
}

func BenchmarkGetTemplateCache(b *testing.B) {
	cache := NewTemplateCache[string, []byte]()
	cache.WithCallback(getmessage2)
	for i := 0; i < b.N; i++ {
		cache.Get("123")
	}
}

func BenchmarkWriteToTemplateCache(b *testing.B) {
	b.Run("_size2^6", func(b *testing.B) {
		m := blob2('a', 1<<6)
		writeToTemplateCache(b, m)
	})
	b.Run("_size2^8", func(b *testing.B) {
		m := blob2('a', 1<<8)
		writeToTemplateCache(b, m)
	})
	b.Run("_size2^10", func(b *testing.B) {
		m := blob2('a', 1<<10)
		writeToTemplateCache(b, m)
	})
	b.Run("_size2^15", func(b *testing.B) {
		m := blob2('a', 1<<15)
		writeToTemplateCache(b, m)
	})
	b.Run("_size2^20", func(b *testing.B) {
		m := blob2('a', 1<<20)
		writeToTemplateCache(b, m)
	})
}

func writeToTemplateCache(b *testing.B, data []byte) {
	cache := NewTemplateCache[string, []byte]()
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

func BenchmarkReadFromTemplateCache(b *testing.B) {
	readFromTemplateCache(b)
}

func readFromTemplateCache(b *testing.B) {
	cache := NewTemplateCache[string, []byte]()
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

func BenchmarkReadFromTemplateCacheNonExistentKeys(b *testing.B) {
	readFromTemplateCacheNonExistentKeys(b)
}

func readFromTemplateCacheNonExistentKeys(b *testing.B) {
	cache := NewTemplateCache[string, []byte]()
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
// go test -bench=BenchmarkReadTemplateCache -benchtime=6s
func BenchmarkReadTemplateCache(b *testing.B) {
	lens := []int{1, 10, 100, 1000, 10000}
	for _, l := range lens {
		b.Run(fmt.Sprintf("request %d", l), func(b *testing.B) {
			readFromTemplateCacheKKeys(b, l)
		})
	}
}

func readFromTemplateCacheKKeys(b *testing.B, l int) {
	keys := make([]int, l)
	for i := 0; i < l; i++ {
		keys[i] = i
	}
	cache := NewTemplateCache[string, []byte]()
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
