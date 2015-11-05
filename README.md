## memstatsd

Package memstatsd implements a simple to use reporting tool that sends actual runtime.MemStats values and their diffs to a statsd server.

### Example

```
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

    time.Sleep(time.Minute)
}
```