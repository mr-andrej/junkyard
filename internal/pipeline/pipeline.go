package pipeline

import (
	"errors"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mr-andrej/junkyard/internal/model"
)

// ErrQueueFull is returned by Enqueue when the bounded queue is at capacity.
// Receivers translate this into OTLP backpressure (RESOURCE_EXHAUSTED / 503).
var ErrQueueFull = errors.New("pipeline: queue full")

// ErrShutdown is returned by Enqueue after Shutdown has begun.
var ErrShutdown = errors.New("pipeline: shut down")

// Writer persists one batch of log entries. It is invoked from a single
// goroutine, so implementations don't need their own locking.
type Writer func([]model.LogEntry) error

// Pipeline is a bounded queue feeding a batcher and a single writer
// goroutine, per the v2 architecture (§4.2). The queue is a fixed-capacity
// buffered channel; enqueue never blocks.
type Pipeline struct {
	queue         chan model.LogEntry
	batchSize     int
	flushInterval time.Duration
	writer        Writer

	done    chan struct{}
	wg      sync.WaitGroup
	stopped atomic.Bool

	droppedBatches atomic.Int64
}

// New creates a pipeline and starts its writer goroutine.
func New(capacity, batchSize int, flushInterval time.Duration, writer Writer) *Pipeline {
	p := &Pipeline{
		queue:         make(chan model.LogEntry, capacity),
		batchSize:     batchSize,
		flushInterval: flushInterval,
		writer:        writer,
		done:          make(chan struct{}),
	}
	p.wg.Add(1)
	go p.run()
	return p
}

// Enqueue adds one entry without blocking. It returns ErrQueueFull when the
// queue is at capacity (the backpressure signal) or ErrShutdown after
// Shutdown has begun.
func (p *Pipeline) Enqueue(entry model.LogEntry) error {
	if p.stopped.Load() {
		return ErrShutdown
	}
	select {
	case p.queue <- entry:
		return nil
	default:
		return ErrQueueFull
	}
}

// EnqueueBatch adds a whole batch, rejecting it at the first entry that
// doesn't fit. Each rejection increments the dropped-batch counter.
func (p *Pipeline) EnqueueBatch(entries []model.LogEntry) error {
	for _, entry := range entries {
		if err := p.Enqueue(entry); err != nil {
			p.droppedBatches.Add(1)
			return err
		}
	}
	return nil
}

// DroppedBatches returns how many batches were rejected because the queue
// was full or the pipeline was shutting down.
func (p *Pipeline) DroppedBatches() int64 {
	return p.droppedBatches.Load()
}

// Shutdown stops accepting entries, drains the queue, flushes the final
// partial batch, and waits for the writer goroutine to finish.
func (p *Pipeline) Shutdown() {
	if p.stopped.Swap(true) {
		return
	}
	close(p.done)
	p.wg.Wait()
}

func (p *Pipeline) run() {
	defer p.wg.Done()

	batch := make([]model.LogEntry, 0, p.batchSize)
	timer := time.NewTimer(p.flushInterval)
	defer timer.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		if err := p.writer(batch); err != nil {
			log.Printf("pipeline: batch write failed, dropping %d entries: %v", len(batch), err)
		}
		batch = make([]model.LogEntry, 0, p.batchSize)
	}

	resetTimer := func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(p.flushInterval)
	}

	for {
		select {
		case entry := <-p.queue:
			batch = append(batch, entry)
			if len(batch) >= p.batchSize {
				flush()
				resetTimer()
			}
		case <-timer.C:
			flush()
			timer.Reset(p.flushInterval)
		case <-p.done:
			for {
				select {
				case entry := <-p.queue:
					batch = append(batch, entry)
					if len(batch) >= p.batchSize {
						flush()
					}
				default:
					flush()
					return
				}
			}
		}
	}
}
