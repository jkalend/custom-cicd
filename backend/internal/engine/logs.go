package engine

import (
	"sync"
	"time"
)

// LogEvent represents a single line or output chunk emitted during step execution.
type LogEvent struct {
	RunID     string `json:"run_id"`
	StepIndex int    `json:"step_index"`
	StepName  string `json:"step_name"`
	Stream    string `json:"stream"` // "stdout", "stderr", or "system"
	Line      string `json:"line"`
	Timestamp string `json:"timestamp"`
}

// LogBroadcaster manages real-time streaming and history buffers for active runs.
type LogBroadcaster struct {
	mu          sync.RWMutex
	history     map[string][]LogEvent
	subscribers map[string]map[chan LogEvent]struct{}
	maxHistory  int
}

// NewLogBroadcaster creates a broadcaster storing up to maxHistory entries per run.
func NewLogBroadcaster(maxHistory int) *LogBroadcaster {
	if maxHistory <= 0 {
		maxHistory = 1000
	}
	return &LogBroadcaster{
		history:     make(map[string][]LogEvent),
		subscribers: make(map[string]map[chan LogEvent]struct{}),
		maxHistory:  maxHistory,
	}
}

// Publish distributes an event to all active subscribers of the run and stores it in history.
func (b *LogBroadcaster) Publish(event LogEvent) {
	if event.Timestamp == "" {
		event.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Append to history
	hist := b.history[event.RunID]
	if len(hist) >= b.maxHistory {
		hist = hist[1:]
	}
	b.history[event.RunID] = append(hist, event)

	// Broadcast non-blocking to active subscribers
	subs := b.subscribers[event.RunID]
	for ch := range subs {
		select {
		case ch <- event:
		default:
			// If buffer is full, drop to prevent stalling publisher
		}
	}
}

// Subscribe returns historical events and a channel for live updates, plus an unsubscribe cleanup function.
func (b *LogBroadcaster) Subscribe(runID string) ([]LogEvent, <-chan LogEvent, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Copy history
	hist := b.history[runID]
	historyCopy := make([]LogEvent, len(hist))
	copy(historyCopy, hist)

	ch := make(chan LogEvent, 200)
	if b.subscribers[runID] == nil {
		b.subscribers[runID] = make(map[chan LogEvent]struct{})
	}
	b.subscribers[runID][ch] = struct{}{}

	unsubscribe := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if subs, ok := b.subscribers[runID]; ok {
			delete(subs, ch)
			close(ch)
			if len(subs) == 0 {
				delete(b.subscribers, runID)
			}
		}
	}

	return historyCopy, ch, unsubscribe
}

// ClearHistory frees retained log events for a deleted run.
func (b *LogBroadcaster) ClearHistory(runID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.history, runID)
}
