use crate::models::*;
use crate::schedulers::Scheduler;
use async_trait::async_trait;
use anyhow::{Result, anyhow};
use chrono::{Utc, Duration};
use rand::Rng;

pub struct MockScheduler;

impl MockScheduler {
    pub fn new() -> Self {
        Self
    }
}

#[async_trait]
impl Scheduler for MockScheduler {
    async fn get_nodes(&self, partition: &str) -> Result<Vec<Node>> {
        let mut rng = rand::thread_rng();
        
        let (node_count, node_prefix) = match partition {
            "batch" => (25, "batch"),
            "highmem_q" => (8, "highmem"), 
            "gpu_q" => (6, "gpu"),
            _ => return Err(anyhow!("Unknown partition: {}", partition)),
        };
        
        let mut nodes = Vec::new();
        
        for i in 0..node_count {
            let mut node = Node {
                id: format!("{}{:03}", node_prefix, i + 1),
                state: NodeState::Idle,
                total_cores: 0,
                used_cores: 0,
                total_mem_mb: 0,
                used_mem_mb: 0,
                partitions: vec![partition.to_string()],
                jobs: Vec::new(),
            };
            
            // Set specs based on partition
            match partition {
                "batch" => {
                    node.total_cores = 32 + rng.gen_range(0..32);
                    node.total_mem_mb = (128 + rng.gen_range(0..256)) * 1000;
                },
                "highmem_q" => {
                    node.total_cores = 48 + rng.gen_range(0..16);
                    node.total_mem_mb = (512 + rng.gen_range(0..1024)) * 1000;
                },
                "gpu_q" => {
                    node.total_cores = 40 + rng.gen_range(0..20);
                    node.total_mem_mb = (256 + rng.gen_range(0..256)) * 1000;
                },
                _ => {}
            }
            
            // Random states
            let states = [
                NodeState::Idle,
                NodeState::Running,
                NodeState::Running,
                NodeState::Down,
                NodeState::Busy,
            ];
            node.state = states[rng.gen_range(0..states.len())].clone();
            
            // Set usage based on state
            match node.state {
                NodeState::Idle => {
                    node.used_cores = 0;
                    node.used_mem_mb = rng.gen_range(0..node.total_mem_mb / 10);
                },
                NodeState::Running => {
                    node.used_cores = rng.gen_range(0..node.total_cores);
                    node.used_mem_mb = rng.gen_range(0..node.total_mem_mb);
                },
                NodeState::Busy => {
                    node.used_cores = node.total_cores;
                    node.used_mem_mb = node.total_mem_mb - rng.gen_range(0..node.total_mem_mb / 4);
                },
                _ => {
                    node.used_cores = 0;
                    node.used_mem_mb = 0;
                }
            }
            
            // Generate job IDs for running nodes
            if node.state == NodeState::Running && node.used_cores > 0 {
                let job_count = 1 + rng.gen_range(0..3);
                for _ in 0..job_count {
                    node.jobs.push(format!("{}", 100000 + rng.gen_range(0..999999)));
                }
            }
            
            nodes.push(node);
        }
        
        Ok(nodes)
    }

    async fn get_jobs(&self, partition: &str) -> Result<Vec<Job>> {
        let mut rng = rand::thread_rng();
        let job_count = 10 + rng.gen_range(0..20);
        let mut jobs = Vec::new();
        
        let users = ["alice", "bob", "carol", "dave", "eve", "frank", "grace", "henry"];
        
        for i in 0..job_count {
            let job = Job {
                id: format!("{}", 100000 + rng.gen_range(0..999999)),
                user: users[rng.gen_range(0..users.len())].to_string(),
                name: format!("job_{}", i + 1),
                state: JobState::Running,
                partition: partition.to_string(),
                req_nodes: 1 + rng.gen_range(0..4),
                req_cpus: 8 + rng.gen_range(0..32),
                req_mem_mb: (16 + rng.gen_range(0..128)) * 1000,
                elapsed: Duration::seconds(rng.gen_range(0..86400)),
                time_limit: Duration::hours(24),
                cpu_time: Duration::seconds(rng.gen_range(0..86400)),
                submit_time: Utc::now(),
                node_list: vec![format!("{}{:03}", partition, rng.gen_range(1..21))],
            };
            
            jobs.push(job);
        }
        
        Ok(jobs)
    }

    async fn get_user_jobs(&self, user: &str) -> Result<Vec<Job>> {
        let mut rng = rand::thread_rng();
        let job_count = rng.gen_range(0..4);
        let mut jobs = Vec::new();
        
        for i in 0..job_count {
            let job = Job {
                id: format!("{}", 200000 + rng.gen_range(0..999999)),
                user: user.to_string(),
                name: format!("my_job_{}", i + 1),
                state: JobState::Running,
                partition: "batch".to_string(),
                req_nodes: 1,
                req_cpus: 4 + rng.gen_range(0..16),
                req_mem_mb: (8 + rng.gen_range(0..64)) * 1000,
                elapsed: Duration::seconds(rng.gen_range(0..43200)),
                time_limit: Duration::hours(12),
                cpu_time: Duration::seconds(rng.gen_range(0..43200)),
                submit_time: Utc::now(),
                node_list: vec![format!("batch{:03}", rng.gen_range(1..11))],
            };
            
            jobs.push(job);
        }
        
        Ok(jobs)
    }
}