package notifications

import (
	"container/heap"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/google/uuid"
	"github.com/olympsis/models"
	"github.com/sirupsen/logrus"
)

// defaultMaxQueueSize bounds the carousel's in-memory job queue so a stalled
// processor (e.g. a slow APNS push) can't let the queue grow without limit and
// leak memory. Override with the NOTIFICATION_QUEUE_MAX_SIZE env var.
const defaultMaxQueueSize = 1000

// resolveMaxQueueSize reads the queue bound from NOTIFICATION_QUEUE_MAX_SIZE,
// falling back to defaultMaxQueueSize when the var is unset or invalid.
func resolveMaxQueueSize(l *logrus.Logger) int {
	raw := os.Getenv("NOTIFICATION_QUEUE_MAX_SIZE")
	if raw == "" {
		return defaultMaxQueueSize
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		l.Warnf("Invalid NOTIFICATION_QUEUE_MAX_SIZE %q; using default %d", raw, defaultMaxQueueSize)
		return defaultMaxQueueSize
	}
	return n
}

func NewCarousel(l *logrus.Logger, callback func(*models.NotificationPushRequest) error) *Carousel {
	c := &Carousel{
		priorityQueue: make(PriorityQueue, 0),
		logger:        l,
		onProcessJob:  callback,
		maxQueueSize:  resolveMaxQueueSize(l),
		done:          make(chan struct{}),
	}
	c.cond = sync.NewCond(&c.mu)
	heap.Init(&c.priorityQueue)
	l.Infof("Carousel queue bound set to %d jobs", c.maxQueueSize)
	return c
}

func (c *Carousel) AddJob(priority int, req models.NotificationPushRequest) error {
	if priority < 0 || priority > 5 {
		return errors.New("invalid priority: must be between 0 and 5")
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	// Reject new work once shutting down or at capacity — backpressure instead
	// of unbounded memory growth (no silent drops; the rejection is logged and
	// returned to the caller).
	if c.stopped {
		return errors.New("carousel is shutting down; job rejected")
	}
	if len(c.priorityQueue) >= c.maxQueueSize {
		c.logger.Warnf("Carousel queue full (%d/%d); rejecting job", len(c.priorityQueue), c.maxQueueSize)
		return errors.New("notification queue is full")
	}

	job := &Job{ID: uuid.New().String(), Priority: priority, Request: req}
	heap.Push(&c.priorityQueue, job)
	c.cond.Signal()
	return nil
}

func (c *Carousel) RemoveJob() *Job {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.priorityQueue) == 0 {
		return nil
	}
	job := heap.Pop(&c.priorityQueue).(*Job)
	return job
}

func (c *Carousel) Start() {
	c.logger.Info("Starting Carousel...")
	go func() {
		for {
			c.mu.Lock()
			// Wait for work, but also wake when asked to stop.
			for len(c.priorityQueue) == 0 && !c.stopped {
				c.cond.Wait()
			}
			// Once stopped, finish draining the queue, then exit cleanly so the
			// goroutine doesn't leak and queued notifications aren't dropped.
			if len(c.priorityQueue) == 0 && c.stopped {
				c.mu.Unlock()
				c.logger.Info("Carousel stopped.")
				close(c.done)
				return
			}
			job := heap.Pop(&c.priorityQueue).(*Job)
			c.mu.Unlock()
			c.processJob(job)
		}
	}()
}

// Stop signals the worker to drain the queue and exit, then blocks until it has
// finished. Subsequent AddJob calls are rejected. Idempotent.
func (c *Carousel) Stop() {
	c.mu.Lock()
	if c.stopped {
		c.mu.Unlock()
		return
	}
	c.stopped = true
	c.mu.Unlock()
	c.cond.Broadcast() // wake the worker if it's parked on an empty queue
	<-c.done           // wait for drain + clean exit
}

func (c *Carousel) processJob(job *Job) {
	c.logger.Info(fmt.Sprintf("Processing job: %s(%d)", job.ID, job.Priority))
	err := c.onProcessJob(&job.Request)
	if err != nil {
		c.logger.Error(fmt.Sprintf("Failed to process job: %s \nError: %s", job.ID, err.Error()))
		return
	}
}
