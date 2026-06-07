package service

import (
	"context"
	"time"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// reminderEvent is the slim projection of an event needed to build a chat reminder.
type reminderEvent struct {
	ID         bson.ObjectID      `bson:"_id"`
	Title      string             `bson:"title"`
	Body       string             `bson:"body"`
	StartTime  bson.DateTime      `bson:"start_time"`
	Organizers []models.Organizer `bson:"organizers"`
}

// dispatchChatReminder posts an event reminder to every Telegram/Discord chat linked to
// the event's organizing club. It is deduplicated per event with a dedicated cache key
// so it fires at most once, independent of the APNS reminder path.
func (p *EventPollingService) dispatchChatReminder(stripped StrippedEvent) {
	cacheKey := "chat:" + stripped.ID

	sent, err := p.cache.IsNotificationSent(cacheKey)
	if err != nil {
		p.logger.Errorf("Error checking chat reminder cache for event %s: %v", stripped.ID, err)
		return
	}
	if sent {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	oid, err := bson.ObjectIDFromHex(stripped.ID)
	if err != nil {
		return
	}

	// Fetch the slim event projection.
	var event reminderEvent
	projection := bson.M{"_id": 1, "title": 1, "body": 1, "start_time": 1, "organizers": 1}
	err = p.db.EventsCollection.FindOne(ctx, bson.M{"_id": oid}, options.FindOne().SetProjection(projection)).Decode(&event)
	if err != nil {
		p.logger.Errorf("Failed to load event for chat reminder. Event: %s - Error: %s", stripped.ID, err.Error())
		return
	}

	// Find the organizing club (GROUP organizer).
	clubID, ok := organizingClub(event.Organizers)
	if !ok {
		return // not a club-organized event; nothing to do
	}

	// Find active chat links for that club.
	cursor, err := p.db.ClubChatLinksCollection.Find(ctx, bson.M{"club_id": clubID, "status": models.ChatLinkActive})
	if err != nil {
		p.logger.Errorf("Failed to find chat links. Club: %s - Error: %s", clubID.Hex(), err.Error())
		return
	}
	var links []models.ClubChatLink
	if err := cursor.All(ctx, &links); err != nil {
		p.logger.Errorf("Failed to decode chat links. Club: %s - Error: %s", clubID.Hex(), err.Error())
		return
	}
	if len(links) == 0 {
		return
	}

	roster := p.buildRoster(ctx, clubID)

	for _, link := range links {
		reminder := models.BotReminderRequest{
			EventID:   stripped.ID,
			ClubID:    clubID.Hex(),
			Platform:  link.Platform,
			ChatID:    link.ChatID,
			GuildID:   link.GuildID,
			ChannelID: link.ChannelID,
			Title:     event.Title,
			Body:      event.Body,
			StartsAt:  event.StartTime,
			Roster:    roster,
		}
		if err := p.bots.SendReminder(reminder); err != nil {
			p.logger.Errorf("Failed to send chat reminder. Event: %s - Error: %s", stripped.ID, err.Error())
		}
	}

	// Mark sent for the remaining life of the event window.
	ttl := time.Until(stripped.StopTime.Time())
	if ttl <= 0 {
		ttl = time.Hour
	}
	if err := p.cache.MarkNotificationSent(cacheKey, ttl); err != nil {
		p.logger.Errorf("Failed to mark chat reminder sent. Event: %s - Error: %s", stripped.ID, err.Error())
	}
}

// buildRoster returns the club's members as lightweight RSVP-attribution descriptors
// (user_id + username), used by the bot's LLM to map replies back to members.
func (p *EventPollingService) buildRoster(ctx context.Context, clubID bson.ObjectID) []models.ChatRosterMember {
	memberCursor, err := p.db.ClubMembersCollection.Find(ctx, bson.M{"club_id": clubID})
	if err != nil {
		p.logger.Errorf("Failed to find club members. Club: %s - Error: %s", clubID.Hex(), err.Error())
		return nil
	}
	var members []models.MemberDao
	if err := memberCursor.All(ctx, &members); err != nil {
		p.logger.Errorf("Failed to decode club members. Club: %s - Error: %s", clubID.Hex(), err.Error())
		return nil
	}
	if len(members) == 0 {
		return nil
	}

	userIDs := make([]string, 0, len(members))
	for _, m := range members {
		userIDs = append(userIDs, m.UserID)
	}

	// Look up usernames in one query.
	userCursor, err := p.db.UserCollection.Find(ctx, bson.M{"user_id": bson.M{"$in": userIDs}},
		options.Find().SetProjection(bson.M{"user_id": 1, "username": 1}))
	if err != nil {
		p.logger.Errorf("Failed to find roster users. Error: %s", err.Error())
		return nil
	}
	var users []struct {
		UserID   string `bson:"user_id"`
		Username string `bson:"username"`
	}
	if err := userCursor.All(ctx, &users); err != nil {
		p.logger.Errorf("Failed to decode roster users. Error: %s", err.Error())
		return nil
	}

	roster := make([]models.ChatRosterMember, 0, len(users))
	for _, u := range users {
		roster = append(roster, models.ChatRosterMember{
			UserID:   u.UserID,
			Username: u.Username,
		})
	}
	return roster
}

// organizingClub returns the club ObjectID from an event's organizers, if a GROUP
// organizer is present.
func organizingClub(organizers []models.Organizer) (bson.ObjectID, bool) {
	for _, o := range organizers {
		if o.Type == models.OrganizerTypeGroup {
			return o.ID, true
		}
	}
	return bson.NilObjectID, false
}
