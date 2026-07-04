package push

import (
	"errors"
	"math"
	"time"

	"olympsis-server/database"

	"firebase.google.com/go/messaging"
	"github.com/olympsis/models"
	"github.com/sideshow/apns2"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Job priorities (0–5). Reminders are the most time-sensitive, so they drain
// ahead of comment/participant notes.
const (
	priorityDefault  = 1
	priorityReminder = 2
)

// Service is the public entry point for sending the event push notifications.
// It owns the dispatcher (queue + senders) and the data layer.
type Service struct {
	repo       repo
	dispatcher *dispatcher
	logger     *logrus.Logger
}

// New wires the push service. apnsClient and fcm may each be nil (e.g. in an
// environment where a transport isn't configured); the corresponding platform is
// then simply not delivered to rather than panicking.
func New(apnsClient *apns2.Client, fcm *messaging.Client, l *logrus.Logger, db *database.Database) *Service {
	r := repo{db: db}

	senders := map[string]sender{}
	if apnsClient != nil {
		senders["ios"] = apnsSender{client: apnsClient}
	}
	if fcm != nil {
		senders["android"] = fcmSender{client: fcm}
	}

	d := newDispatcher(r, senders, l)
	d.start()
	return &Service{repo: r, dispatcher: d, logger: l}
}

// Stop drains the dispatcher and blocks until it has exited. Call once on
// shutdown, before closing Mongo (queued jobs still need the DB).
func (s *Service) Stop() {
	s.dispatcher.stop()
}

// Reminder enqueues an "event starts in N minutes" note to the event's
// participants (the event-id topic). N is computed from the event's start time so
// it matches when the reminder actually fires.
func (s *Service) Reminder(eventID string) error {
	event, err := s.event(eventID)
	if err != nil {
		return err
	}
	if event.StartTime == nil {
		return errors.New("reminder: event has no start time")
	}
	minutes := int(math.Round(time.Until(event.StartTime.Time()).Minutes()))
	if minutes < 1 {
		minutes = 1 // never render "0 minutes"
	}

	note, err := NewReminder(eventID, derefStr(event.Title), derefStr(event.MediaURL), minutes)
	if err != nil {
		return err
	}
	return s.dispatcher.add(&job{prio: priorityReminder, note: note, topic: eventID})
}

// Comment enqueues a "new comment" note to the event's organizer members,
// excluding the event poster (preserving the prior recipient semantics).
func (s *Service) Comment(eventID, commentID string) error {
	event, err := s.event(eventID)
	if err != nil {
		return err
	}

	exclude := ""
	if event.PosterID != nil {
		exclude = *event.PosterID
	}
	users, err := s.organizerMembers(event, exclude)
	if err != nil {
		return err
	}

	note, err := NewComment(eventID, derefStr(event.Title), derefStr(event.MediaURL), commentID)
	if err != nil {
		return err
	}
	return s.dispatcher.add(&job{prio: priorityDefault, note: note, users: users})
}

// Participant enqueues a "new participant" note to the event's organizers only,
// excluding the user who just joined.
func (s *Service) Participant(eventID, participantID, joinerUserID string) error {
	event, err := s.event(eventID)
	if err != nil {
		return err
	}

	users, err := s.organizerMembers(event, joinerUserID)
	if err != nil {
		return err
	}

	note, err := NewParticipant(eventID, derefStr(event.Title), derefStr(event.MediaURL), participantID)
	if err != nil {
		return err
	}
	return s.dispatcher.add(&job{prio: priorityDefault, note: note, users: users})
}

// event fetches an event by its hex id.
func (s *Service) event(eventID string) (*models.EventDao, error) {
	oid, err := bson.ObjectIDFromHex(eventID)
	if err != nil {
		return nil, err
	}
	return s.repo.findEvent(oid)
}

// organizerMembers returns the de-duplicated union of users subscribed to the
// event's organizer topics, minus the excluded user. Mirrors the recipient
// resolution used by the legacy event notifications.
func (s *Service) organizerMembers(event *models.EventDao, exclude string) ([]string, error) {
	if event.Organizers == nil {
		return nil, errors.New("no event organizers found")
	}

	names := make([]string, 0)
	for _, o := range *event.Organizers {
		names = append(names, o.ID.Hex())
	}
	if len(names) == 0 {
		return nil, errors.New("no organizer topics found")
	}

	topics, err := s.repo.findTopics(names)
	if err != nil {
		return nil, err
	}

	set := make(map[string]struct{})
	for i := range topics {
		for _, u := range topics[i].Users {
			if u != exclude {
				set[u] = struct{}{}
			}
		}
	}

	users := make([]string, 0, len(set))
	for u := range set {
		users = append(users, u)
	}
	return users, nil
}

// derefStr safely dereferences an optional string field.
func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
