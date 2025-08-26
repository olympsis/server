package notifications

import (
	"github.com/olympsis/models"
)

type Job struct {
	ID       string
	Index    int
	Priority int
	Request  models.NotificationPushRequest
}

type PriorityQueue []*Job

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// Higher priority jobs come first.
	return pq[i].Priority > pq[j].Priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	job := x.(*Job)
	job.Index = n
	*pq = append(*pq, job)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	job := old[n-1]
	job.Index = -1 // for safety
	*pq = old[0 : n-1]
	return job
}
