mod slurm;
mod torque;
mod mock_scheduler;

pub use slurm::SlurmScheduler;
pub use torque::TorqueScheduler;
pub use mock_scheduler::MockScheduler;

use crate::models::{Node, Job};
use async_trait::async_trait;
use anyhow::Result;

#[derive(Debug, Clone)]
pub enum SchedulerType {
    Slurm,
    Torque,
    Mock,
}

#[async_trait]
pub trait Scheduler: Send + Sync {
    async fn get_nodes(&self, partition: &str) -> Result<Vec<Node>>;
    async fn get_jobs(&self, partition: &str) -> Result<Vec<Job>>;
    async fn get_user_jobs(&self, user: &str) -> Result<Vec<Job>>;
}

pub fn create_scheduler(scheduler_type: SchedulerType) -> Box<dyn Scheduler> {
    match scheduler_type {
        SchedulerType::Slurm => Box::new(SlurmScheduler::new()),
        SchedulerType::Torque => Box::new(TorqueScheduler::new()),
        SchedulerType::Mock => Box::new(MockScheduler::new()),
    }
}