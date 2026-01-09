package models

import "time"

// NodeState represents the state of a node
type NodeState string

const (
	StateIdle    NodeState = "Idle"
	StateRunning NodeState = "Running"
	StateDown    NodeState = "Down"
	StateOffline NodeState = "Offline"
	StateBusy    NodeState = "Busy"
	StateDrained NodeState = "Drained"
)

// Node represents a single compute node
type Node struct {
	ID         string    `json:"id"`
	State      NodeState `json:"state"`
	TotalCores int       `json:"total_cores"`
	UsedCores  int       `json:"used_cores"`
	TotalMemMB int       `json:"total_mem_mb"` // Memory in MB for precision
	UsedMemMB  int       `json:"used_mem_mb"`
	Partitions []string  `json:"partitions"`
	Jobs       []string  `json:"jobs"` // Job IDs running on this node
}

// GetAvailCores returns available CPU cores
func (n *Node) GetAvailCores() int {
	return n.TotalCores - n.UsedCores
}

// GetAvailMemGB returns available memory in GB
func (n *Node) GetAvailMemGB() int {
	return (n.TotalMemMB - n.UsedMemMB) / 1000
}

// GetTotalMemGB returns total memory in GB
func (n *Node) GetTotalMemGB() int {
	return n.TotalMemMB / 1000
}

// GetUsedMemGB returns used memory in GB
func (n *Node) GetUsedMemGB() int {
	return n.UsedMemMB / 1000
}

// IsAvailable returns true if node is available for jobs
func (n *Node) IsAvailable() bool {
	return (n.State == StateIdle || n.State == StateRunning) &&
		n.GetAvailCores() > 0 && n.GetAvailMemGB() > 0
}

// GetCPUUtilization returns CPU utilization as a percentage (0-100)
func (n *Node) GetCPUUtilization() float64 {
	if n.TotalCores == 0 {
		return 0
	}
	return float64(n.UsedCores) / float64(n.TotalCores)
}

// GetMemoryUtilization returns memory utilization as a percentage (0-100)
func (n *Node) GetMemoryUtilization() float64 {
	if n.TotalMemMB == 0 {
		return 0
	}
	return float64(n.UsedMemMB) / float64(n.TotalMemMB)
}

// JobState represents the state of a job
type JobState string

const (
	JobRunning   JobState = "R"
	JobPending   JobState = "PD"
	JobCompleted JobState = "C"
	JobCancelled JobState = "CA"
	JobFailed    JobState = "F"
)

// Job represents a single job
type Job struct {
	ID         string        `json:"id"`
	User       string        `json:"user"`
	Name       string        `json:"name"`
	State      JobState      `json:"state"`
	NodeList   []string      `json:"node_list"`
	Partition  string        `json:"partition"`
	ReqNodes   int           `json:"req_nodes"`
	ReqCPUs    int           `json:"req_cpus"`
	ReqMemMB   int           `json:"req_mem_mb"`
	TimeLimit  time.Duration `json:"time_limit"`
	Elapsed    time.Duration `json:"elapsed"`
	CPUTime    time.Duration `json:"cpu_time"`
	SubmitTime time.Time     `json:"submit_time"`
}

// GetReqMemGB returns requested memory in GB
func (j *Job) GetReqMemGB() int {
	return j.ReqMemMB / 1000
}

// Partition represents a cluster partition/queue
type Partition struct {
	Name       string `json:"name"`
	Nodes      []Node `json:"nodes"`
	TotalNodes int    `json:"total_nodes"`
	AvailNodes int    `json:"avail_nodes"`
	IdleNodes  int    `json:"idle_nodes"`
	DownNodes  int    `json:"down_nodes"`
}

// ClusterStats represents overall cluster statistics
type ClusterStats struct {
	TotalNodes    int `json:"total_nodes"`
	AvailNodes    int `json:"avail_nodes"`
	TotalCores    int `json:"total_cores"`
	UsedCores     int `json:"used_cores"`
	AvailCores    int `json:"avail_cores"`
	TotalMemoryGB int `json:"total_memory_gb"`
	UsedMemoryGB  int `json:"used_memory_gb"`
	AvailMemoryGB int `json:"avail_memory_gb"`
}
