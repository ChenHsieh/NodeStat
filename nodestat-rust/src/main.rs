use clap::Parser;

mod models;
mod schedulers;
mod ui;

use schedulers::*;
use ui::App;

#[derive(Parser)]
#[command(name = "nodestat")]
#[command(about = "Modern TUI for cluster monitoring")]
struct Cli {
    /// Partition/queue to display
    #[arg(short = 'q', long = "partition", default_value = "batch")]
    partition: String,

    /// Scheduler system (slurm, torque, mock)
    #[arg(short = 's', long = "scheduler", default_value = "slurm")]
    scheduler: String,

    /// Show version
    #[arg(short = 'v', long = "version")]
    version: bool,
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let cli = Cli::parse();

    if cli.version {
        println!("NodeStat TUI v1.0.0 (Rust)");
        println!("A modern terminal UI for cluster monitoring");
        return Ok(());
    }

    let scheduler_type = match cli.scheduler.as_str() {
        "slurm" => SchedulerType::Slurm,
        "torque" => SchedulerType::Torque,
        "mock" => SchedulerType::Mock,
        _ => {
            eprintln!("Error: Invalid scheduler type '{}'. Use 'slurm', 'torque', or 'mock'", cli.scheduler);
            std::process::exit(1);
        }
    };

    let scheduler = create_scheduler(scheduler_type);
    let mut app = App::new(scheduler, cli.partition).await?;
    
    app.run().await?;

    Ok(())
}