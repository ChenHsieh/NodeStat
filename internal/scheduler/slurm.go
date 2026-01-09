package scheduler

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"nodestat/internal/models"
)

// SlurmScheduler implements the Scheduler interface for SLURM
type SlurmScheduler struct{}

// GetSystemType returns the scheduler type
func (s *SlurmScheduler) GetSystemType() string {
	return "slurm"
}

// GetNodes retrieves all nodes for a given partition
func (s *SlurmScheduler) GetNodes(partition string) ([]models.Node, error) {
	cmd := exec.Command("scontrol", "show", "nodes")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run scontrol: %w", err)
	}

	var nodes []models.Node
	var nodeInfo string

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "NodeName=") {
			// Process previous node if exists
			if nodeInfo != "" {
				node, err := s.parseNodeInfo(nodeInfo)
				if err == nil && s.nodeInPartition(node, partition) {
					nodes = append(nodes, node)
				}
				nodeInfo = ""
			}
		}
		nodeInfo += line + " "
	}

	// Process final node
	if nodeInfo != "" {
		node, err := s.parseNodeInfo(nodeInfo)
		if err == nil && s.nodeInPartition(node, partition) {
			nodes = append(nodes, node)
		}
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes found in partition: %s", partition)
	}

	return nodes, nil
}

// parseNodeInfo parses SLURM node info string
func (s *SlurmScheduler) parseNodeInfo(nodeInfo string) (models.Node, error) {
	node := models.Node{}

	fields := strings.Fields(nodeInfo)
	for _, field := range fields {
		parts := strings.Split(field, "=")
		if len(parts) != 2 {
			continue
		}

		key, value := parts[0], parts[1]
		switch key {
		case "NodeName":
			node.ID = value
		case "State":
			node.State = s.parseNodeState(value)
		case "CPUAlloc":
			if val, err := strconv.Atoi(value); err == nil {
				node.UsedCores = val
			}
		case "CPUTot":
			if val, err := strconv.Atoi(value); err == nil {
				node.TotalCores = val
			}
		case "AllocMem":
			if val, err := strconv.Atoi(value); err == nil {
				node.UsedMemMB = val
			}
		case "RealMemory":
			if val, err := strconv.Atoi(value); err == nil {
				node.TotalMemMB = val
			}
		case "Partitions":
			node.Partitions = strings.Split(value, ",")
		}
	}

	return node, nil
}

// parseNodeState converts SLURM state to our NodeState
func (s *SlurmScheduler) parseNodeState(state string) models.NodeState {
	// Handle states like "IDLE+CLOUD" or "ALLOCATED+CLOUD"
	baseState := strings.Split(state, "+")[0]

	switch strings.ToLower(baseState) {
	case "idle":
		return models.StateIdle
	case "allocated", "mixed":
		return models.StateRunning
	case "down":
		return models.StateDown
	case "offline":
		return models.StateOffline
	case "drained", "drain":
		return models.StateDrained
	default:
		return models.StateBusy
	}
}

// nodeInPartition checks if a node belongs to the specified partition
func (s *SlurmScheduler) nodeInPartition(node models.Node, partition string) bool {
	for _, p := range node.Partitions {
		if p == partition {
			return true
		}
	}
	return false
}

// GetJobs retrieves running jobs for a given partition
func (s *SlurmScheduler) GetJobs(partition string) ([]models.Job, error) {
	cmd := exec.Command("sacct", "-a", "--format",
		"partition,NodeList,JobID,User,jobname,State,ReqNodes,ReqCPUs,ReqMem,Timelimit,Elapsed,CPUTime", "-p")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run sacct: %w", err)
	}

	var jobs []models.Job
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	// Skip header line
	if scanner.Scan() {
		// header line
	}

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, ".extern") {
			continue // Skip extern jobs
		}

		fields := strings.Split(line, "|")
		if len(fields) < 12 {
			continue
		}

		if fields[0] == partition {
			job, err := s.parseJobInfo(fields)
			if err == nil && job.State == models.JobRunning {
				jobs = append(jobs, job)
			}
		}
	}

	return jobs, nil
}

// parseJobInfo parses SLURM job info from sacct output
func (s *SlurmScheduler) parseJobInfo(fields []string) (models.Job, error) {
	job := models.Job{}

	if len(fields) < 12 {
		return job, fmt.Errorf("insufficient fields")
	}

	job.Partition = fields[0]
	job.NodeList = strings.Split(fields[1], ",")
	job.ID = fields[2]
	job.User = fields[3]
	job.Name = fields[4]
	job.State = models.JobState(fields[5][:1]) // Just first character

	// Parse numeric fields
	if val, err := strconv.Atoi(fields[6]); err == nil {
		job.ReqNodes = val
	}
	if val, err := strconv.Atoi(fields[7]); err == nil {
		job.ReqCPUs = val
	}

	// Parse memory (handle different formats like "1000Mc", "1Gn")
	memStr := fields[8]
	memStr = strings.ReplaceAll(memStr, "Mc", "")
	memStr = strings.ReplaceAll(memStr, "Mn", "")
	memStr = strings.ReplaceAll(memStr, "n", "")
	memStr = strings.ReplaceAll(memStr, "c", "")
	memStr = strings.ReplaceAll(memStr, "G", "000")
	if memVal, err := strconv.ParseFloat(memStr, 64); err == nil {
		job.ReqMemMB = int(memVal)
	}

	// Parse time durations
	if duration, err := s.parseTimeString(fields[9]); err == nil {
		job.TimeLimit = duration
	}
	if duration, err := s.parseTimeString(fields[10]); err == nil {
		job.Elapsed = duration
	}
	if duration, err := s.parseTimeString(fields[11]); err == nil {
		job.CPUTime = duration
	}

	return job, nil
}

// parseTimeString parses time strings like "01:30:45" or "2-12:30:45"
func (s *SlurmScheduler) parseTimeString(timeStr string) (time.Duration, error) {
	if timeStr == "" {
		return 0, nil
	}

	var days, hours, minutes, seconds int

	// Handle format with days: "2-12:30:45"
	if strings.Contains(timeStr, "-") {
		parts := strings.Split(timeStr, "-")
		if len(parts) == 2 {
			days, _ = strconv.Atoi(parts[0])
			timeStr = parts[1]
		}
	}

	// Parse HH:MM:SS
	timeParts := strings.Split(timeStr, ":")
	if len(timeParts) == 3 {
		hours, _ = strconv.Atoi(timeParts[0])
		minutes, _ = strconv.Atoi(timeParts[1])
		seconds, _ = strconv.Atoi(timeParts[2])
	}

	totalSeconds := days*24*3600 + hours*3600 + minutes*60 + seconds
	return time.Duration(totalSeconds) * time.Second, nil
}

// GetPartitions retrieves available partitions
func (s *SlurmScheduler) GetPartitions() ([]string, error) {
	cmd := exec.Command("sinfo", "-h", "-o", "%P")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run sinfo: %w", err)
	}

	var partitions []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		partition := strings.TrimSpace(scanner.Text())
		partition = strings.TrimSuffix(partition, "*") // Remove default marker
		if partition != "" {
			partitions = append(partitions, partition)
		}
	}

	return partitions, nil
}

// GetUserJobs retrieves jobs for a specific user
func (s *SlurmScheduler) GetUserJobs(user string) ([]models.Job, error) {
	cmd := exec.Command("sacct", "-u", user, "--format",
		"partition,NodeList,JobID,User,jobname,State,ReqNodes,ReqCPUs,ReqMem,Timelimit,Elapsed,CPUTime", "-p")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run sacct for user %s: %w", user, err)
	}

	var jobs []models.Job
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	// Skip header
	if scanner.Scan() {
	}

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, ".extern") {
			continue
		}

		fields := strings.Split(line, "|")
		if len(fields) >= 12 {
			job, err := s.parseJobInfo(fields)
			if err == nil {
				jobs = append(jobs, job)
			}
		}
	}

	return jobs, nil
}
