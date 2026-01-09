use crate::models::*;
use crate::schedulers::Scheduler;
use async_trait::async_trait;
use anyhow::{Result, Context};
use std::process::Command;
use std::env;
use chrono::{Duration, Utc};

pub struct SlurmScheduler;

impl SlurmScheduler {
    pub fn new() -> Self {
        Self
    }

    fn parse_node_state(state_str: &str) -> NodeState {
        match state_str.to_uppercase().as_str() {
            "IDLE" => NodeState::Idle,
            "MIXED" | "ALLOC" => NodeState::Running,
            "DOWN" | "DOWN*" => NodeState::Down,
            "DRAINED" => NodeState::Drained,
            _ => NodeState::Offline,
        }
    }

    fn parse_job_state(state_str: &str) -> JobState {
        match state_str.chars().next().unwrap_or('?') {
            'R' => JobState::Running,
            'P' => JobState::Pending,
            'C' => JobState::Completed,
            'F' => JobState::Failed,
            'C' if state_str.starts_with("CA") => JobState::Cancelled,
            _ => JobState::Failed,
        }
    }

    fn parse_duration(time_str: &str) -> Duration {
        // Parse time in format HH:MM:SS or days-HH:MM:SS
        let parts: Vec<&str> = time_str.split(':').collect();
        if parts.len() >= 3 {
            let hours: i64 = parts[parts.len()-3].parse().unwrap_or(0);
            let minutes: i64 = parts[parts.len()-2].parse().unwrap_or(0);
            let seconds: i64 = parts[parts.len()-1].parse().unwrap_or(0);
            Duration::seconds(hours * 3600 + minutes * 60 + seconds)
        } else {
            Duration::seconds(0)
        }
    }

    fn parse_node_info(node_info: &str, partition: &str) -> Option<Node> {
        let mut node = Node {
            id: String::new(),
            state: NodeState::Offline,
            total_cores: 0,
            used_cores: 0,
            total_mem_mb: 0,
            used_mem_mb: 0,
            jobs: Vec::new(),
            partitions: Vec::new(),
        };

        let mut has_partition = false;
        
        for info in node_info.split_whitespace() {
            if let Some((key, value)) = info.split_once('=') {
                match key {
                    "NodeName" => node.id = value.to_string(),
                    "State" => node.state = Self::parse_node_state(value),
                    "CPUAlloc" => {
                        if let Ok(val) = value.parse::<u32>() {
                            node.used_cores = val;
                        }
                    },
                    "CPUTot" => {
                        if let Ok(val) = value.parse::<u32>() {
                            node.total_cores = val;
                        }
                    },
                    "AllocMem" => {
                        if let Ok(val) = value.parse::<u32>() {
                            node.used_mem_mb = val;
                        }
                    },
                    "RealMemory" => {
                        if let Ok(val) = value.parse::<u32>() {
                            node.total_mem_mb = val;
                        }
                    },
                    "Partitions" => {
                        node.partitions = value.split(',').map(|s| s.to_string()).collect();
                        if node.partitions.iter().any(|p| p == partition) {
                            has_partition = true;
                        }
                    },
                    _ => {}
                }
            }
        }

        if has_partition && !node.id.is_empty() {
            Some(node)
        } else {
            None
        }
    }

    fn parse_job_line(line: &str, partition: &str) -> Option<Job> {
        let fields: Vec<&str> = line.split('|').collect();
        if fields.len() < 12 {
            return None;
        }

        // Skip .extern jobs and check partition
        if fields[2].contains(".extern") || !fields[0].contains(partition) {
            return None;
        }

        // Only include running jobs
        if !fields[5].starts_with('R') {
            return None;
        }

        let mut req_mem = fields[8].to_string();
        // Clean up memory format
        req_mem = req_mem.replace("Mc", "").replace("Mn", "").replace("n", "").replace("c", "");
        if req_mem.contains('G') {
            req_mem = req_mem.replace('G', "000");
        }
        
        let memory_mb = req_mem.parse::<f64>().unwrap_or(0.0) as u32;

        Some(Job {
            id: fields[2].to_string(),
            user: fields[3].to_string(),
            name: fields[4].to_string(),
            state: Self::parse_job_state(fields[5]),
            node_list: fields[1].split(',').map(|s| s.to_string()).collect(),
            partition: fields[0].to_string(),
            req_nodes: fields[6].parse().unwrap_or(1),
            req_cpus: fields[7].parse().unwrap_or(0),
            req_mem_mb: memory_mb,
            time_limit: Self::parse_duration(fields[9]),
            elapsed: Self::parse_duration(fields[10]),
            cpu_time: Self::parse_duration(fields[11]),
            submit_time: Utc::now(), // We don't have submit time in this format
        })
    }
}

#[async_trait]
impl Scheduler for SlurmScheduler {
    async fn get_nodes(&self, partition: &str) -> Result<Vec<Node>> {
        let output = Command::new("scontrol")
            .args(["show", "nodes"])
            .output()
            .context("Failed to execute scontrol command")?;

        if !output.status.success() {
            return Err(anyhow::anyhow!(
                "scontrol command failed: {}", 
                String::from_utf8_lossy(&output.stderr)
            ));
        }

        let output_str = String::from_utf8_lossy(&output.stdout);
        let mut nodes = Vec::new();
        let mut current_node_info = String::new();

        for line in output_str.lines() {
            if line.starts_with("NodeName=") {
                if !current_node_info.is_empty() {
                    if let Some(node) = Self::parse_node_info(&current_node_info, partition) {
                        nodes.push(node);
                    }
                }
                current_node_info = line.to_string();
            } else {
                current_node_info.push(' ');
                current_node_info.push_str(line);
            }
        }

        // Don't forget the last node
        if !current_node_info.is_empty() {
            if let Some(node) = Self::parse_node_info(&current_node_info, partition) {
                nodes.push(node);
            }
        }

        if nodes.is_empty() {
            return Err(anyhow::anyhow!("No nodes found in partition: {}", partition));
        }

        Ok(nodes)
    }

    async fn get_jobs(&self, partition: &str) -> Result<Vec<Job>> {
        let output = Command::new("sacct")
            .args([
                "-a",
                "--format",
                "partition,NodeList,JobID,User,jobname,State,ReqNodes,ReqCPUs,ReqMem,Timelimit,Elapsed,CPUTime",
                "-p"
            ])
            .output()
            .context("Failed to execute sacct command")?;

        if !output.status.success() {
            return Err(anyhow::anyhow!(
                "sacct command failed: {}", 
                String::from_utf8_lossy(&output.stderr)
            ));
        }

        let output_str = String::from_utf8_lossy(&output.stdout);
        let mut jobs = Vec::new();

        for (ln, line) in output_str.lines().enumerate() {
            if ln > 0 { // Skip header
                if let Some(job) = Self::parse_job_line(line, partition) {
                    jobs.push(job);
                }
            }
        }

        Ok(jobs)
    }

    async fn get_user_jobs(&self, user: &str) -> Result<Vec<Job>> {
        let current_user = env::var("USER").unwrap_or_else(|_| user.to_string());
        
        let output = Command::new("sacct")
            .args([
                "-u", &current_user,
                "--format",
                "partition,NodeList,JobID,User,jobname,State,ReqNodes,ReqCPUs,ReqMem,Timelimit,Elapsed,CPUTime",
                "-p"
            ])
            .output()
            .context("Failed to execute sacct command")?;

        if !output.status.success() {
            return Err(anyhow::anyhow!(
                "sacct command failed: {}", 
                String::from_utf8_lossy(&output.stderr)
            ));
        }

        let output_str = String::from_utf8_lossy(&output.stdout);
        let mut jobs = Vec::new();

        for (ln, line) in output_str.lines().enumerate() {
            if ln > 0 { // Skip header
                let fields: Vec<&str> = line.split('|').collect();
                if fields.len() >= 12 && !fields[2].contains(".extern") && fields[5].starts_with('R') {
                    if let Some(job) = Self::parse_job_line(line, "") { // Don't filter by partition for user jobs
                        jobs.push(job);
                    }
                }
            }
        }

        Ok(jobs)
    }
}