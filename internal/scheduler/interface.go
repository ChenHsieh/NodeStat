package scheduler

import (
	"nodestat/internal/models"
)

// Scheduler defines the interface for different schedulers (SLURM, Torque)
type Scheduler interface {
	// GetNodes retrieves all nodes for a given partition
	GetNodes(partition string) ([]models.Node, error)

	// GetJobs retrieves running jobs for a given partition
	GetJobs(partition string) ([]models.Job, error)

	// GetPartitions retrieves available partitions
	GetPartitions() ([]string, error)

	// GetUserJobs retrieves jobs for a specific user
	GetUserJobs(user string) ([]models.Job, error)

	// GetSystemType returns the scheduler type
	GetSystemType() string
}

// SchedulerType represents different scheduler systems
type SchedulerType string

const (
	SLURM  SchedulerType = "slurm"
	Torque SchedulerType = "torque"
	Mock   SchedulerType = "mock"
)

// NewScheduler creates a new scheduler based on the type
func NewScheduler(schedulerType SchedulerType) Scheduler {
	switch schedulerType {
	case SLURM:
		return &SlurmScheduler{}
	case Torque:
		return &TorqueScheduler{}
	case Mock:
		return NewMockScheduler()
	default:
		return &SlurmScheduler{} // default to SLURM
	}
}
