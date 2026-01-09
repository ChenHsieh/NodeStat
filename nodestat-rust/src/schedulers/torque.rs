// Placeholder for Torque implementation
use crate::models::*;
use crate::schedulers::Scheduler;
use async_trait::async_trait;
use anyhow::Result;

pub struct TorqueScheduler;

impl TorqueScheduler {
    pub fn new() -> Self {
        Self
    }
}

#[async_trait]
impl Scheduler for TorqueScheduler {
    async fn get_nodes(&self, _partition: &str) -> Result<Vec<Node>> {
        Ok(vec![])
    }

    async fn get_jobs(&self, _partition: &str) -> Result<Vec<Job>> {
        Ok(vec![])
    }

    async fn get_user_jobs(&self, _user: &str) -> Result<Vec<Job>> {
        Ok(vec![])
    }
}