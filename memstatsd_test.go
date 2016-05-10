package memstatsd

import (
	"fmt"
	"testing"
	"time"
)

type statter struct{}

func (s statter) Timing(bucket string, d time.Duration) {
	fmt.Println(bucket, d)
}

func (s statter) Gauge(bucket string, value int) {
	fmt.Println(bucket, value)
}

func TestMemstatsd(t *testing.T) {
	msd := New("memstatsd.test.", statter{}, true)
	msd.Run(5 * time.Second)
	time.Sleep(time.Second * 10)

	go func() {
		time.Sleep(time.Minute)
	}()

	time.Sleep(time.Minute)
}
