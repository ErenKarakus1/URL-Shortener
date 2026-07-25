package main

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var ErrShortURLCollision = errors.New("short URL belongs to a different long URL")

type MongoStore struct {
	collection *mongo.Collection
}

func NewMongoStore(collection *mongo.Collection) *MongoStore {
	return &MongoStore{collection: collection}
}

func (s *MongoStore) CreateIndexes(ctx context.Context) error {
	index := mongo.IndexModel{
		Keys:    bson.D{{Key: "shorturl", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	_, err := s.collection.Indexes().CreateOne(ctx, index)
	return err
}

func (s *MongoStore) Save(ctx context.Context, url URLData) (bool, error) {
	existing, err := s.FindByShortURL(ctx, url.ShortURL)
	if err == nil {
		if existing.LongURL == url.LongURL {
			return false, nil
		}
		return false, ErrShortURLCollision
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return false, err
	}

	_, err = s.collection.InsertOne(ctx, url)
	if mongo.IsDuplicateKeyError(err) {
		existing, findErr := s.FindByShortURL(ctx, url.ShortURL)
		if findErr != nil {
			return false, findErr
		}
		if existing.LongURL == url.LongURL {
			return false, nil
		}
		return false, ErrShortURLCollision
	}
	return err == nil, err
}

func (s *MongoStore) FindByShortURL(ctx context.Context, shortURL string) (URLData, error) {
	var url URLData
	err := s.collection.FindOne(ctx, bson.M{"shorturl": shortURL}).Decode(&url)
	return url, err
}
