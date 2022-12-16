package basic

import (
	"errors"
	"math/rand"
)

type Color int8

const (
	defaultSize = 1000
)

const (
	black Color = iota
	white       = iota
)

type noCopy struct{}

func (*noCopy) Lock() {}

type Type int8

const (
	LRU Type = iota
	LFU      = iota
	ARC      = iota
)

var (
	NotFound = errors.New("not found")
	Timeout  = errors.New("timeout")
	Disuse   = errors.New("disuse")
)

func randfunc(t, d int64) bool {
	id := rand.Int63n(d * d * d)
	if id < t*t*t {
		return true
	}
	return false
}
