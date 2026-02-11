/*
Copyright 2025. The Jumpstarter Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package host

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// OutputBuffer collects output for a single host, allowing atomic flush to stdout.
type OutputBuffer struct {
	buf           bytes.Buffer
	hostName      string
	hasChanges    bool
	hasErrors     bool
	instanceCount int
	retryItems    []RetryItem
	startTime     time.Time
	duration      time.Duration
}

// NewOutputBuffer creates a new OutputBuffer for the given host.
func NewOutputBuffer(hostName string, instanceCount int) *OutputBuffer {
	return &OutputBuffer{
		hostName:      hostName,
		instanceCount: instanceCount,
		startTime:     time.Now(),
	}
}

// Done records the completion time for this host.
func (o *OutputBuffer) Done() {
	o.duration = time.Since(o.startTime)
}

// Printf writes formatted output to the buffer.
func (o *OutputBuffer) Printf(format string, args ...any) {
	_, _ = fmt.Fprintf(&o.buf, format, args...)
}

// Writer returns the underlying writer for passing to subsystems.
func (o *OutputBuffer) Writer() io.Writer {
	return &o.buf
}

// MarkChanged marks this host as having changes.
func (o *OutputBuffer) MarkChanged() {
	o.hasChanges = true
}

// MarkError marks this host as having errors.
func (o *OutputBuffer) MarkError() {
	o.hasErrors = true
}

// AddRetryItem adds a retry item to this buffer's collection.
func (o *OutputBuffer) AddRetryItem(item RetryItem) {
	o.retryItems = append(o.retryItems, item)
}

// SyncPrinter handles synchronized output to stdout from multiple goroutines.
type SyncPrinter struct {
	mu             sync.Mutex
	startTime      time.Time
	okCount        atomic.Int32
	changedCount   atomic.Int32
	failedCount    atomic.Int32
	totalInstances atomic.Int32
	retryCount     atomic.Int32
	retrySuccess   atomic.Int32
	failedHosts    []string
}

// NewSyncPrinter creates a new SyncPrinter.
func NewSyncPrinter() *SyncPrinter {
	return &SyncPrinter{
		startTime: time.Now(),
	}
}

// AddRetryStats records retry queue statistics.
func (p *SyncPrinter) AddRetryStats(queued, succeeded int32) {
	p.retryCount.Add(queued)
	p.retrySuccess.Add(succeeded)
}

// FlushBuffer atomically writes a host's buffered output to stdout.
// Hosts with no changes get a compact one-line summary.
// Hosts with changes or errors show their full buffered output.
func (p *SyncPrinter) FlushBuffer(ob *OutputBuffer) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.totalInstances.Add(int32(ob.instanceCount))

	if ob.hasErrors {
		p.failedCount.Add(1)
		p.failedHosts = append(p.failedHosts, ob.hostName)
		// Print full output for hosts with errors
		_, _ = fmt.Fprint(os.Stdout, ob.buf.String())
	} else if ob.hasChanges {
		p.changedCount.Add(1)
		// Print full output for hosts with changes
		_, _ = fmt.Fprint(os.Stdout, ob.buf.String())
	} else {
		p.okCount.Add(1)
		// Compact one-liner for hosts with no changes
		_, _ = fmt.Fprintf(os.Stdout, "  ✅ %s (%d instances, no changes) [%s]\n", ob.hostName, ob.instanceCount, formatDuration(ob.duration))
	}
}

// PrintSummary prints a final summary of all host processing results.
func (p *SyncPrinter) PrintSummary() {
	p.mu.Lock()
	defer p.mu.Unlock()

	elapsed := time.Since(p.startTime)
	totalHosts := p.okCount.Load() + p.changedCount.Load() + p.failedCount.Load()
	if totalHosts == 0 {
		return
	}

	_, _ = fmt.Fprintf(os.Stdout, "\n📊 Summary\n")
	_, _ = fmt.Fprintf(os.Stdout, "  Hosts:      %d processed (%d ok, %d changed, %d failed)\n",
		totalHosts, p.okCount.Load(), p.changedCount.Load(), p.failedCount.Load())
	_, _ = fmt.Fprintf(os.Stdout, "  Instances:  %d total\n", p.totalInstances.Load())

	if p.retryCount.Load() > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "  Retries:    %d queued, %d succeeded, %d gave up\n",
			p.retryCount.Load(), p.retrySuccess.Load(), p.retryCount.Load()-p.retrySuccess.Load())
	}

	_, _ = fmt.Fprintf(os.Stdout, "  Runtime:    %s\n", formatDuration(elapsed))

	if len(p.failedHosts) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "  Failed hosts:\n")
		for _, host := range p.failedHosts {
			_, _ = fmt.Fprintf(os.Stdout, "    ❌ %s\n", host)
		}
	}
}

// formatDuration formats a duration into a human-friendly string.
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%ds", minutes, seconds)
}
