use crate::models::*;
use crate::schedulers::Scheduler;
use async_trait::async_trait;
use anyhow::{Result, Context};
use std::process::Command;
use std::env;
use chrono::{Duration, Utc};

pub struct TorqueScheduler;

impl TorqueScheduler {
    pub fn new() -> Self {
        Self
    }

    fn parse_node_state(state_str: &str) -> NodeState {
        match state_str.to_uppercase().as_str() {
            "IDLE" => NodeState::Idle,
            "BUSY" => NodeState::Busy,
            "DOWN" | "DOWN*" => NodeState::Down,
            "OFFLINE" => NodeState::Offline,
            "DRAINED" => NodeState::Drained,
            _ => NodeState::Offline,
        }
    }

    fn parse_job_state(state_str: &str) -> JobState {
        match state_str {
            "R" => JobState::Running,
            "Q" | "H" => JobState::Pending,
            "C" => JobState::Completed,
            "E" => JobState::Failed,
            _ => JobState::Failed,
        }
    }

    fn parse_duration(time_str: &str) -> Duration {
        // Parse time in format HH:MM:SS
        let parts: Vec<&str> = time_str.split(':').collect();
        if parts.len() >= 3 {
            let hours: i64 = parts[0].parse().unwrap_or(0);
            let minutes: i64 = parts[1].parse().unwrap_or(0);
            let seconds: i64 = parts[2].parse().unwrap_or(0);
            Duration::seconds(hours * 3600 + minutes * 60 + seconds)
        } else {
            Duration::seconds(0)
        }
    }

    fn parse_node_info(node_info: &str, partition: &str) -> Option<Node> {
        // Clean up excessive spaces (from Python: for x in range(30,1, -1): node_info = node_info.replace(" "*x, " "))
        let mut cleaned_info = node_info.to_string();
        for x in (2..=30).rev() {
            cleaned_info = cleaned_info.replace(&" ".repeat(x), " ");
        }

        let fields: Vec<&str> = cleaned_info.split_whitespace().collect();
        if fields.len() < 4 {
            return None;
        }

        // Check if this node belongs to the partition
        if !node_info.contains(&format!("[{}]", partition)) || node_info.contains(&format!("[{}][", partition)) {
            return None;
        }

        let id = fields[0].to_string();
        let state = Self::parse_node_state(fields[1]);
        
        // Parse CPU info (format: "available:total")
        let cpu_info: Vec<&str> = fields[2].split(':').collect();
        let total_cores = cpu_info.get(1)?.parse::<u32>().ok()?;
        let available_cores = cpu_info.get(0)?.parse::<u32>().ok()?;
        let used_cores = total_cores.saturating_sub(available_cores);

        // Parse memory info (format: "available:total" in MB)
        let mem_info: Vec<&str> = fields[3].split(':').collect();
        let total_mem_mb = mem_info.get(1)?.parse::<u32>().ok()?;
        let available_mem_mb = mem_info.get(0)?.parse::<u32>().ok()?;
        let used_mem_mb = total_mem_mb.saturating_sub(available_mem_mb);

        Some(Node {
            id,
            state,
            total_cores,
            used_cores,
            total_mem_mb,
            used_mem_mb,
            jobs: Vec::new(),
            partitions: vec![partition.to_string()],
        })
    }

    fn parse_job_info(job_text: &str) -> Option<Job> {
        let mut job_id = String::new();
        let mut name = String::new();
        let mut owner = String::new();
        let mut cpu_time = "00:00:00".to_string();
        let mut wall_time = "00:00:00".to_string();
        let mut req_time = "00:00:00".to_string();
        let mut state = String::new();
        let mut req_mem = "1gb".to_string();
        let mut req_cpu = "1".to_string();
        let mut node_id = "?".to_string();

        for line in job_text.lines() {
            if line.contains("Job Id:") {
                if let Some(pos) = line.find(':') {
                    job_id = line[pos + 2..].trim().to_string();
                }
            } else if line.contains("Job_Name =") {
                if let Some(pos) = line.find('=') {
                    name = line[pos + 2..].trim().to_string();
                }
            } else if line.contains("Job_Owner =") {
                if let Some(pos) = line.find('=') {
                    let line_part = &line[pos + 2..];
                    if let Some(at_pos) = line_part.find('@') {
                        owner = line_part[..at_pos].to_string();
                    }
                }
            } else if line.contains("resources_used.cput =") {
                if let Some(pos) = line.find('=') {
                    cpu_time = line[pos + 2..].trim().to_string();
                }
            } else if line.contains("resources_used.walltime =") {
                if let Some(pos) = line.find('=') {
                    wall_time = line[pos + 2..].trim().to_string();
                }
            } else if line.contains("Resource_List.walltime =") {
                if let Some(pos) = line.find('=') {
                    req_time = line[pos + 2..].trim().to_string();
                }
            } else if line.contains("job_state =") {
                if let Some(pos) = line.find('=') {
                    state = line[pos + 2..].trim().to_string();
                }
            } else if line.contains("Resource_List.mem =") {
                if let Some(pos) = line.find('=') {
                    req_mem = line[pos + 2..].trim().to_string();
                }
            } else if line.contains("Resource_List.nodes =") {
                if let Some(pos) = line.find('=') {
                    let line_part = &line[pos + 2..];
                    if let Some(colon_pos) = line_part.find(':') {
                        if let Some(eq_pos) = line_part.find('=') {
                            let cpu_part = &line_part[eq_pos + 1..];
                            if let Some(colon_pos2) = cpu_part.find(':') {
                                req_cpu = cpu_part[..colon_pos2].to_string();
                            } else {
                                req_cpu = cpu_part.trim().to_string();
                            }
                        }
                    }
                }
            } else if line.contains("exec_host =") {
                if let Some(pos) = line.find('=') {
                    let line_part = &line[pos + 2..];
                    if let Some(slash_pos) = line_part.find('/') {
                        node_id = line_part[..slash_pos].to_string();
                    }
                }
            }
        }

        // Only return running jobs
        if state == "R" {
            // Parse memory (remove 'gb' and convert to number, keep in MB)
            let memory_str = req_mem.to_lowercase().replace("gb", "").replace("mb", "");
            let memory_mb = if req_mem.to_lowercase().contains("gb") {
                memory_str.parse::<u32>().unwrap_or(1) * 1000
            } else {
                memory_str.parse::<u32>().unwrap_or(1000)
            };
            
            Some(Job {
                id: job_id,
                user: owner,
                name,
                state: Self::parse_job_state(&state),
                node_list: vec![node_id],
                partition: "default".to_string(), // Torque doesn't use partitions like SLURM
                req_nodes: 1,
                req_cpus: req_cpu.parse().unwrap_or(1),
                req_mem_mb: memory_mb,
                time_limit: Self::parse_duration(&req_time),
                elapsed: Self::parse_duration(&wall_time),
                cpu_time: Self::parse_duration(&cpu_time),
                submit_time: Utc::now(), // We don't have submit time in this format
            })
        } else {
            None
        }
    }
}

#[async_trait]
impl Scheduler for TorqueScheduler {
    async fn get_nodes(&self, partition: &str) -> Result<Vec<Node>> {
        let output = Command::new("mdiag")
            .args(["-n", "-v"])
            .output()
            .context("Failed to execute mdiag command")?;

        if !output.status.success() {
            return Err(anyhow::anyhow!(
                "mdiag command failed: {}", 
                String::from_utf8_lossy(&output.stderr)
            ));
        }

        let output_str = String::from_utf8_lossy(&output.stdout);
        let mut nodes = Vec::new();

        for line in output_str.lines() {
            if let Some(node) = Self::parse_node_info(line, partition) {
                nodes.push(node);
            }
        }

        if nodes.is_empty() {
            return Err(anyhow::anyhow!("No nodes found in partition: {}", partition));
        }

        Ok(nodes)
    }

    async fn get_jobs(&self, partition: &str) -> Result<Vec<Job>> {
        let output = Command::new("qstat")
            .args(["-f", partition])
            .output()
            .context("Failed to execute qstat command")?;

        if !output.status.success() {
            return Err(anyhow::anyhow!(
                "qstat command failed: {}", 
                String::from_utf8_lossy(&output.stderr)
            ));
        }

        let output_str = String::from_utf8_lossy(&output.stdout);
        let mut jobs = Vec::new();
        let mut current_job_info = String::new();

        for line in output_str.lines() {
            if line.contains("Job Id:") {
                if !current_job_info.is_empty() {
                    if let Some(job) = Self::parse_job_info(&current_job_info) {
                        jobs.push(job);
                    }
                }
                current_job_info = line.to_string();
            } else {
                current_job_info.push('\n');
                current_job_info.push_str(line);
            }
        }

        // Don't forget the last job
        if !current_job_info.is_empty() {
            if let Some(job) = Self::parse_job_info(&current_job_info) {
                jobs.push(job);
            }
        }

        Ok(jobs)
    }

    async fn get_user_jobs(&self, user: &str) -> Result<Vec<Job>> {
        let current_user = env::var("USER").unwrap_or_else(|_| user.to_string());
        
        let output = Command::new("qstat")
            .args(["-u", &current_user])
            .output()
            .context("Failed to execute qstat command")?;

        if !output.status.success() {
            return Err(anyhow::anyhow!(
                "qstat command failed: {}", 
                String::from_utf8_lossy(&output.stderr)
            ));
        }

        let output_str = String::from_utf8_lossy(&output.stdout);
        let mut jobs = Vec::new();

        for line in output_str.lines().skip(2) { // Skip header lines
            let fields: Vec<&str> = line.split_whitespace().collect();
            if fields.len() >= 6 && fields[4] == "R" { // Running jobs
                let job = Job {
                    id: fields[0].to_string(),
                    user: fields[1].to_string(),
                    name: fields[2].to_string(),
                    state: JobState::Running,
                    node_list: vec![fields.get(7).unwrap_or(&"?").to_string()],
                    partition: "default".to_string(),
                    req_nodes: 1,
                    req_cpus: 1, // qstat doesn't show cores directly
                    req_mem_mb: 1000, // qstat doesn't show memory directly
                    time_limit: Self::parse_duration(fields.get(8).unwrap_or(&"00:00:00")),
                    elapsed: Self::parse_duration(fields.get(10).unwrap_or(&"00:00:00")),
                    cpu_time: Duration::seconds(0),
                    submit_time: Utc::now(),
                };
                jobs.push(job);
            }
        }

        Ok(jobs)
    }
}