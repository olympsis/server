package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"olympsis-server/database"
	"olympsis-server/utils"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/olympsis/models"
	"github.com/olympsis/notif"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

/*
Event Service Struct
*/
type Service struct {
	Database     *database.Database // database for read/write operations
	Logger       *logrus.Logger     // logger for logging errors
	Router       *mux.Router        // router for handling incoming requests
	NotifService *notif.Service     // notification service for notifying users
}

/*
Create new event service struct
*/
func NewEventService(l *logrus.Logger, r *mux.Router, d *database.Database, n *notif.Service) *Service {
	return &Service{Logger: l, Router: r, Database: d, NotifService: n}
}

/*
Create Event Data (POST)

	http handler
	create an event and add it into the database
*/
func (e *Service) CreateEvent() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("UUID")

		// decode request
		var req models.EventDao
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			e.Logger.Error("failed to decode request: " + err.Error())
			http.Error(rw, `{ "msg": "failed to decode event" }`, http.StatusBadRequest)
			return
		}

		// create timestamp & participant
		timestamp := time.Now().Unix()
		participant := models.ParticipantDao{
			ID:        primitive.NewObjectID(),
			UUID:      uuid,
			Status:    "yes",
			CreatedAt: &timestamp,
		}

		// create event model
		event := models.EventDao{
			Type:            req.Type,
			Poster:          &uuid,
			Organizers:      req.Organizers,
			Field:           req.Field,
			ImageURL:        req.ImageURL,
			Title:           req.Title,
			Body:            req.Body,
			Sport:           req.Sport,
			StartTime:       req.StartTime,
			StopTime:        req.StopTime,
			MinParticipants: req.MinParticipants,
			MaxParticipants: req.MaxParticipants,
			Participants:    &[]models.ParticipantDao{participant},
			Level:           req.Level,
			Visibility:      req.Visibility,
			ExternalLink:    req.ExternalLink,
			CreatedAt:       &timestamp,
		}

		// insert event into database
		id, err := e.InsertEvent(context.Background(), &event)
		if err != nil {
			e.Logger.Error("failed to insert event: ", err.Error())
			http.Error(rw, `{ "msg": "failed to insert event" }`, http.StatusInternalServerError)
			return
		}

		// subscribe owner to notifications
		e.NotifService.CreateTopic(id.Hex())
		e.NotifService.AddTokenToTopic(id.Hex(), uuid)

		organizers := *event.Organizers

		// notify all of the organizers and their members about this new event
		for i := range organizers {
			organizer := organizers[i]
			note := notif.Notification{
				Title: "New Event Created",
				Body:  *event.Title,
				Topic: organizer.ID.Hex(),
				Data:  fmt.Sprintf(`"id": "%s"`, id.Hex()),
			}
			e.NotifService.SendNotificationToTopic(&note)
		}

		rw.WriteHeader(http.StatusCreated)
		rw.Write([]byte(fmt.Sprintf(`{"id": "%s"}`, id.Hex())))
	}
}

/*
Get Event (GET)

	http handler
	get an event by it's id
*/
func (e *Service) GetEvent() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(rw, `{ "msg": "no/bad event id found in request" }`, http.StatusBadRequest)
			return
		}

		oid, _ := primitive.ObjectIDFromHex(id)

		event, err := FindEvent(oid, e.Database)
		if err != nil {
			e.Logger.Error("failed to find event", err.Error())
			http.Error(rw, `{ "msg": "failed to find event" }`, http.StatusInternalServerError)
		}

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(event)
	}
}

/*
Get Events (GET)

	http handler
	gets a list of events by a location
*/
func (e *Service) GetEventsByLocation() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// query params
		longitude, _ := strconv.ParseFloat(r.URL.Query().Get("longitude"), 64)
		latitude, _ := strconv.ParseFloat(r.URL.Query().Get("latitude"), 64)
		radius, _ := strconv.ParseFloat(r.URL.Query().Get("radius"), 64)
		sports := r.URL.Query().Get("sports")
		status := r.URL.Query().Get("status")

		// query checking
		if longitude == 0 || latitude == 0 || radius == 0 || sports == "" || status == "" {
			http.Error(rw, `{ "msg": "missing query param - long/lat/sports/status" }`, http.StatusBadRequest)
			return
		}

		splicedSports := strings.Split(sports, ",")

		// user location
		loc := models.GeoJSON{
			Type:        "Point",
			Coordinates: []float64{longitude, latitude},
		}

		uuid := r.Header.Get("UUID")

		// fields filter
		filter := bson.M{
			"location": bson.M{
				"$near": bson.M{
					"$geometry":    loc,
					"$maxDistance": radius,
				},
			},
			"sports": bson.M{
				"$in": splicedSports,
			},
		}
		// fetch the nearby fields
		cursor, err := e.Database.FieldCol.Find(context.Background(), filter)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(rw, `{ "msg": "no events found" }`, http.StatusNotFound)
				return
			} else {
				e.Logger.Error("failed to find fields", err.Error())
				http.Error(rw, `{ "msg": "failed to find events" }`, http.StatusInternalServerError)
			}
		}

		// decode fields
		fieldsIDs := []primitive.ObjectID{}
		for cursor.Next(context.TODO()) {
			// decode field
			var field models.Field
			err := cursor.Decode(&field)
			if err != nil {
				e.Logger.Error("failed to decode field", err.Error())
			}
			fieldsIDs = append(fieldsIDs, field.ID)
		}

		// find the events
		events, err := FindEvents(uuid, splicedSports, fieldsIDs, loc, int(radius), 100, e.Database)
		if err != nil { // unexpected error
			e.Logger.Error("failed to find events", err.Error())
			http.Error(rw, `{ "msg": "failed to find events" }`, http.StatusInternalServerError)
			return
		}
		if len(*events) == 0 { // no content error
			http.Error(rw, `{ "msg": "no events found" }`, http.StatusNoContent)
			return
		}
		resp := models.EventsResponse{
			TotalEvents: len(*events),
			Events:      *events,
		}

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(resp)
	}
}

/*
Get Events (GET)

	http handler
	gets a list of events by a field id
*/
func (e *Service) GetEventsByField() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no field id found in request." }`))
			return
		}
		oid, _ := primitive.ObjectIDFromHex(id)
		events, err := FindEventsByField(oid, 100, e.Database)
		if err != nil {
			e.Logger.Error("failed to find events", err.Error())
			http.Error(rw, `{ "msg": "failed to find events"}`, http.StatusInternalServerError)
			return
		}
		if len(*events) == 0 {
			http.Error(rw, `{ "msg": "no events found" }`, http.StatusNoContent)
			return
		}

		resp := models.EventsResponse{
			TotalEvents: len(*events),
			Events:      *events,
		}

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(resp)
	}
}

/*
Update Event (UPDATE)

	http handler
	updates an event in the database
*/
func (e *Service) UpdateAnEvent() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// grab event id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(rw, `{ "msg": "no event Id found in request." }`, http.StatusInternalServerError)
			return
		}

		// decode request
		var req models.EventDao
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(rw, `{ "msg": "failed to decode user" }`, http.StatusBadRequest)
			return
		}

		oid, _ := primitive.ObjectIDFromHex(id)
		filter := bson.M{"_id": oid}
		changes := bson.M{}
		updates := bson.M{"$set": changes}

		if req.Title != nil {
			changes["title"] = req.Title
		}
		if req.Body != nil {
			changes["body"] = req.Body
		}
		if req.ImageURL != nil {
			changes["image_url"] = req.ImageURL
		}
		if req.StartTime != nil {
			changes["start_time"] = req.StartTime
		}
		if req.ActualStartTime != nil {
			changes["actual_start_time"] = req.ActualStartTime
		}
		if req.ActualStopTime != nil {
			changes["actual_stop_time"] = req.ActualStopTime
		}
		if req.Visibility != nil {
			changes["visibility"] = req.Visibility
		}

		if req.MinParticipants != nil {
			changes["min_participants"] = req.MinParticipants
		}

		if req.MaxParticipants != nil {
			changes["max_participants"] = req.MaxParticipants
		}

		if req.Level != nil {
			changes["level"] = req.Level
		}

		if req.ExternalLink != nil {
			changes["external_link"] = req.ExternalLink
		}

		if req.StopTime == nil {
			updates["$unset"] = bson.M{"stop_time": 1}
		} else {
			changes["stop_time"] = req.StopTime
		}

		err = e.UpdateEvent(context.Background(), filter, updates)
		if err != nil {
			e.Logger.Error("failed to find event", err.Error())
			http.Error(rw, `{ "msg": "failed to update event" }`, http.StatusInternalServerError)
			return
		}

		event, err := e.FindEvent(context.Background(), bson.M{"_id": oid})
		if err != nil {
			e.Logger.Error("failed to find event", err.Error())
			http.Error(rw, `{ "msg": "failed to find event" }`, http.StatusInternalServerError)
			return
		}

		// notify participants
		if req.ActualStartTime != nil {
			// notify participants that the event is starting
			note := notif.Notification{
				Title: *event.Title,
				Body:  "Event is starting!",
				Topic: id,
			}
			e.NotifService.SendNotificationToTopic(&note)

		} else if req.ActualStopTime != nil {
			// notify participants that the event ended
			note := notif.Notification{
				Title: *event.Title,
				Body:  "Event ended!",
				Topic: id,
			}
			e.NotifService.SendNotificationToTopic(&note)
			e.NotifService.DeleteTopic(id)

		} else {
			// notify participants that the details changes
			note := notif.Notification{
				Title: *event.Title,
				Body:  "Event details have changed!",
				Topic: id,
			}
			e.NotifService.SendNotificationToTopic(&note)

		}

		rw.WriteHeader(http.StatusOK)
	}
}

/*
Delete Event Data (Delete)

	http handler
	deletes an event from the database
*/
func (e *Service) DeleteAnEvent() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// grab id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(rw, `{ "msg": "bad event id" }`, http.StatusBadRequest)
			return
		}

		oid, _ := primitive.ObjectIDFromHex(id)
		filter := bson.M{"_id": oid}
		err := e.DeleteEvent(context.Background(), filter)
		if err != nil {
			e.Logger.Error("failed to delete event", err.Error())
			http.Error(rw, `{ "msg": "failed to delete event" }`, http.StatusInternalServerError)
		}

		// delete notification topic
		e.NotifService.DeleteTopic(id)
		rw.WriteHeader(http.StatusOK)
	}
}

/*
Add Participant (POST)

	http handler
	adds a participant to an event
	adds a participant to an event's topic
*/
func (e *Service) AddParticipant() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("UUID")

		// grab id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(rw, `"msg": "bad event id" `, http.StatusBadRequest)
			return
		}

		// decode request
		var req models.ParticipantDao
		json.NewDecoder(r.Body).Decode(&req)

		oid, _ := primitive.ObjectIDFromHex(id)
		timestamp := time.Now().Unix()

		// find event data in database
		filter := bson.M{"_id": oid}
		event, err := e.FindEvent(context.Background(), filter)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(rw, `{ "msg": "event not found" }`, http.StatusNotFound)
				return
			}
			e.Logger.Error("failed to find event", err.Error())
			http.Error(rw, `{ "msg": "failed to find event" }`, http.StatusInternalServerError)
			return
		}

		// check if participant already exists
		particpants := *event.Participants
		for i := range particpants {
			if particpants[i].UUID == uuid {
				rw.WriteHeader(http.StatusOK)
				return
			}
		}

		// if event is full
		if *event.MaxParticipants != 0 {
			if len(particpants) >= int(*event.MaxParticipants) {
				http.Error(rw, `{ "msg": "event capacity is full" }`, http.StatusBadRequest)
				return
			}
		}

		// participant object
		part := &models.ParticipantDao{
			ID:        primitive.NewObjectID(),
			UUID:      uuid,
			Status:    req.Status,
			CreatedAt: &timestamp,
		}
		change := bson.M{"$push": bson.M{"participants": part}}
		err = e.UpdateEvent(context.Background(), filter, change)
		if err != nil {
			e.Logger.Error("failed to update event", err.Error())
			http.Error(rw, "failed to update event", http.StatusInternalServerError)
			return
		}

		// notify all participants
		notif := notif.Notification{
			Title: *event.Title,
			Body:  "New Participant RSVP'd!",
			Topic: id,
		}
		e.NotifService.SendNotificationToTopic(&notif)

		// subscribe user to notifications
		e.NotifService.AddTokenToTopic(id, uuid)

		rw.WriteHeader(http.StatusOK)
	}
}

/*
Remove Participant (DELETE)

	http handler
	removes participant from the event object
	removes participant from the event topic
*/
func (e *Service) RemoveParticipant() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("UUID")

		// grab id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(rw, `{ "msg": "bad event id" }`, http.StatusBadRequest)
			return
		}

		oid, _ := primitive.ObjectIDFromHex(id)
		match := bson.M{"_id": oid}
		change := bson.M{"$pull": bson.M{"participants": bson.M{"uuid": uuid}}}

		err := e.UpdateEvent(context.Background(), match, change)
		if err != nil {
			e.Logger.Error("failed to update event: ", err.Error())
			http.Error(rw, `{ "msg": "failed to update event" }`, http.StatusInternalServerError)
			return
		}

		// unsubscribe user from notifications
		e.NotifService.RemoveTokenFromTopic(id, uuid)
		rw.WriteHeader(http.StatusOK)
	}
}

/*
Notify Event Participants (POST)

	http handler
	sends a custom notification to the event's participants
*/
func (e *Service) NotifyParticipants() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// grab id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(rw, `{ "msg": "bad event id" }`, http.StatusBadRequest)
			return
		}

		// decode request
		var req notif.Notification
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			e.Logger.Error("failed to decode notification", err.Error())
			http.Error(rw, `{ "msg": "failed to decode notification" }`, http.StatusInternalServerError)
			return
		}

		req.Topic = id
		e.NotifService.SendNotificationToTopic(&req)
		rw.WriteHeader(http.StatusOK)
	}
}

/*
Notify Club members (POST)

	http handler
	notifies all club members of an event
*/
func (e *Service) NotifyClubMembers() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// grab id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(vars["id"]) < 24 {
			http.Error(rw, `{ "msg": "bad event id" }`, http.StatusBadRequest)
			return
		}

		// decode request
		var req notif.Notification
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(rw, `{ "msg": "failed to decode request" }`, http.StatusInternalServerError)
			return
		}

		req.Topic = id
		e.NotifService.SendNotificationToTopic(&req)
		rw.WriteHeader(http.StatusOK)
	}
}

func (e *Service) Location() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// query params
		longitude, _ := strconv.ParseFloat(r.URL.Query().Get("longitude"), 64)
		latitude, _ := strconv.ParseFloat(r.URL.Query().Get("latitude"), 64)
		radius, _ := strconv.ParseFloat(r.URL.Query().Get("radius"), 64)
		limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 2, 64)

		sports := r.URL.Query().Get("sports")
		status := r.URL.Query().Get("status")

		// query checking
		if longitude == 0 || latitude == 0 || radius == 0 || sports == "" || status == "" {
			http.Error(rw, "missing query param - long/lat/sports/status", http.StatusBadRequest)
			return
		}

		if limit == 0 {
			limit = 100 // max query size for events
		}

		splicedSports := strings.Split(sports, ",")

		// user location
		loc := models.GeoJSON{
			Type:        "Point",
			Coordinates: []float64{longitude, latitude},
		}

		// fields filter
		filter := bson.M{
			"location": bson.M{
				"$nearSphere": bson.M{
					"$geometry":    loc,
					"$maxDistance": radius,
				},
			},
			"sports": bson.M{
				"$in": splicedSports,
			},
		}

		uuid := r.Header.Get("UUID")
		fields := utils.NewSafeFields()
		fieldsArr := []models.Field{}

		// fetch the nearby fields
		cursor, err := e.Database.FieldCol.Find(context.Background(), filter)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				e.Logger.Error(err.Error())
				http.Error(rw, "no events found", http.StatusNotFound)
				return
			} else {
				e.Logger.Error(err.Error())
			}
		}

		// decode fields
		var fieldsIDs []primitive.ObjectID
		for cursor.Next(context.TODO()) {
			// decode field
			var field models.Field
			err := cursor.Decode(&field)
			if err != nil {
				e.Logger.Error(err.Error())
			}
			fieldsIDs = append(fieldsIDs, field.ID)
			fields.AddField(&field)
			fieldsArr = append(fieldsArr, field)
		}

		if len(fieldsIDs) == 0 {
			http.Error(rw, "no events", http.StatusNoContent)
			return
		}

		events, err := FindEvents(uuid, splicedSports, fieldsIDs, loc, int(radius), 100, e.Database)
		if err != nil {
			e.Logger.Error("failed to find events", err.Error())
			http.Error(rw, `{ "msg": "failed to find event" }`, http.StatusInternalServerError)
			return
		}
		resp := models.LocationResponse{
			Fields: &fieldsArr,
			Events: events,
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(resp)
	}
}
