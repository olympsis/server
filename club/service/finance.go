package service

import (
	"context"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

/*
	Financial Database Operations
*/

// FindFinancialAccount finds a club's financial account by filter
func (s *Service) FindFinancialAccount(ctx context.Context, filter bson.M) (*models.ClubFinancialAccount, error) {
	collection := s.Database.ClubFinancialAccountsCollection
	var account models.ClubFinancialAccount

	err := collection.FindOne(ctx, filter).Decode(&account)
	if err != nil {
		return nil, err
	}

	return &account, nil
}

// InsertFinancialAccount creates a new financial account record
func (s *Service) InsertFinancialAccount(ctx context.Context, account *models.ClubFinancialAccount) (bson.ObjectID, error) {
	collection := s.Database.ClubFinancialAccountsCollection

	result, err := collection.InsertOne(ctx, account)
	if err != nil {
		return bson.NilObjectID, err
	}

	return result.InsertedID.(bson.ObjectID), nil
}

// UpdateFinancialAccount updates a financial account record
func (s *Service) UpdateFinancialAccount(ctx context.Context, filter bson.M, update bson.M) error {
	collection := s.Database.ClubFinancialAccountsCollection

	_, err := collection.UpdateOne(ctx, filter, update)
	return err
}

// FindTransaction finds a specific transaction by filter
func (s *Service) FindTransaction(ctx context.Context, filter bson.M) (*models.ClubTransaction, error) {
	collection := s.Database.ClubTransactionsCollection
	var transaction models.ClubTransaction

	err := collection.FindOne(ctx, filter).Decode(&transaction)
	if err != nil {
		return nil, err
	}

	return &transaction, nil
}

// FindTransactions finds multiple transactions with filters and pagination
func (s *Service) FindTransactions(ctx context.Context, filter bson.M, limit, skip int64) (*[]models.ClubTransaction, error) {
	collection := s.Database.ClubTransactionsCollection
	var transactions []models.ClubTransaction

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	err = cursor.All(ctx, &transactions)
	if err != nil {
		return nil, err
	}

	return &transactions, nil
}

// InsertTransaction creates a new transaction record
func (s *Service) InsertTransaction(ctx context.Context, transaction *models.ClubTransaction) (bson.ObjectID, error) {
	collection := s.Database.ClubTransactionsCollection

	result, err := collection.InsertOne(ctx, transaction)
	if err != nil {
		return bson.NilObjectID, err
	}

	return result.InsertedID.(bson.ObjectID), nil
}

// UpdateTransaction updates a transaction record
func (s *Service) UpdateTransaction(ctx context.Context, filter bson.M, update bson.M) error {
	collection := s.Database.ClubTransactionsCollection

	_, err := collection.UpdateOne(ctx, filter, update)
	return err
}
