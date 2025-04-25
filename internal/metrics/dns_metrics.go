package metrics

import (
	"fmt"
	"sync"
	"time"
)

type DNSMetrics struct {
	FullNodeDirectResolution     []time.Duration
	LightNodeForwardedResolution []time.Duration
	mutex                        sync.RWMutex
}

var (
	instance *DNSMetrics
	once     sync.Once
)

func GetDNSMetrics() *DNSMetrics {
	once.Do(func() {
		instance = &DNSMetrics{
			FullNodeDirectResolution:     make([]time.Duration, 0),
			LightNodeForwardedResolution: make([]time.Duration, 0),
		}
	})
	return instance
}

func (m *DNSMetrics) AddFullNodeDirectResolution(duration time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.FullNodeDirectResolution = append(m.FullNodeDirectResolution, duration)
}

func (m *DNSMetrics) AddLightNodeForwardedResolution(duration time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.LightNodeForwardedResolution = append(m.LightNodeForwardedResolution, duration)
}

func (m *DNSMetrics) GetAverageLatencies() (time.Duration, time.Duration) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	fullNodeAvg := calculateAverage(m.FullNodeDirectResolution)
	lightNodeAvg := calculateAverage(m.LightNodeForwardedResolution)

	return fullNodeAvg, lightNodeAvg
}

func calculateAverage(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	var sum time.Duration
	for _, d := range durations {
		sum += d
	}
	return sum / time.Duration(len(durations))
}

func (m *DNSMetrics) PrintMetrics() {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	fmt.Printf("Avg Full Node Direct Lookup: %v\n", calculateAverage(m.FullNodeDirectResolution))
	fmt.Printf("Avg Light Node Forwarded Lookup: %v\n", calculateAverage(m.LightNodeForwardedResolution))
}
