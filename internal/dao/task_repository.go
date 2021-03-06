package dao

import (
	"context"
	"fmt"
	"github.com/gmkornilov/chess-puzzle-book-backend/internal/db"
	"github.com/gmkornilov/chess-puzzle-book-backend/pkg/puzgen"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"math"
	"time"
)

type TaskRepository interface {
	GetRandomTaskForElo(elo int) (puzgen.Task, error)

	InsertTask(task puzgen.Task) error

	InsertAllTasks(tasks []puzgen.Task) error

	GetFirstUserTask(username string) (puzgen.Task, error)

	GetLastUserTask(username string) (puzgen.Task, error)

	GetLastUserTasks(username string, n int64) ([]puzgen.Task, int, error)

	GetUserTasksBetweenDates(username string, startTime primitive.DateTime, endTime primitive.DateTime) ([]puzgen.Task, error)
}

const batchSize = 20

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

func (t *taskRepository) InsertAllTasks(tasks []puzgen.Task) error {
	ctx, cancel := context.WithTimeout(context.TODO(), 20*time.Second)
	defer cancel()

	for i := 0; i < len(tasks); i += batchSize {
		sz := int(math.Min(float64(len(tasks)-i), float64(batchSize)))
		toInsert := make([]interface{}, sz)
		for j := 0; j < sz; j++ {
			toInsert[j] = tasks[i+j]
		}
		_, err := t.dbClient.TaskCollection.InsertMany(ctx, toInsert)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *taskRepository) GetLastUserTasks(username string, n int64) ([]puzgen.Task, int, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	matchStage := bson.D{
		{"$match", bson.D{
			{"$or", bson.A{
				bson.D{{"game_data.white_player", username}},
				bson.D{{"game_data.black_player", username}},
			}},
		}},
	}
	groupStage := bson.D{
		{"$group", bson.D{
			{"_id", "$game_data.date"},
			{"data", bson.D{
				{"$push", "$$ROOT"},
			}},
		}},
	}
	sortStage := bson.D{
		{"$sort", bson.D{
			{"_id", -1},
		}},
	}
	limitStage := bson.D{
		{"$limit", n},
	}
	nullGroupStage := bson.D{
		{"$group", bson.D{
			{"_id", "null"},
			{"data", bson.D{
				{"$push", "$data"},
			}},
			{"count", bson.D{
				{"$sum", 1},
			}},
		}},
	}
	projectStage := bson.D{
		{"$project", bson.D{
			{"result", bson.D{
				{"$reduce", bson.D{
					{"input", "$data"},
					{"initialValue", bson.A{}},
					{"in", bson.D{
						{"$concatArrays", bson.A{"$$value", "$$this"}},
					}},
				}},
			}},
			{"_id", 0},
			{"count", 1},
		}},
	}
	cur, err := t.dbClient.TaskCollection.Aggregate(ctx, mongo.Pipeline{matchStage, groupStage, sortStage, limitStage, nullGroupStage, projectStage})
	if err != nil {
		return nil, 0, err
	}
	var result []struct {
		Result []puzgen.Task `bson:"result"`
		Count  int           `bson:"count"`
	}
	if err := cur.All(ctx, &result); err != nil {
		return nil, 0, err
	}
	if len(result) != 1 {
		return nil, 0, fmt.Errorf("null aggregate returned more than one group(???)")
	}

	return result[0].Result, result[0].Count, nil
}

func (t *taskRepository) GetLastUserTask(username string) (puzgen.Task, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	opts := options.FindOne()
	opts.SetSort(bson.D{{"game_data.date", -1}})

	filter := bson.D{
		{"$or", bson.A{
			bson.D{{"game_data.white_player", username}},
			bson.D{{"game_data.black_player", username}},
		},
		}}
	cur := t.dbClient.TaskCollection.FindOne(ctx, filter, opts)
	var task puzgen.Task
	if err := cur.Decode(&task); err != nil {
		if err == mongo.ErrNoDocuments {
			return puzgen.Task{}, nil
		}
		return puzgen.Task{}, err
	}
	return task, nil
}

func (t *taskRepository) GetFirstUserTask(username string) (puzgen.Task, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	opts := options.FindOne()
	opts.SetSort(bson.D{{"game_data.date", 1}})

	filter := bson.D{
		{"$or", bson.A{
			bson.D{{"game_data.white_player", username}},
			bson.D{{"game_data.black_player", username}},
		},
		}}
	cur := t.dbClient.TaskCollection.FindOne(ctx, filter, opts)
	var task puzgen.Task
	if err := cur.Decode(&task); err != nil {
		if err == mongo.ErrNoDocuments {
			return puzgen.Task{}, nil
		}
		return puzgen.Task{}, err
	}
	return task, nil
}

func (t *taskRepository) GetUserTasksBetweenDates(username string, startTime primitive.DateTime, endTime primitive.DateTime) ([]puzgen.Task, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	filter := bson.D{
		{"$or", bson.A{
			bson.D{{"game_data.white_player", username}},
			bson.D{{"game_data.black_player", username}},
		}},
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
