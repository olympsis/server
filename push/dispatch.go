package push

import (
	"container/heap"
	"errors"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/olympsis/models"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// defaultMaxQueueSize bounds the in-memory job queue so a stalled sender can't
// let the queue grow without limit. Override with NOTIFICATION_QUEUE_MAX_SIZE
// (shared with the legacy carousel — same operational knob).
const defaultMaxQueueSize = 1000

// prefKeyFor maps a device platform to the NotificationPreference.Types key that
// gates push for it. iOS uses the long-standing "push" master toggle; Android
// uses its own "android" key (the Android client must set it to true).
var prefKeyFor = map[string]string{
	"ios":     "push",
	"android": "android",
}

// job is a single notification to fan out. Recipients come from EITHER a topic
// name (resolved to its members) OR an explicit user-id list, never both.
type job struct {
	id    string
	prio  int
	note  EventNote
	topic string
	users []string
	index int // maintained by the heap
}

// dispatcher is the push worker: a bounded priority queue drained by one
// goroutine, mirroring the proven carousel shape but Encoder/sender-based and
// with dead-token deactivation.
type dispatcher struct {
	mu      sync.Mutex
	cond    *sync.Cond
	queue   pushQueue
	maxSize int
	stopped bool
	done    chan struct{}

	repo    repo
	senders map[string]sender
	logger  *logrus.Logger
}

func newDispatcher(r repo, senders map[string]sender, l *logrus.Logger) *dispatcher {
	d := &dispatcher{
		queue:   make(pushQueue, 0),
		maxSize: resolveMaxQueueSize(l),
		done:    make(chan struct{}),
		repo:    r,
		senders: senders,
		logger:  l,
	}
	d.cond = sync.NewCond(&d.mu)
	heap.Init(&d.queue)
	return d
}

func resolveMaxQueueSize(l *logrus.Logger) int {
	raw := os.Getenv("NOTIFICATION_QUEUE_MAX_SIZE")
	if raw == "" {
		return defaultMaxQueueSize
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		l.Warnf("[Push] invalid NOTIFICATION_QUEUE_MAX_SIZE %q; using default %d", raw, defaultMaxQueueSize)
		return defaultMaxQueueSize
	}
	return n
}

// add enqueues a job, applying backpressure: rejected when shutting down or at
// capacity (logged + returned, never silently dropped).
func (d *dispatcher) add(j *job) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.stopped {
		return errors.New("[Push] shutting down; job rejected")
	}
	if len(d.queue) >= d.maxSize {
		d.logger.Warnf("[Push] queue full (%d/%d); rejecting job", len(d.queue), d.maxSize)
		return errors.New("push queue is full")
	}

	j.id = uuid.New().String()
	heap.Push(&d.queue, j)
	d.cond.Signal()
	return nil
}

func (d *dispatcher) start() {
	go func() {
		for {
			d.mu.Lock()
			for len(d.queue) == 0 && !d.stopped {
				d.cond.Wait()
			}
			// Once stopped, finish draining before exiting so queued
			// notifications aren't dropped and the goroutine doesn't leak.
			if len(d.queue) == 0 && d.stopped {
				d.mu.Unlock()
				d.logger.Info("[Push] Teardown...")
				close(d.done)
				return
			}
			j := heap.Pop(&d.queue).(*job)
			d.mu.Unlock()
			d.process(j)
		}
	}()
	d.logger.Info("[Push] Initialized...")
}

// stop signals the worker to drain and exit, then blocks until it has. Idempotent.
func (d *dispatcher) stop() {
	d.mu.Lock()
	if d.stopped {
		d.mu.Unlock()
		return
	}
	d.stopped = true
	d.mu.Unlock()
	d.cond.Broadcast()
	<-d.done
}

// process persists the notification, resolves recipients, and fans out to every
// eligible device.
func (d *dispatcher) process(j *job) {
	note := j.note

	// 1. Persist the canonical notification record (feeds the in-app inbox).
	pushNotif := &models.PushNotification{
		ID:        bson.NewObjectID(),
		Title:     note.Title,
		Body:      "", // localized on-device from loc_key/loc_args; no server body
		Type:      string(note.Type),
		Category:  "events",
		Data:      note.auditData(),
		CreatedAt: bson.NewDateTimeFromTime(time.Now()),
	}
	if err := d.repo.createPushNotification(pushNotif); err != nil {
		d.logger.Errorf("[Push] failed to persist notification %s: %s", j.id, err.Error())
		return
	}

	// 2. Resolve recipients.
	userIDs, err := d.recipients(j)
	if err != nil {
		d.logger.Errorf("[Push] failed to resolve recipients for %s: %s", j.id, err.Error())
		return
	}
	if len(userIDs) == 0 {
		return
	}

	users, err := d.repo.findUsers(userIDs)
	if err != nil {
		d.logger.Errorf("[Push] failed to load users for %s: %s", j.id, err.Error())
		return
	}

	// 3. Fan out.
	for _, user := range users {
		d.sendToUser(user, note, pushNotif.ID)
	}
}

// recipients returns the user-id list for a job: a topic's members, or the
// explicit list.
func (d *dispatcher) recipients(j *job) ([]string, error) {
	if j.topic != "" {
		topic, err := d.repo.findTopic(j.topic)
		if err != nil {
			return nil, err
		}
		return topic.Users, nil
	}
	return j.users, nil
}

// sendToUser records the user's inbox entry and pushes to each of their eligible
// devices.
func (d *dispatcher) sendToUser(user models.User, note EventNote, notifID bson.ObjectID) {
	if err := d.repo.createUserNotification(&models.UserNotification{
		ID:             bson.NewObjectID(),
		UserID:         user.UserID,
		NotificationID: notifID,
		IsRead:         false,
		CreatedAt:      bson.NewDateTimeFromTime(time.Now()),
	}); err != nil {
		d.logger.Errorf("[Push] failed to create user notification for %s: %s", user.UserID, err.Error())
		// continue — a missing inbox row shouldn't block the push itself
	}

	if user.NotificationDevices == nil || user.NotificationPreference == nil {
		return
	}

	for _, device := range *user.NotificationDevices {
		platform := device.DeviceInfo.Platform

		// Skip tokens we've already marked dead.
		if device.Active != nil && !*device.Active {
			continue
		}
		// Skip platforms we don't (yet) deliver to.
		sndr, ok := d.senders[platform]
		if !ok {
			continue
		}
		// Honor the user's per-platform preference.
		if prefKey, ok := prefKeyFor[platform]; !ok || !user.NotificationPreference.Types[prefKey] {
			continue
		}

		d.deliver(sndr, note, user.UserID, device.Token, platform, notifID)
	}
}

// deliver sends one note to one device, logs the attempt, and deactivates the
// device if its token is permanently invalid.
func (d *dispatcher) deliver(sndr sender, note EventNote, userID, token, platform string, notifID bson.ObjectID) {
	res := sndr.send(note, token)

	log := &models.NotificationLog{
		ID:             bson.NewObjectID(),
		NotificationID: notifID,
		Platform:       platform,
		Status:         "sent",
		CreatedAt:      bson.NewDateTimeFromTime(time.Now()),
	}
	if !res.sent {
		log.Status = "failed"
		if res.err != nil {
			errStr := res.err.Error()
			log.Error = &errStr
		}
	}
	if err := d.repo.createNotificationLog(log); err != nil {
		d.logger.Errorf("[Push] failed to write notification log: %s", err.Error())
	}

	if res.deadToken {
		if err := d.repo.deactivateDevice(userID, token); err != nil {
			d.logger.Errorf("[Push] failed to deactivate dead token for %s: %s", userID, err.Error())
		}
	}
}

// ---- priority queue (higher priority drains first) ----

type pushQueue []*job

func (q pushQueue) Len() int           { return len(q) }
func (q pushQueue) Less(i, j int) bool { return q[i].prio > q[j].prio }
func (q pushQueue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index = i
	q[j].index = j
}

func (q *pushQueue) Push(x any) {
	j := x.(*job)
	j.index = len(*q)
	*q = append(*q, j)
}

func (q *pushQueue) Pop() any {
	old := *q
	n := len(old)
	j := old[n-1]
	j.index = -1
	*q = old[:n-1]
	return j
}
