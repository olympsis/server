package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (s *Service) AddComment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		uuid := r.Header.Get("userID")

		// Grab event id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(w, `{"msg": "bad event id"}`, http.StatusBadRequest)
			return
		}
		oid, err := bson.ObjectIDFromHex(id)
		if err != nil {
			s.Logger.Error("Failed to convert id to ObjectID. Error: ", err.Error())
			http.Error(w, `{"msg": "failed to encode id"}`, http.StatusBadRequest)
			return
		}

		// Decode request
		var req models.EventCommentDao
		err = json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			s.Logger.Error("Failed to decode request. Error: ", err.Error())
			http.Error(w, `{"msg":"failed to decode request"}`, http.StatusBadRequest)
			return
		}

		// Add comment to the database
		timestamp := bson.NewDateTimeFromTime(time.Now())
		req.UserID = &uuid
		req.EventID = &oid
		req.CreatedAt = &timestamp
		cid, err := s.InsertComment(ctx, &req)
		if err != nil {
			s.Logger.Error("Failed to insert comment into the database. Error: ", err.Error())
			http.Error(w, `{"msg": "failed to create comment"}`, http.StatusInternalServerError)
		}

		// Notify participants
		if err = s.Notification.NewEventComment(oid, *req.Text); err != nil {
			s.Logger.Errorf("Failed to notify event participants. Event ID: %s - Error: %s", id, err.Error())
		}

		w.WriteHeader(http.StatusCreated)
		w.Write(fmt.Appendf(nil, `{"id": "%s"}`, cid.Hex()))
	}
}

func (s *Service) RemoveComment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// Grab comment id from path
		vars := mux.Vars(r)
		id := vars["commentID"]
		if len(id) < 24 {
			http.Error(w, `{"msg": "bad comment id"}`, http.StatusBadRequest)
			return
		}
		oid, err := bson.ObjectIDFromHex(id)
		if err != nil {
			s.Logger.Error("Failed to convert id to ObjectID. Error: ", err.Error())
			http.Error(w, `{"msg": "failed to encode id"}`, http.StatusBadRequest)
			return
		}

		// Delete comment from db
		err = s.DeleteComment(ctx, bson.M{"_id": oid})
		if err != nil {
			s.Logger.Error("Failed to delete comment. Error: ", err.Error())
			http.Error(w, `{"msg": "failed to remove comment"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"msg": "OK"}`))
	}
}

/*****************
DATABASE FUNCTIONS
******************/

// Insert new comment into database
func (s *Service) InsertComment(ctx context.Context, comment *models.EventCommentDao) (bson.ObjectID, error) {
	resp, err := s.Database.EventCommentsCollection.InsertOne(ctx, comment)
	if err != nil {
		return bson.NilObjectID, err
	}
	id := resp.InsertedID.(bson.ObjectID)
	return id, nil
}

// Insert multiple comments into database
func (s *Service) InsertComments(ctx context.Context, comments []any) ([]bson.ObjectID, error) {
	resp, err := s.Database.EventCommentsCollection.InsertMany(ctx, comments)
	if err != nil {
		return nil, err
	}

	// Convert inserted IDs to ObjectIDs
	ids := make([]bson.ObjectID, len(resp.InsertedIDs))
	for i, id := range resp.InsertedIDs {
		ids[i] = id.(bson.ObjectID)
	}

	return ids, nil
}

// Get comment from database
func (s *Service) FindComment(ctx context.Context, filter bson.M) (*models.EventCommentDao, error) {
	var comment models.EventCommentDao
	err := s.Database.EventCommentsCollection.FindOne(ctx, filter).Decode(&comment)
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// Get comments from database
func (s *Service) FindComments(ctx context.Context, filter bson.M, limit int64, skip int64) ([]models.EventCommentDao, error) {
	opts := options.Find()
	if limit > 0 {
		opts.SetLimit(limit)
	}
	if skip > 0 {
		opts.SetSkip(skip)
	}
	// Sort by creation time, newest first
	opts.SetSort(bson.M{"created_at": -1})

	var comments []models.EventCommentDao
	cursor, err := s.Database.EventCommentsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &comments); err != nil {
		return nil, err
	}
	return comments, nil
}

// Update comment in database
func (s *Service) UpdateComment(ctx context.Context, filter bson.M, update bson.M) error {
	_, err := s.Database.EventCommentsCollection.UpdateOne(ctx, filter, update)
	return err
}

// Update many comments in database
func (s *Service) UpdateComments(ctx context.Context, filter bson.M, update bson.M) error {
	_, err := s.Database.EventCommentsCollection.UpdateMany(ctx, filter, update)
	return err
}

// Delete comment from database
func (s *Service) DeleteComment(ctx context.Context, filter bson.M) error {
	_, err := s.Database.EventCommentsCollection.DeleteOne(ctx, filter)
	return err
}

// Delete many comments from database
func (s *Service) DeleteComments(ctx context.Context, filter bson.M) (int64, error) {
	result, err := s.Database.EventCommentsCollection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}
