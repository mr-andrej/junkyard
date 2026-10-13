package pipeline

import (
	"sync"
	"testing"
	"time"

	"github.com/mr-andrej/junkyard/internal/model"
)

// collector is a Writer that records every batch it receives.
type collector struct {
	mu      sync.Mutex
	batches [][]model.LogEntry
	notify  chan struct{}
}

func newCollector() *collector {
	return &collector{notify: make(chan struct{}, 100)}
}

func (c *collector) write(batch []model.LogEntry) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := make([]model.LogEntry, len(batch))
	copy(cp, batch)
	c.batches = append(c.batches, cp)
	c.notify <- struct{}{}
	return nil
}

func (c *collector) entries() []model.LogEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []model.LogEntry
	for _, b := range c.batches {
		out = append(out, b...)
	}
	return out
}

// waitBatches blocks until the collector has received at least n batches.
func (c *collector) waitBatches(t *testing.T, n int) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for i := 0; i < n; i++ {
		select {
		case <-c.notify:
		case <-deadline:
			t.Fatalf("timed out waiting for batch %d of %d", i+1, n)
		}
	}
}

func entry(body string) model.LogEntry {
	return model.LogEntry{Timestamp: time.Now(), Body: body}
}

func TestEnqueueOnFullQueueIsRejected(t *testing.T) {
	blocked := make(chan struct{})
	started := make(chan struct{})
	var once sync.Once

	// The writer blocks until the test ends, so whatever the writer
	// goroutine picks up never leaves the pipeline; everything else stays
	// queued. batchSize 1 makes the writer fire on the first entry.
	p := New(4, 1, time.Hour, func(batch []model.LogEntry) error {
		once.Do(func() { close(started) })
		<-blocked
		return nil
	})
	defer func() {
		close(blocked)
		p.Shutdown()
	}()

	// One entry goes to the blocked writer goroutine.
	if err := p.Enqueue(entry("writer")); err != nil {
		t.Fatalf("enqueue writer entry: %v", err)
	}
	<-started

	// Fill the queue to capacity.
	for i := 0; i < 4; i++ {
		if err := p.Enqueue(entry("queued")); err != nil {
			t.Fatalf("enqueue %d: %v", i, err)
		}
	}

	// The next enqueue must fail immediately, not block.
	done := make(chan error, 1)
	go func() { done <- p.Enqueue(entry("overflow")) }()
	select {
	case err := <-done:
		if err != ErrQueueFull {
			t.Fatalf("expected ErrQueueFull, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Enqueue blocked on a full queue")
	}
}

func TestBatchFlushes(t *testing.T) {
	tests := []struct {
		name          string
		batchSize     int
		flushInterval time.Duration
		enqueue       int
		wantBatchLen  int
	}{
		{"size threshold", 3, time.Hour, 3, 3},
		{"time threshold", 100, 20 * time.Millisecond, 2, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCollector()
			p := New(16, tt.batchSize, tt.flushInterval, c.write)
			defer p.Shutdown()

			for i := 0; i < tt.enqueue; i++ {
				if err := p.Enqueue(entry("x")); err != nil {
					t.Fatalf("enqueue %d: %v", i, err)
				}
			}

			c.waitBatches(t, 1)

			c.mu.Lock()
			got := len(c.batches[0])
			c.mu.Unlock()
			if got != tt.wantBatchLen {
				t.Fatalf("expected batch of %d, got %d", tt.wantBatchLen, got)
			}
		})
	}
}

func TestShutdownDrainsEverything(t *testing.T) {
	c := newCollector()
	p := New(64, 7, time.Hour, c.write)

	const total = 20
	for i := 0; i < total; i++ {
		if err := p.Enqueue(entry("x")); err != nil {
			t.Fatalf("enqueue %d: %v", i, err)
		}
	}

	p.Shutdown()

	if got := len(c.entries()); got != total {
		t.Fatalf("expected %d entries written after shutdown, got %d", total, got)
	}
}

func TestEnqueueAfterShutdown(t *testing.T) {
	c := newCollector()
	p := New(4, 2, time.Hour, c.write)
	p.Shutdown()

	if err := p.Enqueue(entry("late")); err != ErrShutdown {
		t.Fatalf("expected ErrShutdown, got %v", err)
	}
}

func TestDroppedBatchesCounted(t *testing.T) {
	blocked := make(chan struct{})
	started := make(chan struct{})
	var once sync.Once

	p := New(2, 1, time.Hour, func(batch []model.LogEntry) error {
		once.Do(func() { close(started) })
		<-blocked
		return nil
	})
	defer func() {
		close(blocked)
		p.Shutdown()
	}()

	if err := p.Enqueue(entry("writer")); err != nil {
		t.Fatalf("enqueue writer entry: %v", err)
	}
	<-started

	// Fill the queue (capacity 2), then submit a batch that doesn't fit.
	batch := []model.LogEntry{entry("a"), entry("b"), entry("c")}
	if err := p.EnqueueBatch(batch); err != ErrQueueFull {
		t.Fatalf("expected ErrQueueFull, got %v", err)
	}

	if got := p.DroppedBatches(); got != 1 {
		t.Fatalf("expected 1 dropped batch, got %d", got)
	}
}
