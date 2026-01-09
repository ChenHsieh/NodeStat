package scheduler

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"nodestat/internal/models"
)

// TorqueScheduler implements the Scheduler interface for Torque/PBS
type TorqueScheduler struct{}

// GetSystemType returns the scheduler type
func (t *TorqueScheduler) GetSystemType() string {
	return "torque"
}

// GetNodes retrieves all nodes for a given partition
func (t *TorqueScheduler) GetNodes(partition string) ([]models.Node, error) {
	cmd := exec.Command("mdiag", "-n", "-v")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run mdiag: %w", err)
	}

	var nodes []models.Node
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "["+partition+"]") && !strings.Contains(line, "["+partition+"][") {
			node, err := t.parseNodeInfo(line)
			if err == nil {
				nodes = append(nodes, node)
			}
		}
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes found in partition: %s", partition)
	}

	return nodes, nil
}

// parseNodeInfo parses Torque node info string
func (t *TorqueScheduler) parseNodeInfo(line string) (models.Node, error) {
	node := models.Node{}

	// Clean up multiple spaces
	for i := 30; i > 1; i-- {
		line = strings.ReplaceAll(line, strings.Repeat(" ", i), " ")
	}

	fields := strings.Split(line, " ")
	if len(fields) < 4 {
		return node, fmt.Errorf("insufficient fields in node info")
	}

	node.ID = fields[0]
	node.State = t.parseNodeState(fields[1])

	// Parse CPU info "available:total"
	if cpuParts := strings.Split(fields[2], ":"); len(cpuParts) == 2 {
		if total, err := strconv.Atoi(cpuParts[1]); err == nil {
			node.TotalCores = total
		}
		if avail, err := strconv.Atoi(cpuParts[0]); err == nil {
			node.UsedCores = node.TotalCores - avail
		}
	}

	// Parse memory info "available:total" (in KB)
	if memParts := strings.Split(fields[3], ":"); len(memParts) == 2 {
		if total, err := strconv.Atoi(memParts[1]); err == nil {
			node.TotalMemMB = total / 1000 // Convert KB to MB
		}
		if avail, err := strconv.Atoi(memParts[0]); err == nil {
			node.UsedMemMB = node.TotalMemMB - (avail / 1000)
		}
	}

	return node, nil
}

// parseNodeState converts Torque state to our NodeState
func (t *TorqueScheduler) parseNodeState(state string) models.NodeState {
	switch strings.ToLower(state) {
	case "free":
		return models.StateIdle
	case "busy":
		return models.StateBusy
	case "down":
		return models.StateDown
	case "offline":
		return models.StateOffline
	default:
		return models.StateBusy
	}
}

// GetJobs retrieves running jobs for a given partition
func (t *TorqueScheduler) GetJobs(partition string) ([]models.Job, error) {
	cmd := exec.Command("qstat", "-f", partition)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run qstat: %w", err)
	}

	var jobs []models.Job
	var jobInfo string

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Job Id:") {
			if jobInfo != "" {
				job, err := t.parseJobInfo(jobInfo)
				if err == nil && job.State == models.JobRunning {
					jobs = append(jobs, job)
				}
			}
			jobInfo = line
		} else {
			jobInfo += line
		}
	}

	// Process final job
	if jobInfo != "" {
		job, err := t.parseJobInfo(jobInfo)
		if err == nil && job.State == models.JobRunning {
			jobs = append(jobs, job)
		}
	}

	return jobs, nil
}

// parseJobInfo parses Torque job info
func (t *TorqueScheduler) parseJobInfo(jobInfo string) (models.Job, error) {
	job := models.Job{}
	lines := strings.Split(jobInfo, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Job Id:") {
			job.ID = strings.TrimSpace(line[strings.Index(line, ":")+1:])
		} else if strings.Contains(line, "Job_Name =") {
			job.Name = strings.TrimSpace(line[strings.Index(line, "=")+1:])
		} else if strings.Contains(line, "Job_Owner =") {
			owner := strings.TrimSpace(line[strings.Index(line, "=")+1:])
			if atIndex := strings.Index(owner, "@"); atIndex != -1 {
				job.User = owner[:atIndex]
			}
		} else if strings.Contains(line, "job_state =") {
			state := strings.TrimSpace(line[strings.Index(line, "=")+1:])
			job.State = models.JobState(state)
		} else if strings.Contains(line, "exec_host =") {
			hostLine := strings.TrimSpace(line[strings.Index(line, "=")+1:])
			if slashIndex := strings.Index(hostLine, "/"); slashIndex != -1 {
				job.NodeList = []string{hostLine[:slashIndex]}
			}
		}
	}

	return job, nil
}

// GetPartitions retrieves available partitions (queues in Torque)
func (t *TorqueScheduler) GetPartitions() ([]string, error) {
	// For Torque, we'll return common queue names
	// In a real implementation, you might parse qstat -Q or similar
	return []string{"batch", "highmem_q", "gpu_q", "s_interq"}, nil
}

// GetUserJobs retrieves jobs for a specific user
func (t *TorqueScheduler) GetUserJobs(user string) ([]models.Job, error) {
	cmd := exec.Command("qstat", "-u", user)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run qstat for user %s: %w", user, err)
	}

	var jobs []models.Job
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		// Parse qstat output - this would need more detailed implementation
		// based on the actual qstat format for your system
		_ = line // placeholder
	}

	return jobs, nil
}
