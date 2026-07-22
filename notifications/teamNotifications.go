package notifications

import (
	"time"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Team RSVP notifications (day tournaments).
//
// These mirror the club member notifications: they build a PushNotification and
// enqueue it on the carousel, targeting explicit user IDs. The event/team Daos
// supply the display data (event title, team name) and are already loaded by the
// caller, so these methods do no extra reads beyond an optional applicant lookup.

// derefString safely dereferences an optional string field for display.
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// teamNoteData assembles the common data payload every team push carries so the
// iOS/Android handlers can route and deep-link consistently.
func teamNoteData(t models.NotificationType, event *models.EventDao, team *models.TeamDao) map[string]any {
	data := map[string]any{
		"type":       t,
		"group_type": "team",
	}
	if event != nil {
		if event.ID != nil {
			data["event_id"] = event.ID.Hex()
		}
		data["event_name"] = derefString(event.Title)
		data["event_media_url"] = derefString(event.MediaURL)
	}
	if team != nil {
		if team.ID != nil {
			data["team_id"] = team.ID.Hex()
		}
		data["team_name"] = derefString(team.Name)
	}
	return data
}

// TeamApplication notifies the team owner(s) that someone applied to their
// closed team and is awaiting approval.
func (n *Service) TeamApplication(event *models.EventDao, team *models.TeamDao, applicantUserID string, recipients []string) error {
	if len(recipients) == 0 {
		return nil
	}

	data := teamNoteData(models.TeamApplicationType, event, team)
	// Best-effort applicant enrichment so the owner sees who applied.
	if user, err := n.findUser(applicantUserID); err == nil && user != nil {
		data["username"] = user.UserName
		data["image_url"] = user.ImageURL
	}

	note := models.PushNotification{
		ID:        bson.NewObjectID(),
		Title:     "New team application!",
		Body:      derefString(team.Name),
		Type:      "push",
		Category:  "events",
		Data:      data,
		CreatedAt: bson.NewDateTimeFromTime(time.Now()),
	}
	return n.carousel.AddJob(1, models.NotificationPushRequest{
		Users:        &recipients,
		Notification: note,
	})
}

// TeamApplicationUpdate notifies the applicant that their application was
// approved or denied.
func (n *Service) TeamApplicationUpdate(event *models.EventDao, team *models.TeamDao, applicantUserID string, approved bool) error {
	users := []string{applicantUserID}
	title := "Your team application was denied."
	if approved {
		title = "You've been added to the team!"
	}

	note := models.PushNotification{
		ID:        bson.NewObjectID(),
		Title:     title,
		Body:      derefString(team.Name),
		Type:      "push",
		Category:  "events",
		Data:      teamNoteData(models.TeamApplicationUpdateType, event, team),
		CreatedAt: bson.NewDateTimeFromTime(time.Now()),
	}
	return n.carousel.AddJob(1, models.NotificationPushRequest{
		Users:        &users,
		Notification: note,
	})
}

// TeamKick notifies a member that the owner removed them from the team (which
// cancels their RSVP for the event).
func (n *Service) TeamKick(event *models.EventDao, team *models.TeamDao, kickedUserID string) error {
	users := []string{kickedUserID}
	note := models.PushNotification{
		ID:        bson.NewObjectID(),
		Title:     "You've been removed from the team.",
		Body:      derefString(team.Name),
		Type:      "push",
		Category:  "events",
		Data:      teamNoteData(models.TeamKickType, event, team),
		CreatedAt: bson.NewDateTimeFromTime(time.Now()),
	}
	return n.carousel.AddJob(1, models.NotificationPushRequest{
		Users:        &users,
		Notification: note,
	})
}

// TeamMemberRoleChange notifies a member that their role changed — used when
// ownership is transferred to them.
func (n *Service) TeamMemberRoleChange(event *models.EventDao, team *models.TeamDao, userID string, newRole models.MemberRole) error {
	users := []string{userID}
	title := "Your team role changed."
	if newRole == models.OwnerMember {
		title = "You're now the team owner."
	}

	data := teamNoteData(models.TeamMemberRoleChangeType, event, team)
	data["role"] = newRole

	note := models.PushNotification{
		ID:        bson.NewObjectID(),
		Title:     title,
		Body:      derefString(team.Name),
		Type:      "push",
		Category:  "events",
		Data:      data,
		CreatedAt: bson.NewDateTimeFromTime(time.Now()),
	}
	return n.carousel.AddJob(1, models.NotificationPushRequest{
		Users:        &users,
		Notification: note,
	})
}

// TeamDeleted notifies the given members that the team was disbanded and their
// RSVP for the event has been cancelled.
func (n *Service) TeamDeleted(event *models.EventDao, team *models.TeamDao, recipients []string) error {
	if len(recipients) == 0 {
		return nil
	}
	note := models.PushNotification{
		ID:        bson.NewObjectID(),
		Title:     "Your team was disbanded and your RSVP cancelled.",
		Body:      derefString(team.Name),
		Type:      "push",
		Category:  "events",
		Data:      teamNoteData(models.TeamDeletedType, event, team),
		CreatedAt: bson.NewDateTimeFromTime(time.Now()),
	}
	return n.carousel.AddJob(1, models.NotificationPushRequest{
		Users:        &recipients,
		Notification: note,
	})
}
