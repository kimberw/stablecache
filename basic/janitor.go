package basic

import (
	"fmt"
	"time"
)

type Janitor struct {
	Interval time.Duration
	stop     chan bool
}

func (j *Janitor) Run(del func()) {
	ticker := time.NewTicker(j.Interval)
	for {
		select {
		case <-ticker.C:
			del()
		case <-j.stop:
			ticker.Stop()
			return
		}
	}
}

func (j *Janitor) Stop() {
	fmt.Println("janitor stop")
	j.stop <- true
}

func NewJanitor(ci time.Duration, del func()) *Janitor {
	j := &Janitor{
		Interval: ci,
		stop:     make(chan bool),
	}
	go j.Run(del)
	return j
}
