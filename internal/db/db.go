package db

import (
	"context"
	"fmt"
	"github.com/gmkornilov/chess-puzzle-book-backend/internal/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TaskDbClient struct {
	client         *mongo.Client
	TaskCollection *mongo.Collection
}

func (r *TaskDbClient) Close() error {
	return r.client.Disconnect(context.TODO())
}

func NewDbClient(cfg *config.Configuration) (*TaskDbClient, error) {
	clientOpts := options.Client().ApplyURI(cfg.Database.Address)

	dbClient := &TaskDbClient{}

	client, err := mongo.Connect(context.TODO(), clientOpts)
	if err != nil {
		return nil, err
	}
	dbClient.client = client

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		return nil, err
	}

	dbClient.TaskCollection = client.Database(cfg.Database.DatabaseName).Collection(cfg.Database.Collection)
	if dbClient.TaskCollection == nil {
		return nil, fmt.Errorf("Can't resolve collection %s", cfg.Database.DatabaseName + "." + cfg.Database.Collection)
	}
	return dbClient, nil
}
