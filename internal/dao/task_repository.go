package dao

import (
	"context"
	"fmt"
	"github.com/gmkornilov/chess-puzzle-generator/internal/db"
	"github.com/gmkornilov/chess-puzzle-generator/pkg/puzgen"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type TaskRepository interface {
	GetRandomTaskForElo(elo int) (puzgen.Task, error)

	InsertTask(task puzgen.Task) error

	GetLastUserTask(username string) (puzgen.Task, error)

	GetUserTasksBetweenDates(startTime primitive.DateTime, endTime primitive.DateTime) ([]puzgen.Task, error)
}

type taskRepository struct {
	dbClient *db.TaskDbClient
}

func NewTaskRepository(dbClient *db.TaskDbClient) TaskRepository {
	return &taskRepository{dbClient}
}

func (t *taskRepository) GetRandomTaskForElo(elo int) (puzgen.Task, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	matchStage := bson.D{{"$match", bson.D{{
		"target_elo", bson.D{{"$gte", elo - 100}, {"$lte", elo + 100}},
	}}}}
	sampleStage := bson.D{{"$sample", bson.D{{"size", 1}}}}


	cursor, err := t.dbClient.TaskCollection.Aggregate(ctx, mongo.Pipeline{matchStage, sampleStage})
	if err != nil {
		return puzgen.Task{}, err
	}

	var loadedTasks []puzgen.Task
	if err = cursor.All(ctx, &loadedTasks); err != nil {
		return puzgen.Task{}, err
	}
	if len(loadedTasks) != 1 {
		return puzgen.Task{}, fmt.Errorf("aggregate with $size = 1 returned more than 1 samples or no samples")
	}
	return loadedTasks[0], nil
}

func (t *taskRepository) InsertTask(task puzgen.Task) error {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	_, err := t.dbClient.TaskCollection.InsertOne(ctx, task)
	return err
}

func (t *taskRepository) GetLastUserTask(username string) (puzgen.Task, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	opts := options.FindOne()
	opts.SetSort(bson.D{{"game_data.date", -1}})

	filter := bson.D{
		{"$or", bson.D{
			{"game_data.white_player", username},
			{"game_data.black_player", username},
		},
	}}
	cur := t.dbClient.TaskCollection.FindOne(ctx, filter, opts)
	var task puzgen.Task
	if err := cur.Decode(&task); err != nil {
		return puzgen.Task{}, err
	}
	return task, nil
}

func (t *taskRepository) GetUserTasksBetweenDates(startTime primitive.DateTime, endTime primitive.DateTime) ([]puzgen.Task, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	filter := bson.D{
		{
			"game_data.date", bson.D{
					{"$gte", startTime},
					{"$lte", endTime},
				},
		},
	}

	cur, err := t.dbClient.TaskCollection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	var tasks []puzgen.Task
	if err = cur.All(ctx, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}
