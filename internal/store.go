package internal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

const (
	operationTimeout        = 5 * time.Second
	schedulesCollectionName = "schedules"
)

type Store struct {
	schedules *mongo.Collection
}

func NewStore(db *mongo.Database) *Store {
	wc := writeconcern.Majority()
	wc.WTimeout = operationTimeout
	connectionOpts := options.Collection().SetWriteConcern(wc)
	return &Store{
		schedules: db.Collection(schedulesCollectionName, connectionOpts),
	}
}

// CreateSchedule creates a new schedule and will return ErrScheduleAlreadyExists if it already exists
func (s *Store) CreateSchedule(ctx context.Context, schedule *Schedule) error {
	ctxTimeout, cnl := context.WithTimeout(ctx, operationTimeout)
	defer cnl()
	schedule.ID = uuid.New().String()
	_, err := s.schedules.InsertOne(ctxTimeout, schedule)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return errScheduleAlreadyExists
		}
		return fmt.Errorf("create schedule failed: %w", err)
	}
	return nil
}

func (s *Store) FindSchedules(ctx context.Context) ([]*Schedule, error) {
	ctxTimeout, cnl := context.WithTimeout(ctx, operationTimeout)
	defer cnl()
	filter := bson.M{}
	opts := options.Find()
	cursor, err := s.schedules.Find(ctxTimeout, filter, opts)
	if err != nil {
		return nil, err
	}
	schedules := make([]*Schedule, 0)
	for cursor.Next(ctxTimeout) {
		var dto Schedule
		err = cursor.Decode(&dto)
		if err != nil {
			return nil, fmt.Errorf("failed to decode schedule: %w", err)
		}
		schedules = append(schedules, &dto)
	}
	return schedules, nil
}

func (s *Store) UpdateSchedule(
	ctx context.Context,
	schedule *Schedule,
) error {
	ctxTimeout, cnl := context.WithTimeout(ctx, operationTimeout)
	defer cnl()
	_, err := s.schedules.UpdateOne(ctxTimeout,
		bson.M{"_id": schedule.ID},
		schedule)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return errScheduleNotFound
		}
		return err
	}
	return nil
}

func (s *Store) DeleteSchedule(
	ctx context.Context,
	id string,
) (*Schedule, error) {
	ctxTimeout, cnl := context.WithTimeout(ctx, operationTimeout)
	defer cnl()

	filter := bson.M{"_id": id}

	res := s.schedules.FindOneAndDelete(ctxTimeout, filter)
	if err := res.Err(); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errScheduleNotFound
		}
		return nil, fmt.Errorf("error deleting schedule: %w", err)
	}

	dto := Schedule{}
	if err := res.Decode(&dto); err != nil {
		return nil, fmt.Errorf("failed to decode schedule: %w", err)
	}

	return &dto, nil
}

func (s *Store) WatchSchedules(ctx context.Context) (<-chan ChangeEvent[Schedule], error) {
	cs, err := s.schedules.Watch(
		ctx,
		mongo.Pipeline{},
		// full document on update (default is to have only the updated fields)
		options.ChangeStream().SetFullDocument(options.UpdateLookup),
	)
	if err != nil {
		return nil, fmt.Errorf("error watching schedules: %w", err)
	}
	return s.iterateChangeStreamSchedules(ctx, cs), nil
}

type scheduleChangeStreamDto struct {
	DocumentKey   documentKeyDto `bson:"documentKey"`
	OperationType string         `bson:"operationType"`
	FullDocument  *Schedule      `bson:"fullDocument"`
}

type documentKeyDto struct {
	ID string `bson:"_id,omitempty"`
}

func (s *Store) iterateChangeStreamSchedules(ctx context.Context, cs *mongo.ChangeStream) <-chan ChangeEvent[Schedule] {
	ch := make(chan ChangeEvent[Schedule], 100)
	go func() {
		defer close(ch)
		defer cs.Close(ctx)
		for cs.Next(ctx) {
			var dto scheduleChangeStreamDto
			if err := cs.Decode(&dto); err != nil {
				slog.With("error", err).Error("error decoding change stream element")
			} else {
				model := dto.FullDocument
				var op Operation
				switch dto.OperationType {
				case "insert":
					op = Insert
				case "update":
					op = Update
				case "delete":
					op = Delete
				default:
					slog.With("operationType", dto.OperationType).Error("invalid operation")
					continue
				}
				ch <- ChangeEvent[Schedule]{
					Operation: op,
					ID:        dto.DocumentKey.ID,
					Data:      model,
				}
			}
		}
	}()
	return ch
}
