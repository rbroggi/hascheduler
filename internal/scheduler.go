package internal

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

type store interface {
	FindSchedules(
		ctx context.Context,
	) ([]*Schedule, error)
	WatchSchedules(ctx context.Context) (
		<-chan ChangeEvent[Schedule],
		error,
	)
}

type Scheduler struct {
	scheduler gocron.Scheduler
	elector   gocron.Elector
	store     store
	// bookkeeping
	// internal schedule mongo id to scheduler in-memory id (glabal to local)
	globalToLocalId map[string]uuid.UUID
	// scheduler in-memory id to internal schedule mongo id (local to global)
	localToGlobalId map[uuid.UUID]string
	rwm             sync.RWMutex
}

func NewScheduler(
	elector gocron.Elector,
	store store,
) (*Scheduler, error) {
	scheduler, err := gocron.NewScheduler(gocron.WithDistributedElector(elector))
	if err != nil {
		return nil, fmt.Errorf("error creating scheduler: %w", err)
	}
	return &Scheduler{
		scheduler:       scheduler,
		elector:         elector,
		store:           store,
		globalToLocalId: make(map[string]uuid.UUID),
		localToGlobalId: make(map[uuid.UUID]string),
	}, nil
}

// Start is a blocking call that starts the scheduler and watches for updates on
// the persisted schedules.
//
// For graceful shutdown, cancel the input context
func (s *Scheduler) Start(ctx context.Context) error {
	slog.Info("Starting scheduler")
	// all stored schedules
	storedSchedules, err := s.store.FindSchedules(ctx)
	if err != nil {
		return err
	}
	for _, storedSch := range storedSchedules {
		if err := s.upsertSchedule(*storedSch); err != nil {
			slog.
				With("error", err).
				With("schedule", storedSch).
				Error("could not schedule schedule - ignoring it and proceeding")

		}
	}
	s.scheduler.Start()
	defer func() {
		if err := s.scheduler.Shutdown(); err != nil {
			slog.With("error", err).Error("failed to gracefully shutdown scheduler")
		}
	}()

	// watch schedule changes
	return s.watchSchedules(ctx)
}

func (s *Scheduler) toDefinition(schedule Schedule) gocron.JobDefinition {
	switch schedule.Type {
	case ScheduleTypeCron:
		return gocron.CronJob(schedule.ScheduleDefinition.CronExpression, parsableWithSeconds(schedule.ScheduleDefinition.CronExpression))
	case ScheduleTypeAtTimes:
		return gocron.OneTimeJob(gocron.OneTimeJobStartDateTimes(schedule.ScheduleDefinition.Times...))
	case ScheduleTypeDuration:
		return gocron.DurationJob(time.Duration(schedule.ScheduleDefinition.Interval))
	}
	slog.With("schedule", schedule).Error("invalid/unsupported definition")
	return nil
}

func parsableWithSeconds(expression string) bool {
	p := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	_, err := p.Parse(expression)
	return err == nil
}

func (s *Scheduler) job(schedule Schedule) func() error {
	return func() error {
		slog.With("schedule", schedule).Info("running schedule")
		return nil
	}
}

// watchSchedules is a blocking call that terminates upon canceling incoming context
func (s *Scheduler) watchSchedules(ctx context.Context) error {
	scheduleChanges, err := s.store.WatchSchedules(ctx)
	if err != nil {
		return fmt.Errorf("error watching schedules: %w", err)
	}
	for changeEvent := range scheduleChanges {
		slog.With("changeEvent", changeEvent).Info("received schedule change event")
		switch changeEvent.Operation {
		case Insert, Update:
			if err := s.upsertSchedule(*changeEvent.Data); err != nil {
				slog.
					With("error", err).
					With("schedule", changeEvent.Data).
					Error("error upserting schedule")
			}
		case Delete:
			if err := s.removeSchedule(changeEvent.ID); err != nil {
				slog.
					With("error", err).
					With("schedule", changeEvent.Data).
					Error("error removing schedule")

			}
		default:
			slog.
				With("operation", changeEvent.Operation).
				Error("invalid operation in change event")
		}
	}
	return nil
}

func (s *Scheduler) upsertSchedule(sch Schedule) error {
	localID, ok := s.globalToLocalId[sch.ID]
	definition := s.toDefinition(sch)
	if definition == nil {
		return fmt.Errorf("unsupported definition for schedule id %s", sch.ID)
	}
	lgr := slog.
		With("schedule.id", sch.ID).
		With("schedule.name", sch.Name).
		With("schedule.definition", sch.ScheduleDefinition)
	// update
	if ok {
		job, err := s.scheduler.Update(localID,
			definition,
			gocron.NewTask(func() error {
				slog.With("schedule", sch).Info("running schedule")
				return nil
			}),
			gocron.WithName(sch.Name),
		)
		if err != nil {
			return fmt.Errorf("while upserting schedule [id: %s, name: %s]: %w", sch.ID, sch.Name, err)
		}
		lgr.With("local.id", job.ID().String()).Info("updated scheduled job")
		return nil
	}
	// insert
	job, err := s.scheduler.NewJob(
		definition,
		gocron.NewTask(func() error {
			slog.With("schedule", sch).Info("running schedule")
			return nil
		}),
		gocron.WithName(sch.Name),
	)
	if err != nil {
		return fmt.Errorf("error adding new entry to cron: %w", err)
	}
	jobID := job.ID()
	s.globalToLocalId[sch.ID] = jobID
	s.localToGlobalId[jobID] = sch.ID
	lgr.With("local.id", jobID.String()).Info("added scheduled job")
	return nil
}

func (s *Scheduler) removeSchedule(id string) error {
	s.rwm.Lock()
	defer s.rwm.Unlock()
	localID, ok := s.globalToLocalId[id]
	if !ok {
		return nil
	}
	if err := s.scheduler.RemoveJob(localID); err != nil {
		return fmt.Errorf("error removing job with id %s: %w", localID.String(), err)
	}
	delete(s.globalToLocalId, id)
	delete(s.localToGlobalId, localID)
	slog.With("local.id", localID).
		With("schedule.id", id).
		Info("removed scheduled job")
	return nil
}
