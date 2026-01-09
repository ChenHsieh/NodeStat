use chrono::{DateTime, Duration, Utc};
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub enum NodeState {
    Idle,
    Running,
    Down,
    Offline,
    Busy,
    Drained,
}

impl std::fmt::Display for NodeState {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            NodeState::Idle => write!(f, "Idle"),
            NodeState::Running => write!(f, "Running"),
            NodeState::Down => write!(f, "Down"),
            NodeState::Offline => write!(f, "Offline"),
            NodeState::Busy => write!(f, "Busy"),
            NodeState::Drained => write!(f, "Drained"),
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Node {
    pub id: String,
    pub state: NodeState,
    pub total_cores: u32,
    pub used_cores: u32,
    pub total_mem_mb: u32,
    pub used_mem_mb: u32,
    pub partitions: Vec<String>,
    pub jobs: Vec<String>,
}

impl Node {
    pub fn available_cores(&self) -> u32 {
        self.total_cores.saturating_sub(self.used_cores)
    }

    pub fn available_mem_gb(&self) -> u32 {
        (self.total_mem_mb.saturating_sub(self.used_mem_mb)) / 1000
    }

    pub fn total_mem_gb(&self) -> u32 {
        self.total_mem_mb / 1000
    }

    pub fn used_mem_gb(&self) -> u32 {
        self.used_mem_mb / 1000
    }

    pub fn is_available(&self) -> bool {
        matches!(self.state, NodeState::Idle | NodeState::Running)
            && self.available_cores() > 0
            && self.available_mem_gb() > 0
    }
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub enum JobState {
    Running,
    Pending,
    Completed,
    Cancelled,
    Failed,
}

impl std::fmt::Display for JobState {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            JobState::Running => write!(f, "R"),
            JobState::Pending => write!(f, "PD"),
            JobState::Completed => write!(f, "C"),
            JobState::Cancelled => write!(f, "CA"),
            JobState::Failed => write!(f, "F"),
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Job {
    pub id: String,
    pub user: String,
    pub name: String,
    pub state: JobState,
    pub node_list: Vec<String>,
    pub partition: String,
    pub req_nodes: u32,
    pub req_cpus: u32,
    pub req_mem_mb: u32,
    pub time_limit: Duration,
    pub elapsed: Duration,
    pub cpu_time: Duration,
    pub submit_time: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClusterStats {
    pub total_nodes: u32,
    pub avail_nodes: u32,
    pub total_cores: u32,
    pub used_cores: u32,
    pub avail_cores: u32,
    pub total_memory_gb: u32,
    pub used_memory_gb: u32,
    pub avail_memory_gb: u32,
}