package notifications

import (
	"container/heap"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/olympsis/models"
	"github.com/sirupsen/logrus"
)

func NewCarousel(l *logrus.Logger, callback func(*models.NotificationPushRequest) error) *Carousel {
	c := &Carousel{
		priorityQueue: make(PriorityQueue, 0),
		logger:        l,
		onProcessJob:  callback,
	}
	c.cond = sync.NewCond(&c.mu)
	heap.Init(&c.priorityQueue)
	return c
}

func (c *Carousel) AddJob(priority int, req models.NotificationPushRequest) error {
	if priority < 0 || priority > 5 {
		return errors.New("invalid priority: must be between 0 and 5")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
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
			for len(c.priorityQueue) == 0 {
				c.cond.Wait()
			}
			job := heap.Pop(&c.priorityQueue).(*Job)
			c.mu.Unlock()
			c.processJob(job)
		}
	}()
}

func (c *Carousel) processJob(job *Job) {
	c.logger.Info(fmt.Sprintf("Processing job: %s(%d)", job.ID, job.Priority))
	err := c.onProcessJob(&job.Request)
	if err != nil {
		c.logger.Error(fmt.Sprintf("Failed to process job: %s \nError: %s", job.ID, err.Error()))
		return
	}
}
