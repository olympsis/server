package service

import (
	"context"
	"encoding/json"
	"net/http"
	"olympsis-server/database"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/olympsis/models"
	"github.com/olympsis/notif"
	"github.com/olympsis/search"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

/*
Event Service Struct
*/
type Service struct {
	Database *database.Database

	// logrus logger to Logger information about service and errors
	Logger *logrus.Logger

	// mux Router to complete http requests
	Router *mux.Router

	// notif service
	NotifService *notif.Service

	// search service
	SearchService *search.Service
}

/*
Create new field service struct
*/
func NewEventService(l *logrus.Logger, r *mux.Router, d *database.Database, n *notif.Service, sh *search.Service) *Service {
	return &Service{Logger: l, Router: r, Database: d, NotifService: n, SearchService: sh}
}

/*
Create Event Data (POST)

  - Creates new Event for olympsis

  - Decode request body

  - Create Event data in databse

Returns:

	Http handler
		- Writes object back to client
*/
func (e *Service) CreateEvent() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("UUID")

		// decode request
		var req models.Event
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		event := models.Event{
			ID:              primitive.NewObjectID(),
			Poster:          uuid,
			ClubID:          req.ClubID,
			FieldID:         req.FieldID,
			ImageURL:        req.ImageURL,
			Title:           req.Title,
			Body:            req.Body,
			Sport:           req.Sport,
			StartTime:       req.StartTime,
			MaxParticipants: req.MaxParticipants,
			Participants:    req.Participants,
			Likes:           []models.Like{},
			Level:           req.Level,
			Status:          "pending",
			Visibility:      req.Visibility,
			CreatedAt:       time.Now().Unix(),
		}

		// insert event in database
		err = e.InsertEvent(context.Background(), &event)
		if err != nil {
			e.Logger.Error(err.Error())
			http.Error(rw, "failed to insert event", http.StatusInternalServerError)
			return
		}

		// subscribe owner to notifications
		e.NotifService.CreateTopic(event.ID.Hex())
		e.NotifService.AddTokenToTopic(event.ID.Hex(), uuid)

		// notify club members
		note := notif.Notification{
			Title: "New Event Created",
			Body:  event.Title,
			Topic: event.ClubID.Hex(),
			Data:  event,
		}
		e.NotifService.SendNotificationToTopic(&note)

		rw.WriteHeader(http.StatusCreated)
		json.NewEncoder(rw).Encode(event)
	}
}

/*
Get Event (GET)
-	Grab event id from path
-	Grabs event data from database

Returns:

	Http handler
		- Writes event back to client
*/
func (e *Service) GetEvent() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("UUID")

		// grab event id from path
		vars := mux.Vars(r)
		if len(vars["id"]) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no event id found in request" }`))
			return
		}
		id := vars["id"]

		// find field data in database
		var event models.Event
		OID, _ := primitive.ObjectIDFromHex(id)
		filter := bson.D{primitive.E{Key: "_id", Value: OID}}
		err := e.FindEvent(context.TODO(), filter, &event)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(rw, "event not found", http.StatusNotFound)
				return
			}
		}

		// add user data to participants
		for ptp := range event.Participants {
			data, err := e.SearchService.SearchUserByUUID(event.Participants[ptp].UUID)
			if err != nil {
				e.Logger.Error(err.Error())
			}
			event.Participants[ptp].Data = &data
		}

		// event data
		user, err := e.SearchService.SearchUserByUUID(uuid)
		if err != nil {
			e.Logger.Error(err.Error())
		}

		var field models.Field
		e.Database.FieldCol.FindOne(context.Background(), bson.M{"_id": event.FieldID}).Decode(&field)

		var club models.Club
		e.Database.ClubCol.FindOne(context.Background(), bson.M{"_id": event.ClubID}).Decode(&club)

		data := models.EventData{
			Poster: &user,
			Field:  &field,
			Club:   &club,
		}
		event.Data = &data

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(event)
	}
}

/*
Get Events (GET)
-	Grabs events from database

Returns:

	Http handler
		- Writes an array of events back to client
*/
func (e *Service) GetEventsByLocation() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// query params
		longitude, _ := strconv.ParseFloat(r.URL.Query().Get("longitude"), 64)
		latitude, _ := strconv.ParseFloat(r.URL.Query().Get("latitude"), 64)
		radius, _ := strconv.ParseFloat(r.URL.Query().Get("radius"), 64)
		sports := r.URL.Query().Get("sports")

		if longitude == 0 || latitude == 0 || sports == "" {
			http.Error(rw, "missing query param - long/lat/sports", http.StatusBadRequest)
			return
		}

		splicedSports := strings.Split(sports, ",")
		var events []models.Event

		loc := models.GeoJSON{
			Type:        "Point",
			Coordinates: []float64{longitude, latitude},
		}
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

		projection := bson.M{"_id": 1}

		cursor, err := e.Database.FieldCol.Find(context.Background(), filter, options.Find().SetProjection(projection))
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(rw, "no events found", http.StatusNotFound)
				return
			}
		}

		var fieldsIDs []primitive.ObjectID
		for cursor.Next(context.TODO()) {
			// decode field
			var field models.Field
			err := cursor.Decode(&field)
			if err != nil {
				e.Logger.Error(err.Error())
			}
			fieldsIDs = append(fieldsIDs, field.ID)
		}

		// filter to find events
		filter = bson.M{
			"field_id": bson.M{
				"$in": fieldsIDs,
			},
			"sport": bson.M{
				"$in": splicedSports,
			},
			"visibility": "public",
			"$or": []interface{}{
				bson.M{"status": "pending"},
				bson.M{"status": "in-progress"},
			},
		}

		var _events []models.Event
		err = e.FindEvents(context.Background(), filter, &_events)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				rw.WriteHeader(http.StatusNotFound)
				rw.Write([]byte(`{ "msg": "field does not exist" }`))
				return
			}
		}

		// fetch owner data for all events
		for index := range _events {
			// event data
			user, err := e.SearchService.SearchUserByUUID(_events[index].Poster)
			if err != nil {
				e.Logger.Error(err.Error())
			}

			var field models.Field
			e.Database.FieldCol.FindOne(context.Background(), bson.M{"_id": _events[index].FieldID}).Decode(&field)

			var club models.Club
			e.Database.ClubCol.FindOne(context.Background(), bson.M{"_id": _events[index].ClubID}).Decode(&club)

			data := models.EventData{
				Poster: &user,
				Field:  &field,
				Club:   &club,
			}
			_events[index].Data = &data

			// add user data to participants
			for ptp := range _events[index].Participants {
				data, err := e.SearchService.SearchUserByUUID(_events[index].Participants[ptp].UUID)
				if err != nil {
					e.Logger.Error(err.Error())
				}
				_events[index].Participants[ptp].Data = &data
			}
		}

		// add events to array
		events = append(events, _events...)

		if len(events) == 0 {
			http.Error(rw, "no events", http.StatusNoContent)
			return
		}

		resp := models.EventsResponse{
			TotalEvents: len(events),
			Events:      events,
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(resp)
	}
}

/*
Update Event (UPDATE)

  - grab event id from path

  - decode request body

  - update document

Returns:

	Http handler
		- write back ok to client
*/
func (e *Service) UpdateAnEvent() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// grab event id from path
		vars := mux.Vars(r)
		if len(vars["id"]) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no event Id found in request." }`))
			return
		}
		id := vars["id"]

		// decode request
		var req models.Event
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		oid, _ := primitive.ObjectIDFromHex(id)
		filter := bson.M{"_id": oid}
		changes := bson.M{}
		updates := bson.M{"$set": changes}

		if req.Title != "" {
			changes["title"] = req.Title
		}
		if req.Body != "" {
			changes["body"] = req.Body
		}
		if req.ImageURL != "" {
			changes["image_url"] = req.ImageURL
		}
		if req.MaxParticipants != 0 {
			changes["max_participants"] = req.MaxParticipants
		}
		if req.StartTime != 0 {
			changes["start_time"] = req.StartTime
		}
		if req.ActualStartTime != 0 {
			changes["actual_start_time"] = req.ActualStartTime
		}
		if req.StopTime != 0 {
			changes["stop_time"] = req.StopTime
		}
		if req.Status != "" {
			changes["status"] = req.Status
		}
		if req.Level != 0 {
			changes["level"] = req.Level
		}
		if req.Visibility != "" {
			changes["visibility"] = req.Visibility
		}

		var event models.Event

		err = e.UpdateEvent(context.Background(), filter, updates, &event)
		if err != nil {
			e.Logger.Error(err)
			http.Error(rw, "failed to update event", http.StatusInternalServerError)
			return
		}

		// notify participants
		if req.ActualStartTime != 0 && req.Status == "in-progress" {
			// notify participants that the event is starting
			note := notif.Notification{
				Title: event.Title,
				Body:  "Event is starting",
				Topic: event.ID.Hex(),
			}
			e.NotifService.SendNotificationToTopic(&note)

			rw.WriteHeader(http.StatusOK)
			json.NewEncoder(rw).Encode(&event)
			return
		}

		// notify participants
		if req.StopTime != 0 && req.Status == "ended" {
			// notify participants that the event ended
			note := notif.Notification{
				Title: event.Title,
				Body:  "Event ended",
				Topic: event.ID.Hex(),
			}
			e.NotifService.SendNotificationToTopic(&note)
			e.NotifService.DeleteTopic(event.ID.Hex())
			rw.WriteHeader(http.StatusOK)
			json.NewEncoder(rw).Encode(&event)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(&event)
	}
}

/*
Delete Event Data (Delete)

  - Updates event data

  - Grab parameters and update

Returns:

	Http handler
		- Writes OK  back to client
*/
func (e *Service) DeleteAnEvent() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// grab id from path
		vars := mux.Vars(r)
		if len(vars["id"]) < 24 {
			http.Error(rw, "bad event id", http.StatusBadRequest)
			return
		}
		id := vars["id"]

		oid, _ := primitive.ObjectIDFromHex(id)
		filter := bson.M{"_id": oid}
		err := e.DeleteEvent(context.Background(), filter)
		if err != nil {
			e.Logger.Debug(err.Error())
			http.Error(rw, "failed to delete event", http.StatusInternalServerError)
		}

		// delete notification topic
		e.NotifService.DeleteTopic(id)
		rw.WriteHeader(http.StatusOK)
	}
}

/*
Add Participant (POST)

  - grab event id from path

  - decode body

  - add participant to event

Returns:

	Http handler
		- Writes back participant object to client
*/
func (e *Service) AddParticipant() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("UUID")

		// grab id from path
		vars := mux.Vars(r)
		if len(vars["id"]) < 24 {
			http.Error(rw, "bad event id", http.StatusBadRequest)
			return
		}
		id := vars["id"]

		var req models.Participant
		// decode request
		json.NewDecoder(r.Body).Decode(&req)

		req.ID = primitive.NewObjectID()
		req.CreatedAt = time.Now().Unix()

		oid, _ := primitive.ObjectIDFromHex(id)

		// find event data in database
		var event models.Event
		filter := bson.M{"_id": oid}
		err := e.FindEvent(context.Background(), filter, &event)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(rw, "event not found", http.StatusNotFound)
				return
			}
		}

		// check if participant already exists
		for i := range event.Participants {
			if event.Participants[i].UUID == uuid {
				rw.WriteHeader(http.StatusOK)
				json.NewEncoder(rw).Encode(event.Participants[i])
				return
			}
		}

		// if event is full
		if len(event.Participants) >= int(event.MaxParticipants) {
			errResponse := models.FullEventError{
				MSG:   "event capacity is full",
				Event: event,
			}

			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusConflict)
			json.NewEncoder(rw).Encode(errResponse)
			return
		}

		// participant object
		part := &models.Participant{
			ID:        primitive.NewObjectID(),
			UUID:      uuid,
			Status:    req.Status,
			CreatedAt: time.Now().Unix(),
		}

		change := bson.M{"$push": bson.M{"participants": part}}
		err = e.UpdateEvent(context.Background(), filter, change, &event)
		if err != nil {
			e.Logger.Error(err.Error())
			http.Error(rw, "failed to update event", http.StatusInternalServerError)
			return
		}

		// subscribe user to notifications
		e.NotifService.AddTokenToTopic(event.ID.Hex(), uuid)

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(event)
	}
}

/*
Remove Participant (DELETE)

  - grab event id from path

  - grab participant id from path

  - pull participant from event

Returns:

	Http handler
		- Writes back OK to client
*/
func (e *Service) RemoveParticipant() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("UUID")

		// grab id from path
		vars := mux.Vars(r)
		if len(vars["id"]) < 24 {
			http.Error(rw, "bad event id", http.StatusBadRequest)
			return
		}
		id := vars["id"]

		partId := vars["participantID"]

		oid, _ := primitive.ObjectIDFromHex(id)
		poid, _ := primitive.ObjectIDFromHex(partId)

		match := bson.M{"_id": oid}
		change := bson.M{"$pull": bson.M{"participants": bson.M{"_id": poid}}}

		var event models.Event
		err := e.UpdateEvent(context.Background(), match, change, &event)
		if err != nil {
			e.Logger.Error(err.Error())
			http.Error(rw, "failed to update event", http.StatusInternalServerError)
			return
		}

		// unsubscribe user from notifications
		e.NotifService.RemoveTokenFromTopic(event.ID.Hex(), uuid)

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(event)
	}
}

/*
Subscribe to event (POST)

  - grab uuid from token

Returns:

	Http handler
		- Writes back OK to client
*/
func (e *Service) SubscribeToEvent() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("UUID")

		// grab id from path
		vars := mux.Vars(r)
		if len(vars["id"]) < 24 {
			http.Error(rw, "bad event id", http.StatusBadRequest)
			return
		}
		id := vars["id"]

		oid, _ := primitive.ObjectIDFromHex(id)

		// fetch event
		var event models.Event
		filter := bson.M{"_id": oid}
		err := e.FindEvent(context.Background(), filter, &event)
		if err != nil {
			http.Error(rw, "event not found", http.StatusNotFound)
			return
		}

		// subscribe user to notifications
		e.NotifService.AddTokenToTopic(event.ID.Hex(), uuid)
	}
}

/*
Unsubscribe from event (POST)

  - grab uuid from token

Returns:

	Http handler
		- Writes back OK to client
*/
func (e *Service) UnsubscribeFromEvent() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		uuid := r.Header.Get("UUID")

		// grab id from path
		vars := mux.Vars(r)
		if len(vars["id"]) < 24 {
			http.Error(rw, "bad event id", http.StatusBadRequest)
			return
		}
		id := vars["id"]

		oid, _ := primitive.ObjectIDFromHex(id)

		// fetch event
		var event models.Event
		filter := bson.M{"_id": oid}
		err := e.FindEvent(context.Background(), filter, &event)
		if err != nil {
			http.Error(rw, "event not found", http.StatusNotFound)
			return
		}

		// unsubscribe user from notifications
		e.NotifService.RemoveTokenFromTopic(event.ID.Hex(), uuid)

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(event)
	}
}
