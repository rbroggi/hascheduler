package internal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

const (
	defaultNamespace = "default"
	defaultLeaseName = "hascheduler-lease"
)

func NewElector() (*Elector, error) {
	identity := os.Getenv("HOSTNAME")
	if identity == "" {
		identity = uuid.New().String()
	}
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
	}
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	leaseName := os.Getenv("LEASE_KEY")
	if leaseName == "" {
		leaseName = defaultLeaseName
	}

	l, err := resourcelock.New(
		resourcelock.LeasesResourceLock,
		defaultNamespace,
		leaseName,
		kubeClient.CoreV1(),
		kubeClient.CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity: identity,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource lock: %w", err)
	}

	leaderElectionConfig := &leaderelection.LeaderElectionConfig{
		Lock:          l,
		LeaseDuration: 5 * time.Second,
		RenewDeadline: 3 * time.Second,
		RetryPeriod:   1 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				slog.Info("Started leading", "candidate", identity)
			},
			OnStoppedLeading: func() {
				slog.Info("Stopped leading", "candidate", identity)
			},
			OnNewLeader: func(currentLeaderIdentity string) {
				slog.Info("New leader elected", "candidate", identity, "leader", identity)
			},
		},
		// to ensure faster leader transfer
		ReleaseOnCancel: true,
	}

	// 2. Prepare the leader election configuration
	le, err := leaderelection.NewLeaderElector(*leaderElectionConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create leader elector: %w", err)
	}

	return &Elector{elector: le}, nil
}

type Elector struct {
	elector *leaderelection.LeaderElector
}

func (el *Elector) Run(ctx context.Context) {
	el.elector.Run(ctx)
}

func (el *Elector) IsLeader(ctx context.Context) error {
	if el.elector.IsLeader() {
		return nil
	}
	return errors.New("not a leader")
}
