package stablecache

import (
	"bytes"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func blob(char byte, len int) []byte {
	return bytes.Repeat([]byte{char}, len)
}

var message = blob('a', 256)

func getmessage(key string) (m []byte, err error) {
	drt := make([]byte, 256)
	copy(drt, message)
	drt = append(drt, []byte(key)...)
	return drt, nil
}

func TestCache(t *testing.T) {
	Convey("normal cache", t, func() {
		cache := New[string, []byte]("normal")
		cache.WithCallback(getmessage)
		key := "123"
		v, _ := getmessage(key)
		value, err := cache.Get(key)
		So(err, ShouldResemble, nil)
		So(value, ShouldResemble, v)
	})
}
