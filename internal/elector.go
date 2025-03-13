package internal

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/rbroggi/leaderelection"
	"github.com/rbroggi/mongoleasestore"
	"go.mongodb.org/mongo-driver/mongo"
)

func NewElector(db *mongo.Database) (*Elector, error) {
	identity := os.Getenv("HOSTNAME")
	if identity == "" {
		identity = uuid.New().String()
	}
	s, err := mongoleasestore.NewStore(mongoleasestore.Args{
		LeaseCollection: db.Collection("leases"),
		LeaseKey:        "lease-key",
	})
	if err != nil {
		return nil, err
	}
	el, err := leaderelection.NewElector(leaderelection.ElectorConfig{
		LeaseDuration:   3 * time.Second,
		RetryPeriod:     300 * time.Millisecond,
		LeaseStore:      s,
		CandidateID:     identity,
		ReleaseOnCancel: true,
		OnStartedLeading: func(candidateIdentity string) {
			slog.Info("Started leading", "candidate", candidateIdentity)
		},
		OnStoppedLeading: func(candidateIdentity string) {
			slog.Info("Stopped leading", "candidate", candidateIdentity)
		},
		OnNewLeader: func(candidateIdentity string, newLeaderIdentity string) {
			slog.Info("New leader elected", "candidate", candidateIdentity, "leader", newLeaderIdentity)
		},
	})
	if err != nil {
		return nil, err
	}
	return &Elector{elector: el}, nil
}

func (el *Elector) Run(ctx context.Context) <-chan struct{} {
	return el.elector.Run(ctx)
}

type Elector struct {
	elector *leaderelection.Elector
}

func (e *Elector) IsLeader(ctx context.Context) error {
	if e.elector.IsLeader() {
		return nil
	}
	return errors.New("not a leader")
}
