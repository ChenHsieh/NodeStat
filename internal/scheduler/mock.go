package scheduler

import (
	"fmt"
	"math/rand"
	"time"

	"nodestat/internal/models"
)

// MockScheduler provides test data for development/demo purposes
type MockScheduler struct{}

// NewMockScheduler creates a new mock scheduler for testing
func NewMockScheduler() *MockScheduler {
	return &MockScheduler{}
}

// GetSystemType returns the scheduler type
func (m *MockScheduler) GetSystemType() string {
	return "mock"
}

// GetNodes returns mock node data
func (m *MockScheduler) GetNodes(partition string) ([]models.Node, error) {
	rand.Seed(time.Now().UnixNano())

	var nodeCount int
	var nodePrefix string

	switch partition {
	case "batch":
		nodeCount = 25
		nodePrefix = "batch"
	case "highmem_q":
		nodeCount = 8
		nodePrefix = "highmem"
	case "gpu_q":
		nodeCount = 6
		nodePrefix = "gpu"
	default:
		return nil, fmt.Errorf("unknown partition: %s", partition)
	}

	nodes := make([]models.Node, nodeCount)

	for i := 0; i < nodeCount; i++ {
		node := models.Node{
			ID:         fmt.Sprintf("%s%03d", nodePrefix, i+1),
			Partitions: []string{partition},
		}

		// Set node specs based on partition type
		switch partition {
		case "batch":
			node.TotalCores = 32 + rand.Intn(32)            // 32-64 cores
			node.TotalMemMB = (128 + rand.Intn(256)) * 1000 // 128-384 GB
		case "highmem_q":
			node.TotalCores = 48 + rand.Intn(16)             // 48-64 cores
			node.TotalMemMB = (512 + rand.Intn(1024)) * 1000 // 512-1536 GB
		case "gpu_q":
			node.TotalCores = 40 + rand.Intn(20)            // 40-60 cores
			node.TotalMemMB = (256 + rand.Intn(256)) * 1000 // 256-512 GB
		}

		// Randomly assign states
		states := []models.NodeState{
			models.StateIdle,
			models.StateRunning,
			models.StateRunning,
			models.StateDown,
			models.StateBusy,
		}
		node.State = states[rand.Intn(len(states))]

		// Set resource usage based on state
		if node.State == models.StateIdle {
			node.UsedCores = 0
			node.UsedMemMB = rand.Intn(node.TotalMemMB / 10) // minimal memory usage
		} else if node.State == models.StateRunning {
			node.UsedCores = rand.Intn(node.TotalCores)
			node.UsedMemMB = rand.Intn(node.TotalMemMB)
		} else if node.State == models.StateBusy {
			node.UsedCores = node.TotalCores // fully utilized
			node.UsedMemMB = node.TotalMemMB - rand.Intn(node.TotalMemMB/4)
		} else {
			// Down/offline nodes
			node.UsedCores = 0
			node.UsedMemMB = 0
		}

		// Generate some job IDs for running nodes
		if node.State == models.StateRunning && node.UsedCores > 0 {
			jobCount := 1 + rand.Intn(3)
			for j := 0; j < jobCount; j++ {
				node.Jobs = append(node.Jobs, fmt.Sprintf("%d", 100000+rand.Intn(999999)))
			}
		}

		nodes[i] = node
	}

	return nodes, nil
}

// GetJobs returns mock job data
func (m *MockScheduler) GetJobs(partition string) ([]models.Job, error) {
	rand.Seed(time.Now().UnixNano())

	jobCount := 10 + rand.Intn(20)
	jobs := make([]models.Job, jobCount)

	users := []string{"alice", "bob", "carol", "dave", "eve", "frank", "grace", "henry"}

	for i := 0; i < jobCount; i++ {
		job := models.Job{
			ID:        fmt.Sprintf("%d", 100000+rand.Intn(999999)),
			User:      users[rand.Intn(len(users))],
			Name:      fmt.Sprintf("job_%d", i+1),
			State:     models.JobRunning,
			Partition: partition,
			ReqNodes:  1 + rand.Intn(4),
			ReqCPUs:   8 + rand.Intn(32),
			ReqMemMB:  (16 + rand.Intn(128)) * 1000,
			Elapsed:   time.Duration(rand.Intn(86400)) * time.Second,
		}

		// Assign to random nodes (simplified)
		nodeNum := rand.Intn(20) + 1
		job.NodeList = []string{fmt.Sprintf("%s%03d", partition, nodeNum)}

		jobs[i] = job
	}

	return jobs, nil
}

// GetPartitions returns available partitions
func (m *MockScheduler) GetPartitions() ([]string, error) {
	return []string{"batch", "highmem_q", "gpu_q", "debug_q"}, nil
}

// GetUserJobs returns mock user jobs
func (m *MockScheduler) GetUserJobs(user string) ([]models.Job, error) {
	rand.Seed(time.Now().UnixNano())

	// Simulate 0-3 user jobs
	jobCount := rand.Intn(4)
	jobs := make([]models.Job, jobCount)

	for i := 0; i < jobCount; i++ {
		jobs[i] = models.Job{
			ID:        fmt.Sprintf("%d", 200000+rand.Intn(999999)),
			User:      user,
			Name:      fmt.Sprintf("my_job_%d", i+1),
			State:     models.JobRunning,
			Partition: "batch",
			ReqNodes:  1,
			ReqCPUs:   4 + rand.Intn(16),
			ReqMemMB:  (8 + rand.Intn(64)) * 1000,
			Elapsed:   time.Duration(rand.Intn(43200)) * time.Second,
		}

		// Assign to specific nodes for highlighting
		nodeNum := rand.Intn(10) + 1
		jobs[i].NodeList = []string{fmt.Sprintf("batch%03d", nodeNum)}
	}

	return jobs, nil
}
