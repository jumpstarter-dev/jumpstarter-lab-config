package host

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jumpstarter-dev/jumpstarter-lab-config/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeadAnnotationFiltering(t *testing.T) {
	tests := []struct {
		name                   string
		exporterInstances      []*v1alpha1.ExporterInstance
		expectedAliveInstances int
		expectedDeadInstances  int
	}{
		{
			name: "no dead instances",
			exporterInstances: []*v1alpha1.ExporterInstance{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "instance1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "instance2",
					},
				},
			},
			expectedAliveInstances: 2,
			expectedDeadInstances:  0,
		},
		{
			name: "one dead instance",
			exporterInstances: []*v1alpha1.ExporterInstance{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "instance1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "instance2",
						Annotations: map[string]string{
							"dead": "true",
						},
					},
				},
			},
			expectedAliveInstances: 1,
			expectedDeadInstances:  1,
		},
		{
			name: "all dead instances",
			exporterInstances: []*v1alpha1.ExporterInstance{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "instance1",
						Annotations: map[string]string{
							"dead": "true",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "instance2",
						Annotations: map[string]string{
							"dead": "true",
						},
					},
				},
			},
			expectedAliveInstances: 0,
			expectedDeadInstances:  2,
		},
		{
			name: "dead annotation with false value",
			exporterInstances: []*v1alpha1.ExporterInstance{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "instance1",
						Annotations: map[string]string{
							"dead": "false",
						},
					},
				},
			},
			expectedAliveInstances: 0,
			expectedDeadInstances:  1,
		},
		{
			name: "dead annotation with other value",
			exporterInstances: []*v1alpha1.ExporterInstance{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "instance1",
						Annotations: map[string]string{
							"dead": "maybe",
						},
					},
				},
			},
			expectedAliveInstances: 0,
			expectedDeadInstances:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the filtering logic from SyncExporterHosts
			aliveInstances := []*v1alpha1.ExporterInstance{}
			deadCount := 0

			for _, exporterInstance := range tt.exporterInstances {
				if _, exists := exporterInstance.Annotations["dead"]; exists {
					deadCount++
				} else {
					aliveInstances = append(aliveInstances, exporterInstance)
				}
			}

			assert.Equal(t, tt.expectedAliveInstances, len(aliveInstances), "unexpected number of alive instances")
			assert.Equal(t, tt.expectedDeadInstances, deadCount, "unexpected number of dead instances")
		})
	}
}

func TestHostSkippingBehavior(t *testing.T) {
	tests := []struct {
		name              string
		exporterInstances []*v1alpha1.ExporterInstance
		shouldSkipHost    bool
		expectedAlivCount int
		description       string
	}{
		{
			name: "mixed dead and alive instances - should NOT skip host",
			exporterInstances: []*v1alpha1.ExporterInstance{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "alive-instance",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "dead-instance",
						Annotations: map[string]string{
							"dead": "true",
						},
					},
				},
			},
			shouldSkipHost:    false, // Host should be processed because there's an alive instance
			expectedAlivCount: 1,
			description:       "When there are both dead and alive instances, host should be processed",
		},
		{
			name: "all instances dead - should SKIP host",
			exporterInstances: []*v1alpha1.ExporterInstance{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "dead-instance-1",
						Annotations: map[string]string{
							"dead": "true",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "dead-instance-2",
						Annotations: map[string]string{
							"dead": "true",
						},
					},
				},
			},
			shouldSkipHost:    true, // Host should be skipped because all instances are dead
			expectedAlivCount: 0,
			description:       "When all instances are dead, host should be skipped entirely",
		},
		{
			name: "all instances alive - should NOT skip host",
			exporterInstances: []*v1alpha1.ExporterInstance{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "alive-instance-1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "alive-instance-2",
					},
				},
			},
			shouldSkipHost:    false, // Host should be processed because all instances are alive
			expectedAlivCount: 2,
			description:       "When all instances are alive, host should be processed normally",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the new logic from SyncExporterHosts
			// Check if all instances are dead
			allDead := true
			aliveCount := 0

			for _, exporterInstance := range tt.exporterInstances {
				if _, exists := exporterInstance.Annotations["dead"]; !exists {
					allDead = false
					aliveCount++
				}
			}

			// Host is skipped only if all instances are dead (and there are instances)
			hostWouldBeSkipped := len(tt.exporterInstances) > 0 && allDead

			assert.Equal(t, tt.shouldSkipHost, hostWouldBeSkipped, tt.description)
			assert.Equal(t, tt.expectedAlivCount, aliveCount, "unexpected number of alive instances")
		})
	}
}

func TestOutputBuffer(t *testing.T) {
	t.Run("Printf writes to buffer", func(t *testing.T) {
		out := NewOutputBuffer("host-1", 3)
		out.Printf("hello %s", "world")
		out.Printf(" number %d", 42)

		assert.Equal(t, "hello world number 42", out.buf.String())
	})

	t.Run("Writer returns buffer writer", func(t *testing.T) {
		out := NewOutputBuffer("host-1", 2)
		w := out.Writer()
		_, _ = fmt.Fprintf(w, "via writer")

		assert.Equal(t, "via writer", out.buf.String())
	})

	t.Run("MarkChanged and MarkError are independent", func(t *testing.T) {
		out := NewOutputBuffer("host-1", 1)
		assert.False(t, out.hasChanges)
		assert.False(t, out.hasErrors)

		out.MarkChanged()
		assert.True(t, out.hasChanges)
		assert.False(t, out.hasErrors)

		out.MarkError()
		assert.True(t, out.hasChanges)
		assert.True(t, out.hasErrors)
	})

	t.Run("AddRetryItem collects items", func(t *testing.T) {
		out := NewOutputBuffer("host-1", 2)
		assert.Empty(t, out.retryItems)

		out.AddRetryItem(RetryItem{HostName: "host-1", Attempts: 1})
		out.AddRetryItem(RetryItem{HostName: "host-1", Attempts: 1})

		assert.Len(t, out.retryItems, 2)
	})

	t.Run("Done records duration", func(t *testing.T) {
		out := NewOutputBuffer("host-1", 1)
		time.Sleep(10 * time.Millisecond)
		out.Done()

		assert.Greater(t, out.duration, time.Duration(0))
	})
}

func TestSyncPrinterCounters(t *testing.T) {
	t.Run("FlushBuffer tracks ok, changed, failed counts", func(t *testing.T) {
		printer := NewSyncPrinter()

		okBuf := NewOutputBuffer("ok-host", 2)
		okBuf.Done()
		printer.FlushBuffer(okBuf)

		changedBuf := NewOutputBuffer("changed-host", 3)
		changedBuf.MarkChanged()
		changedBuf.Done()
		printer.FlushBuffer(changedBuf)

		failedBuf := NewOutputBuffer("failed-host", 1)
		failedBuf.MarkError()
		failedBuf.Done()
		printer.FlushBuffer(failedBuf)

		assert.Equal(t, int32(1), printer.okCount.Load())
		assert.Equal(t, int32(1), printer.changedCount.Load())
		assert.Equal(t, int32(1), printer.failedCount.Load())
		assert.Equal(t, int32(6), printer.totalInstances.Load())
		assert.Equal(t, []string{"failed-host"}, printer.failedHosts)
	})

	t.Run("error takes precedence over changed", func(t *testing.T) {
		printer := NewSyncPrinter()

		buf := NewOutputBuffer("both-host", 1)
		buf.MarkChanged()
		buf.MarkError()
		buf.Done()
		printer.FlushBuffer(buf)

		// Should count as failed, not changed
		assert.Equal(t, int32(0), printer.okCount.Load())
		assert.Equal(t, int32(0), printer.changedCount.Load())
		assert.Equal(t, int32(1), printer.failedCount.Load())
	})

	t.Run("AddRetryStats accumulates", func(t *testing.T) {
		printer := NewSyncPrinter()
		printer.AddRetryStats(5, 3)
		printer.AddRetryStats(2, 1)

		assert.Equal(t, int32(7), printer.retryCount.Load())
		assert.Equal(t, int32(4), printer.retrySuccess.Load())
	})
}

func TestSyncPrinterConcurrentFlush(t *testing.T) {
	printer := NewSyncPrinter()
	var wg sync.WaitGroup

	// Flush 50 buffers concurrently to verify no races
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			buf := NewOutputBuffer(fmt.Sprintf("host-%d", idx), idx%5+1)
			if idx%3 == 0 {
				buf.MarkChanged()
			}
			if idx%7 == 0 {
				buf.MarkError()
			}
			buf.Done()
			printer.FlushBuffer(buf)
		}(i)
	}
	wg.Wait()

	total := printer.okCount.Load() + printer.changedCount.Load() + printer.failedCount.Load()
	assert.Equal(t, int32(50), total, "all 50 buffers should be counted")
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{0, "0ms"},
		{500 * time.Millisecond, "500ms"},
		{999 * time.Millisecond, "999ms"},
		{1500 * time.Millisecond, "1.5s"},
		{30 * time.Second, "30.0s"},
		{90 * time.Second, "1m30s"},
		{5*time.Minute + 15*time.Second, "5m15s"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatDuration(tt.duration))
		})
	}
}
