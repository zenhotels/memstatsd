// Package memstatsd implements a simple to use reporting tool that sends
// actual runtime.MemStats values and their diffs to a statsd server.
package memstatsd

import (
	"fmt"
	"runtime"
	"time"
)

type Statter interface {
	Timing(bucket string, d time.Duration)
	Gauge(bucket string, value int)
}

type MemStatsd struct {
	prefix string
	statsd Statter
	debug  bool

	previous     *MemStats
	allocLatency time.Duration
}

func New(prefix string, statsd Statter, debug ...bool) MemStatsd {
	m := MemStatsd{
		prefix: prefix,
		statsd: statsd,
	}
	if len(debug) > 0 && debug[0] {
		m.debug = true
	}
	return m
}

func (m *MemStatsd) Run(d time.Duration) {
	t := time.NewTicker(d)
	go func() {
		for range t.C {
			m.pushMemStats()
		}
	}()

	t2 := time.NewTicker(d)
	go func() {
		for range t2.C {
			m.pushAllocLatency()
		}
	}()
}

func (m *MemStatsd) pushMemStats() {
	stats, delta := m.snapshotMemStats()
	if m.debug {
		fmt.Println("pushMemStats @", time.Now())
	}

	m.statsd.Gauge(m.prefix+"alloc", int(stats.Alloc))
	m.statsd.Gauge(m.prefix+"total_alloc", int(stats.TotalAlloc))
	m.statsd.Gauge(m.prefix+"sys", int(stats.Sys))
	m.statsd.Gauge(m.prefix+"lookups", int(stats.Lookups))
	m.statsd.Gauge(m.prefix+"mallocs", int(stats.Mallocs))
	m.statsd.Gauge(m.prefix+"frees", int(stats.Frees))
	m.statsd.Gauge(m.prefix+"heap_alloc", int(stats.HeapAlloc))
	m.statsd.Gauge(m.prefix+"heap_sys", int(stats.HeapSys))
	m.statsd.Gauge(m.prefix+"heap_idle", int(stats.HeapIdle))
	m.statsd.Gauge(m.prefix+"heap_inuse", int(stats.HeapInuse))
	m.statsd.Gauge(m.prefix+"heap_released", int(stats.HeapReleased))
	m.statsd.Gauge(m.prefix+"heap_objects", int(stats.HeapObjects))
	m.statsd.Gauge(m.prefix+"num_gc", int(stats.NumGC))
	m.statsd.Timing(m.prefix+"pause_gc", stats.PauseGC)

	m.statsd.Gauge(m.prefix+"alloc.delta", int(delta.Alloc))
	m.statsd.Gauge(m.prefix+"total_alloc.delta", int(delta.TotalAlloc))
	m.statsd.Gauge(m.prefix+"sys.delta", int(delta.Sys))
	m.statsd.Gauge(m.prefix+"lookups.delta", int(delta.Lookups))
	m.statsd.Gauge(m.prefix+"mallocs.delta", int(delta.Mallocs))
	m.statsd.Gauge(m.prefix+"frees.delta", int(delta.Frees))
	m.statsd.Gauge(m.prefix+"heap_alloc.delta", int(delta.HeapAlloc))
	m.statsd.Gauge(m.prefix+"heap_sys.delta", int(delta.HeapSys))
	m.statsd.Gauge(m.prefix+"heap_idle.delta", int(delta.HeapIdle))
	m.statsd.Gauge(m.prefix+"heap_inuse.delta", int(delta.HeapInuse))
	m.statsd.Gauge(m.prefix+"heap_released.delta", int(delta.HeapReleased))
	m.statsd.Gauge(m.prefix+"heap_objects.delta", int(delta.HeapObjects))
	m.statsd.Gauge(m.prefix+"num_gc.delta", int(delta.NumGC))

	m.statsd.Timing(m.prefix+"pause_gc.delta", delta.PauseGC)
}

func (m *MemStatsd) pushAllocLatency() {
	latency, delta := m.snapshotAllocLatency()
	if m.debug {
		fmt.Println("pushAllocLatency @", time.Now())
	}

	m.statsd.Timing(m.prefix+"alloc_latency", latency)
	m.statsd.Timing(m.prefix+"alloc_latency.delta", delta)
}

type MemStats struct {
	// General stats
	Alloc      uint64 // bytes allocated and not yet freed
	TotalAlloc uint64 // bytes allocated (even if freed)
	Sys        uint64 // bytes obtained from system (sum of XxxSys below)
	Lookups    uint64 // number of pointer lookups
	Mallocs    uint64 // number of mallocs
	Frees      uint64 // number of frees

	// Heap stats
	HeapAlloc    uint64 // bytes allocated and not yet freed (same as Alloc above)
	HeapSys      uint64 // bytes obtained from system
	HeapIdle     uint64 // bytes in idle spans
	HeapInuse    uint64 // bytes in non-idle span
	HeapReleased uint64 // bytes released to the OS
	HeapObjects  uint64 // total number of allocated objects

	// GC stats
	NumGC   uint32
	PauseGC time.Duration
}

func (m *MemStatsd) snapshotAllocLatency() (latency time.Duration, delta time.Duration) {
	const wait = 100 * time.Millisecond
	const size = 10 * 1024

	start := time.Now()
	var _ = make([]byte, size)
	time.Sleep(wait)
	var _ = make([]byte, size)
	latency = time.Since(start) - wait
	if m.allocLatency == 0 {
		m.allocLatency = latency
		return
	}
	delta = latency - m.allocLatency
	m.allocLatency = latency
	return
}

func (m *MemStatsd) snapshotMemStats() (latest *MemStats, delta MemStats) {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	latest = &MemStats{
		Alloc:        stats.Alloc,
		TotalAlloc:   stats.TotalAlloc,
		Sys:          stats.Sys,
		Lookups:      stats.Lookups,
		Mallocs:      stats.Mallocs,
		Frees:        stats.Frees,
		HeapAlloc:    stats.HeapAlloc,
		HeapSys:      stats.HeapSys,
		HeapIdle:     stats.HeapIdle,
		HeapInuse:    stats.HeapInuse,
		HeapReleased: stats.HeapReleased,
		HeapObjects:  stats.HeapObjects,
		NumGC:        stats.NumGC,
		PauseGC:      time.Duration(stats.PauseNs[(stats.NumGC+255)%256]),
	}

	if m.previous == nil {
		m.previous = latest
		return
	}
	delta = MemStats{
		Alloc:        latest.Alloc - m.previous.Alloc,
		TotalAlloc:   latest.TotalAlloc - m.previous.TotalAlloc,
		Sys:          latest.Sys - m.previous.Sys,
		Lookups:      latest.Lookups - m.previous.Lookups,
		Mallocs:      latest.Mallocs - m.previous.Mallocs,
		Frees:        latest.Frees - m.previous.Frees,
		HeapAlloc:    latest.HeapAlloc - m.previous.HeapAlloc,
		HeapSys:      latest.HeapSys - m.previous.HeapSys,
		HeapIdle:     latest.HeapIdle - m.previous.HeapIdle,
		HeapInuse:    latest.HeapInuse - m.previous.HeapInuse,
		HeapReleased: latest.HeapReleased - m.previous.HeapReleased,
		HeapObjects:  latest.HeapObjects - m.previous.HeapObjects,
		NumGC:        latest.NumGC - m.previous.NumGC,
		PauseGC:      latest.PauseGC - m.previous.PauseGC,
	}
	m.previous = latest
	return
}
