package service

import (
	"context"
	"encoding/json"
	"net/http"
	"olympsis-server/database"
	"olympsis-server/utils"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

/*
Create new field service struct
*/
func NewEventService(l *logrus.Logger, r *mux.Router, d *database.Database) *Service {
	return &Service{Logger: l, Router: r, Database: d}
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

		token, err := utils.GetTokenFromHeader(r)
		if err != nil {
			e.Logger.Error(err.Error())
			http.Error(rw, "unauthorized", http.StatusUnauthorized)
			return
		}

		uuid, _, _, err := utils.ValidateAuthToken(token)
		if err != nil {
			e.Logger.Error("Failed to Decode Token: " + err.Error())
			http.Error(rw, "forbidden", http.StatusForbidden)
			return
		}

		// decode request
		var req Event
		err = json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		event := Event{
			ID:              primitive.NewObjectID(),
			OwnerID:         uuid,
			ClubID:          req.ClubID,
			FieldId:         req.FieldId,
			ImageURL:        req.ImageURL,
			Title:           req.Title,
			Body:            req.Body,
			Sport:           req.Sport,
			StartTime:       req.StartTime,
			MaxParticipants: req.MaxParticipants,
			Participants:    []Participant{},
			Likes:           []Like{},
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

		// TODO subscribe owner to events

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
		// grab event id from path
		vars := mux.Vars(r)
		if len(vars["id"]) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no event id found in request" }`))
			return
		}
		id := vars["id"]

		// find field data in database
		var event Event
		OID, _ := primitive.ObjectIDFromHex(id)
		filter := bson.D{primitive.E{Key: "_id", Value: OID}}
		err := e.FindEvent(context.TODO(), filter, &event)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(rw, "event not found", http.StatusNotFound)
				return
			}
		}

		// TODO Fetch data about owner

		// add user data to participants
		for ptp := range event.Participants {
			var participantData UserData
			err = e.FetchDataAboutUser(event.Participants[ptp].UUID, &participantData)
			if err != nil {
				e.Logger.Error(err.Error())
			} else {
				event.Participants[ptp].Data = &participantData
			}
		}

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
		/*
			token, err := utils.GetTokenFromHeader(r)
			if err != nil {
				e.Logger.Error(err.Error())
				http.Error(rw, "unauthorized", http.StatusUnauthorized)
				return
			}

			_, _, _, err := utils.ValidateAuthToken(token)
			if err != nil {
				e.Logger.Error("Failed to Decode Token: " + err.Error())
				http.Error(rw, "forbidden", http.StatusForbidden)
				return
			}*/

		// query params
		longitude, _ := strconv.ParseFloat(r.URL.Query().Get("longitude"), 64)
		latitude, _ := strconv.ParseFloat(r.URL.Query().Get("latitude"), 64)
		radius, _ := strconv.ParseFloat(r.URL.Query().Get("radius"), 64)
		sport := r.URL.Query().Get("sport")

		if longitude == 0 || latitude == 0 || sport == "" {
			http.Error(rw, "missing query param - long/lat/sport", http.StatusBadRequest)
			return
		}

		var events []Event
		var fields []Field

		loc := GeoJSON{
			Type:        "Point",
			Coordinates: []float64{longitude, latitude},
		}

		filter := bson.D{
			{Key: "location",
				Value: bson.D{
					{Key: "$near", Value: bson.D{
						{Key: "$geometry", Value: loc},
						{Key: "$maxDistance", Value: radius},
					}},
				}},
		}

		err := e.FindEvents(context.Background(), filter, &events)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(rw, "no events found", http.StatusNotFound)
				return
			}
		}

		// if there are no fields then there should be no events
		if len(events) == 0 {
			rw.Header().Set("Content-Type", "application/json")
			http.Error(rw, "no events found", http.StatusNoContent)
			return
		}

		// loop through the fields and find the correspoding events
		for i := range fields {
			filter := bson.M{
				"fieldId":    fields[i].ID,
				"sport":      sport,
				"visibility": "public",
				"$or": []interface{}{
					bson.M{"status": "pending"},
					bson.M{"status": "in-progress"},
				},
			}

			err = e.FindEvents(context.Background(), filter, &events)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					rw.WriteHeader(http.StatusNotFound)
					rw.Write([]byte(`{ "msg": "field does not exist" }`))
					return
				}
			}
			//TODO fetch owner data for all events
		}

		if len(events) == 0 {
			http.Error(rw, "no events", http.StatusNoContent)
			return
		}

		resp := EventsResponse{
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
		var req Event
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
			changes["imageURL"] = req.ImageURL
		}
		if req.MaxParticipants != 0 {
			changes["maxParticipants"] = req.MaxParticipants
		}
		if req.StartTime != 0 {
			changes["startTime"] = req.StartTime
		}
		if req.ActualStartTime != 0 {
			changes["actualStartTime"] = req.ActualStartTime
		}
		if req.StopTime != 0 {
			changes["stopTime"] = req.StopTime
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

		var event Event

		err = e.UpdateEvent(context.Background(), filter, updates, &event)
		if err != nil {
			e.Logger.Error(err)
			http.Error(rw, "failed to update event", http.StatusInternalServerError)
		}

		// notify participants
		if req.ActualStartTime != 0 && req.Status == "in-progress" {
			// notify participants that the event is starting
			//e.SendNotificationToTopic(*r, event.Title, "Event is Starting", id)
			rw.WriteHeader(http.StatusOK)
			json.NewEncoder(rw).Encode(&event)
		}

		// notify participants
		if req.StopTime != 0 && req.Status == "ended" {
			// notify participants that the event is starting
			//e.SendNotificationToTopic(*r, event.Title, "Event has Ended", id)
			rw.WriteHeader(http.StatusOK)
			json.NewEncoder(rw).Encode(&event)
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

		token, err := utils.GetTokenFromHeader(r)
		if err != nil {
			e.Logger.Error(err.Error())
			http.Error(rw, "unauthorized", http.StatusUnauthorized)
			return
		}

		uuid, _, _, err := utils.ValidateAuthToken(token)
		if err != nil {
			e.Logger.Error("Failed to Decode Token: " + err.Error())
			http.Error(rw, "forbidden", http.StatusForbidden)
			return
		}

		// grab id from path
		vars := mux.Vars(r)
		if len(vars["id"]) < 24 {
			http.Error(rw, "bad event id", http.StatusBadRequest)
			return
		}
		id := vars["id"]

		var req Participant
		// decode request
		json.NewDecoder(r.Body).Decode(&req)

		req.ID = primitive.NewObjectID()
		req.CreatedAt = time.Now().Unix()

		oid, _ := primitive.ObjectIDFromHex(id)

		// find event data in database
		var event Event
		filter := bson.M{"_id": oid}
		err = e.FindEvent(context.Background(), filter, &event)
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

			// add owner data to event
			var data UserData
			err = e.FetchDataAboutUser(event.OwnerID, &data)
			if err != nil {
				e.Logger.Error(err.Error())
			}
			event.OwnerData = &data

			// fetch participants data
			for ptp := range event.Participants {
				var participantData UserData
				err = e.FetchDataAboutUser(event.Participants[ptp].UUID, &participantData)
				if err != nil {
					e.Logger.Error(err.Error())
				} else {
					event.Participants[ptp].Data = &participantData
				}
			}

			errResponse := FullEventError{
				MSG:   "event capacity is full",
				Event: event,
			}

			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusConflict)
			json.NewEncoder(rw).Encode(errResponse)
			return
		}

		// participant object
		part := &Participant{
			ID:        primitive.NewObjectID(),
			UUID:      uuid,
			Status:    req.Status,
			CreatedAt: time.Now().Unix(),
		}

		change := bson.M{"$push": bson.M{"participants": part}}
		err = e.UpdateEvent(context.Background(), filter, change, &event)
		if err != nil {
			http.Error(rw, "failed to update event", http.StatusInternalServerError)
			return
		}

		// subscribe user to notifications
		var userData UserData
		err = e.FetchDataAboutUser(uuid, &userData)
		if err != nil {
			e.Logger.Error(err.Error())
		} else {
			e.SubscribeToEventTopic(event.ID.Hex(), []string{userData.DeviceToken})
		}

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

		token, err := utils.GetTokenFromHeader(r)
		if err != nil {
			e.Logger.Error(err.Error())
			http.Error(rw, "unauthorized", http.StatusUnauthorized)
			return
		}

		uuid, _, _, err := utils.ValidateAuthToken(token)
		if err != nil {
			e.Logger.Error("Failed to Decode Token: " + err.Error())
			http.Error(rw, "forbidden", http.StatusForbidden)
			return
		}

		// grab id from path
		vars := mux.Vars(r)
		if len(vars["id"]) < 24 {
			http.Error(rw, "bad event id", http.StatusBadRequest)
			return
		}
		id := vars["id"]

		partId := vars["participantId"]

		oid, _ := primitive.ObjectIDFromHex(id)
		poid, _ := primitive.ObjectIDFromHex(partId)

		match := bson.M{"_id": oid}
		change := bson.M{"$pull": bson.M{"participants": bson.M{"_id": poid}}}

		var event Event
		err = e.UpdateEvent(context.Background(), match, change, &event)
		if err != nil {
			e.Logger.Error(err.Error())
			http.Error(rw, "failed to update event", http.StatusInternalServerError)
			return
		}

		// unsubscribe user from notifications
		var userData UserData
		err = e.FetchDataAboutUser(uuid, &userData)
		if err != nil {
			e.Logger.Error(err.Error())
		} else {
			e.UnsubscribeFromEventTopic(id, []string{userData.DeviceToken})
		}

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

		token, err := utils.GetTokenFromHeader(r)
		if err != nil {
			e.Logger.Error(err.Error())
			http.Error(rw, "unauthorized", http.StatusUnauthorized)
			return
		}

		uuid, _, _, err := utils.ValidateAuthToken(token)
		if err != nil {
			e.Logger.Error("Failed to Decode Token: " + err.Error())
			http.Error(rw, "forbidden", http.StatusForbidden)
			return
		}

		// grab id from path
		vars := mux.Vars(r)
		if len(vars["id"]) < 24 {
			http.Error(rw, "bad event id", http.StatusBadRequest)
			return
		}
		id := vars["id"]

		oid, _ := primitive.ObjectIDFromHex(id)

		// fetch event
		var event Event
		filter := bson.M{"_id": oid}
		err = e.FindEvent(context.Background(), filter, &event)
		if err != nil {
			http.Error(rw, "event not found", http.StatusNotFound)
			return
		}

		// fetch device token
		var userData UserData
		err = e.FetchDataAboutUser(uuid, &userData)
		if err != nil {
			e.Logger.Error(err.Error())
		} else {
			e.SubscribeToEventTopic(event.ID.Hex(), []string{userData.DeviceToken})
		}
	}
}
