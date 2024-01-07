package service

import (
	"context"
	"encoding/json"
	"net/http"
	"olympsis-server/database"
	"olympsis-server/utils"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/olympsis/models"
	"github.com/olympsis/notif"
	"github.com/olympsis/search"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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
			http.Error(rw, `{ "msg": " `+err.Error()+`" }`, http.StatusBadRequest)
			e.Logger.Error("Failed to decode request: " + err.Error())
			return
		}

		timestamp := time.Now().Unix()
		participant := models.Participant{
			ID:        primitive.NewObjectID(),
			UUID:      uuid,
			Status:    "yes",
			CreatedAt: timestamp,
		}

		event := models.Event{
			ID:              primitive.NewObjectID(),
			Type:            req.Type,
			Poster:          uuid,
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
			Participants:    []models.Participant{participant},
			Level:           req.Level,
			Visibility:      req.Visibility,
			ExternalLink:    req.ExternalLink,
			CreatedAt:       timestamp,
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

		// notify all of the organizers and their members about this new event
		for i := range event.Organizers {
			organizer := event.Organizers[i]
			note := notif.Notification{
				Title: "New Event Created",
				Body:  event.Title,
				Topic: organizer.ID.Hex(),
				Data:  event,
			}
			e.NotifService.SendNotificationToTopic(&note)
		}

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
		id := vars["id"]
		if len(id) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no event id found in request" }`))
			return
		}

		// find event data in database
		var event models.Event
		oid, _ := primitive.ObjectIDFromHex(id)
		filter := bson.M{"_id": oid}
		err := e.FindEvent(context.TODO(), filter, &event)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(rw, "event not found", http.StatusNotFound)
				return
			}
		}

		// find field data in database
		var field models.Field
		filter = bson.M{"_id": event.Field.ID}
		err = e.Database.FieldCol.FindOne(context.Background(), bson.M{"_id": event.Field.ID}).Decode(&field)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				e.Logger.Error("Event field not found")
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

		data := models.EventData{
			Poster: &user,
			Field:  &field,
		}
		dataClubs := []models.Club{}
		dataOrganizations := []models.Organization{}
		clubs := make(map[primitive.ObjectID]models.Club)
		organizations := make(map[primitive.ObjectID]models.Organization)

		// fetch club/organization data if not in their respective maps
		for j := range event.Organizers {
			if event.Organizers[j].Type == "club" {
				id := event.Organizers[j].ID
				c, ok := clubs[id]
				if !ok {
					var club models.Club
					err := e.Database.ClubCol.FindOne(context.Background(), bson.M{"_id": id}).Decode(&club)
					if err != nil {
						e.Logger.Error(err.Error())
					} else {
						clubs[id] = club
						dataClubs = append(dataClubs, club)
					}
				} else {
					dataClubs = append(dataClubs, c)
				}
			} else {
				id := event.Organizers[j].ID
				o, ok := organizations[id]
				if !ok {
					var org models.Organization
					err := e.Database.OrgCol.FindOne(context.Background(), bson.M{"_id": id}).Decode(&org)
					if err != nil {
						e.Logger.Error(err.Error())
					} else {
						organizations[id] = org
						dataOrganizations = append(dataOrganizations, org)
					}
				} else {
					dataOrganizations = append(dataOrganizations, o)
				}
			}
		}

		data.Clubs = &dataClubs
		data.Organizations = &dataOrganizations
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
		status := r.URL.Query().Get("status")

		// query checking
		if longitude == 0 || latitude == 0 || radius == 0 || sports == "" || status == "" {
			http.Error(rw, "missing query param - long/lat/sports/status", http.StatusBadRequest)
			return
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
				http.Error(rw, "no events found", http.StatusNotFound)
				return
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
		}

		var exists bool

		// status filters
		if status == "live" {
			exists = false
		} else if status == "ended" {
			exists = true
		}

		// filter to find events
		filter = bson.M{
			"field._id": bson.M{
				"$in": fieldsIDs,
			},
			"sport": bson.M{
				"$in": splicedSports,
			},
			"actual_stop_time": bson.M{
				"$exists": exists,
			},
			"visibility": "public",
		}

		var events []models.Event
		err = e.FindEvents(context.Background(), filter, &events)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				rw.WriteHeader(http.StatusNotFound)
				rw.Write([]byte(`{ "msg": "field does not exist" }`))
				return
			}
		}

		data := models.EventData{}
		dataClubs := []models.Club{}
		dataOrganizations := []models.Organization{}

		// fetch owner data for all events
		for i := range events {

			fields := make(map[primitive.ObjectID]models.Field)
			users := make(map[string]models.UserData)
			clubs := make(map[primitive.ObjectID]models.Club)
			organizations := make(map[primitive.ObjectID]models.Organization)

			// fetch user data if not in users map
			u, ok := users[events[i].Poster]
			if !ok {
				user, err := e.SearchService.SearchUserByUUID(events[i].Poster)
				if err != nil {
					e.Logger.Error(err.Error())
				} else {
					users[events[i].Poster] = user
					data.Poster = &user
				}
			} else {
				data.Poster = &u
			}

			// fetch fields if we don't have it on the map
			f, ok := fields[events[i].Field.ID]
			if !ok {
				var field models.Field
				err = e.Database.FieldCol.FindOne(context.Background(), bson.M{"_id": events[i].Field.ID}).Decode(&field)
				if err != nil {
					e.Logger.Error("Failed to fetch field:" + err.Error())
				} else {
					fields[field.ID] = field
					data.Field = &field
				}
			} else {
				data.Field = &f
			}

			// fetch club/organization data if not in their respective maps
			for j := range events[i].Organizers {
				if events[i].Organizers[j].Type == "club" {
					id := events[i].Organizers[j].ID
					c, ok := clubs[id]
					if !ok {
						var club models.Club
						err := e.Database.ClubCol.FindOne(context.Background(), bson.M{"_id": id}).Decode(&club)
						if err != nil {
							e.Logger.Error(err.Error())
						} else {
							clubs[id] = club
							dataClubs = append(dataClubs, club)
						}
					} else {
						dataClubs = append(dataClubs, c)
					}
				} else {
					id := events[i].Organizers[j].ID
					o, ok := organizations[id]
					if !ok {
						var org models.Organization
						err := e.Database.OrgCol.FindOne(context.Background(), bson.M{"_id": id}).Decode(&org)
						if err != nil {
							e.Logger.Error(err.Error())
						} else {
							organizations[id] = org
							dataOrganizations = append(dataOrganizations, org)
						}
					} else {
						dataOrganizations = append(dataOrganizations, o)
					}
				}
			}

			data.Clubs = &dataClubs
			data.Organizations = &dataOrganizations
			events[i].Data = &data

			// add user data to participants
			for ptp := range events[i].Participants {
				data, err := e.SearchService.SearchUserByUUID(events[i].Participants[ptp].UUID)
				if err != nil {
					e.Logger.Error(err.Error())
				}
				events[i].Participants[ptp].Data = &data
			}
		}

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
Get Events (GET)
- 	Grabs events by club ID

Returns:

	Http handler
		- Writes an array of events back to client
*/
func (e *Service) GetEventsByClub() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		if len(vars["id"]) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no club Id found in request." }`))
			return
		}
		id := vars["id"]
		oid, _ := primitive.ObjectIDFromHex(id)
		filter := bson.M{"club_id": oid}

		var events []models.Event
		err := e.FindEvents(context.Background(), filter, &events)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				rw.WriteHeader(http.StatusNotFound)
				rw.Write([]byte(`{ "msg": "club does not exist" }`))
				return
			}
		}

		var club models.Club
		fields := make(map[primitive.ObjectID]models.Field)
		users := make(map[string]models.UserData)
		clubs := make(map[primitive.ObjectID]models.Club)
		organizations := make(map[primitive.ObjectID]models.Organization)

		e.Database.ClubCol.FindOne(context.Background(), bson.M{"_id": oid}).Decode(&club)

		data := models.EventData{}
		dataClubs := []models.Club{club}
		dataOrganizations := []models.Organization{}

		for i := range events {

			// fetch user data if not in users map
			u, ok := users[events[i].Poster]
			if !ok {
				user, err := e.SearchService.SearchUserByUUID(events[i].Poster)
				if err != nil {
					e.Logger.Error(err.Error())
				} else {
					users[events[i].Poster] = user
					data.Poster = &user
				}
			} else {
				data.Poster = &u
			}

			// fetch fields if we don't have it on the map
			f, ok := fields[events[i].Field.ID]
			if !ok {
				var field models.Field
				err = e.Database.FieldCol.FindOne(context.Background(), bson.M{"_id": events[i].Field.ID}).Decode(&field)
				if err != nil {
					e.Logger.Error("Failed to fetch field:" + err.Error())
				} else {
					fields[field.ID] = field
					data.Field = &field
				}
			} else {
				data.Field = &f
			}

			// fetch club/organization data if not in their respective maps
			for j := range events[i].Organizers {
				if events[i].Organizers[j].Type == "club" {
					id := events[i].Organizers[j].ID
					c, ok := clubs[id]
					if !ok {
						var club models.Club
						err := e.Database.ClubCol.FindOne(context.Background(), bson.M{"_id": id}).Decode(&club)
						if err != nil {
							e.Logger.Error(err.Error())
						} else {
							clubs[id] = club
							dataClubs = append(dataClubs, club)
						}
					} else {
						dataClubs = append(dataClubs, c)
					}
				} else {
					id := events[i].Organizers[j].ID
					o, ok := organizations[id]
					if !ok {
						var org models.Organization
						err := e.Database.OrgCol.FindOne(context.Background(), bson.M{"_id": id}).Decode(&org)
						if err != nil {
							e.Logger.Error(err.Error())
						} else {
							organizations[id] = org
							dataOrganizations = append(dataOrganizations, org)
						}
					} else {
						dataOrganizations = append(dataOrganizations, o)
					}
				}
			}

			data.Clubs = &dataClubs
			data.Organizations = &dataOrganizations
			events[i].Data = &data

			// add user data to participants
			for ptp := range events[i].Participants {
				data, err := e.SearchService.SearchUserByUUID(events[i].Participants[ptp].UUID)
				if err != nil {
					e.Logger.Error(err.Error())
				}
				events[i].Participants[ptp].Data = &data
			}
		}

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
Get Events (GET)
- 	Grabs events by club ID

Returns:

	Http handler
		- Writes an array of events back to client
*/
func (e *Service) GetEventsByField() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no field Id found in request." }`))
			return
		}
		oid, _ := primitive.ObjectIDFromHex(id)
		filter := bson.M{"field_id": oid}

		var events []models.Event
		err := e.FindEvents(context.Background(), filter, &events)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				rw.WriteHeader(http.StatusNotFound)
				rw.Write([]byte(`{ "msg": "field does not exist" }`))
				return
			}
		}

		var field models.Field
		users := make(map[string]models.UserData)
		clubs := make(map[primitive.ObjectID]models.Club)
		organizations := make(map[primitive.ObjectID]models.Organization)

		err = e.Database.FieldCol.FindOne(context.Background(), bson.M{"_id": oid}).Decode(&field)
		if err != nil {
			e.Logger.Error("Failed to fetch field: " + err.Error())
		}

		data := models.EventData{
			Field: &field,
		}
		dataClubs := []models.Club{}
		dataOrganizations := []models.Organization{}

		for i := range events {

			// fetch user data if not in users map
			u, ok := users[events[i].Poster]
			if !ok {
				user, err := e.SearchService.SearchUserByUUID(events[i].Poster)
				if err != nil {
					e.Logger.Error(err.Error())
				} else {
					users[events[i].Poster] = user
					data.Poster = &user
				}
			} else {
				data.Poster = &u
			}

			// fetch club/organization data if not in their respective maps
			for j := range events[i].Organizers {
				if events[i].Organizers[j].Type == "club" {
					id := events[i].Organizers[j].ID
					c, ok := clubs[id]
					if !ok {
						var club models.Club
						err := e.Database.ClubCol.FindOne(context.Background(), bson.M{"_id": id}).Decode(&club)
						if err != nil {
							e.Logger.Error(err.Error())
						} else {
							clubs[id] = club
							dataClubs = append(dataClubs, club)
						}
					} else {
						dataClubs = append(dataClubs, c)
					}
				} else {
					id := events[i].Organizers[j].ID
					o, ok := organizations[id]
					if !ok {
						var org models.Organization
						err := e.Database.OrgCol.FindOne(context.Background(), bson.M{"_id": id}).Decode(&org)
						if err != nil {
							e.Logger.Error(err.Error())
						} else {
							organizations[id] = org
							dataOrganizations = append(dataOrganizations, org)
						}
					} else {
						dataOrganizations = append(dataOrganizations, o)
					}
				}
			}

			data.Clubs = &dataClubs
			data.Organizations = &dataOrganizations
			events[i].Data = &data

			// add user data to participants
			for ptp := range events[i].Participants {
				data, err := e.SearchService.SearchUserByUUID(events[i].Participants[ptp].UUID)
				if err != nil {
					e.Logger.Error(err.Error())
				}
				events[i].Participants[ptp].Data = &data
			}
		}

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
		if req.StartTime != 0 {
			changes["start_time"] = req.StartTime
		}
		if req.ActualStartTime != 0 {
			changes["actual_start_time"] = req.ActualStartTime
		}
		if req.ActualStopTime != 0 {
			changes["actual_stop_time"] = req.ActualStopTime
		}
		if req.Visibility != "" {
			changes["visibility"] = req.Visibility
		}

		changes["min_participants"] = req.MinParticipants
		changes["max_participants"] = req.MaxParticipants
		changes["level"] = req.Level
		changes["external_link"] = req.ExternalLink

		if req.StopTime == 0 {
			updates["$unset"] = bson.M{"stop_time": 1}
		} else {
			changes["stop_time"] = req.StopTime
		}

		var event models.Event

		err = e.UpdateEvent(context.Background(), filter, updates, &event)
		if err != nil {
			e.Logger.Error(err)
			http.Error(rw, "failed to update event", http.StatusInternalServerError)
			return
		}

		// notify participants
		if req.ActualStartTime != 0 {
			// notify participants that the event is starting
			note := notif.Notification{
				Title: event.Title,
				Body:  "Event is starting",
				Topic: event.ID.Hex(),
			}
			e.NotifService.SendNotificationToTopic(&note)

		} else if req.ActualStopTime != 0 {
			// notify participants that the event ended
			note := notif.Notification{
				Title: event.Title,
				Body:  "Event ended",
				Topic: event.ID.Hex(),
			}
			e.NotifService.SendNotificationToTopic(&note)
			e.NotifService.DeleteTopic(event.ID.Hex())

		} else {
			// notify participants that the details changes
			note := notif.Notification{
				Title: event.Title,
				Body:  "Event details has changed",
				Topic: event.ID.Hex(),
			}
			e.NotifService.SendNotificationToTopic(&note)

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
		if event.MaxParticipants != 0 {
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

/*
Notify Event Participants

  - Notifies all of event's participants
*/
func (e *Service) NotifyParticipants() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// grab id from path
		vars := mux.Vars(r)
		if len(vars["id"]) < 24 {
			http.Error(rw, "bad event id", http.StatusBadRequest)
			return
		}

		id := vars["id"]

		// decode request
		var req notif.Notification
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		req.Topic = id

		e.NotifService.SendNotificationToTopic(&req)
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
	}
}

/*
Notify Club members

  - Notifies all of club members about event and to RSVP
*/
func (e *Service) NotifyClubMembers() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// grab id from path
		vars := mux.Vars(r)
		if len(vars["id"]) < 24 {
			http.Error(rw, "bad event id", http.StatusBadRequest)
			return
		}

		id := vars["id"]

		// decode request
		var req notif.Notification
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		req.Topic = id

		e.NotifService.SendNotificationToTopic(&req)
		rw.Header().Set("Content-Type", "application/json")
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
				"$near": bson.M{
					"$geometry":    loc,
					"$maxDistance": radius,
				},
			},
			"sports": bson.M{
				"$in": splicedSports,
			},
		}

		clubs := utils.NewSafeClub()
		organizations := utils.NewSafeOrganization()
		fields := utils.NewSafeFields()
		fieldsArr := []models.Field{}
		users := utils.NewSafeUsers()
		// timestamp := time.Now().Unix()

		// fetch the nearby fields
		cursor, err := e.Database.FieldCol.Find(context.Background(), filter)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(rw, "no events found", http.StatusNotFound)
				return
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

		var exists bool

		// status filters
		if status == "live" {
			exists = false
		} else if status == "ended" {
			exists = true
		}

		// filter to find events
		filter = bson.M{
			"field._id": bson.M{
				"$in": fieldsIDs,
			},
			"sport": bson.M{
				"$in": splicedSports,
			},
			// "stop_time": bson.M{
			// 	"$gt": timestamp,
			// },
			"actual_stop_time": bson.M{
				"$exists": exists,
			},
			"visibility": "public",
		}

		var events []models.Event
		err = e.FindEvents(context.Background(), filter, &events)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				rw.WriteHeader(http.StatusNotFound)
				rw.Write([]byte(`{ "msg": "field does not exist" }`))
				return
			}
		}

		var wg sync.WaitGroup

		// fetch owner data for all events
		for i := range events {

			data := models.EventData{}
			dataClubs := []models.Club{}
			dataOrganizations := []models.Organization{}

			wg.Add(1)
			go func(i int) {
				defer wg.Done()

				// fetch user data if not in users map
				user := users.FindUser(events[i].Poster)
				if user == nil {
					user, err := e.SearchService.SearchUserByUUID(events[i].Poster)
					if err != nil {
						e.Logger.Error(err.Error())
					} else {
						users.AddUser(&user)
						data.Poster = &user
					}
				}

				// fetch club/organization data if not in their respective maps
				for j := range events[i].Organizers {
					organizer := events[i].Organizers[j]
					if organizer.Type == "club" {
						id := organizer.ID
						c := clubs.FindClub(id)
						if c == nil {
							var club models.Club
							err := e.Database.ClubCol.FindOne(context.Background(), bson.M{"_id": id}).Decode(&club)
							if err != nil {
								e.Logger.Error(err.Error())
							} else {
								clubs.AddClub(&club)
								dataClubs = append(dataClubs, club)
							}
						} else {
							dataClubs = append(dataClubs, *c)
						}
					} else if organizer.Type == "organization" {
						id := organizer.ID
						o := organizations.FindOrganization(id)
						if o == nil {
							var org models.Organization
							err := e.Database.OrgCol.FindOne(context.Background(), bson.M{"_id": id}).Decode(&org)
							if err != nil {
								e.Logger.Error(err.Error())
							} else {
								organizations.AddOrganization(&org)
								dataOrganizations = append(dataOrganizations, org)
							}
						} else {
							dataOrganizations = append(dataOrganizations, *o)
						}
					}
				}

				data := models.EventData{
					Poster:        user,
					Field:         fields.FindField(events[i].Field.ID),
					Clubs:         &dataClubs,
					Organizations: &dataOrganizations,
				}
				events[i].Data = &data
			}(i)

			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				// add user data to participants
				for ptp := range events[i].Participants {
					data, err := e.SearchService.SearchUserByUUID(events[i].Participants[ptp].UUID)
					if err != nil {
						e.Logger.Error(err.Error())
					}
					events[i].Participants[ptp].Data = &data
				}
			}(i)
		}

		wg.Wait()

		if len(events) == 0 {
			http.Error(rw, "no events", http.StatusNoContent)
			return
		}

		resp := models.LocationResponse{
			Fields: &fieldsArr,
			Events: &events,
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(resp)
	}
}
