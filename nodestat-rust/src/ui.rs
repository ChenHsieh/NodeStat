use crate::models::*;
use crate::schedulers::Scheduler;
use std::time::{Duration, Instant};
use crossterm::{
    event::{self, DisableMouseCapture, EnableMouseCapture, Event, KeyCode, MouseEventKind},
    execute,
    terminal::{disable_raw_mode, enable_raw_mode, EnterAlternateScreen, LeaveAlternateScreen},
};
use ratatui::{
    backend::{Backend, CrosstermBackend},
    layout::{Alignment, Constraint, Direction, Layout, Rect},
    style::{Color, Modifier, Style},
    widgets::{
        Block, Borders, Cell, Gauge, Paragraph, Row, Table, TableState,
    },
    Frame, Terminal,
};
use anyhow::Result;
use std::io;

pub struct App {
    scheduler: Box<dyn Scheduler>,
    current_partition: String,
    nodes: Vec<Node>,
    jobs: Vec<Job>,
    user_jobs: Vec<Job>,
    current_user: String,
    stats: ClusterStats,
    table_state: TableState,
    refresh_interval: Duration,
    last_update: Instant,
    should_quit: bool,
    error_message: Option<String>,
}

impl App {
    pub async fn new(scheduler: Box<dyn Scheduler>, partition: String) -> Result<Self> {
        let current_user = std::env::var("USER").unwrap_or_else(|_| "unknown".to_string());
        
        let mut app = App {
            scheduler,
            current_partition: partition,
            nodes: Vec::new(),
            jobs: Vec::new(), 
            user_jobs: Vec::new(),
            current_user,
            stats: ClusterStats {
                total_nodes: 0,
                avail_nodes: 0,
                total_cores: 0,
                used_cores: 0,
                avail_cores: 0,
                total_memory_gb: 0,
                used_memory_gb: 0,
                avail_memory_gb: 0,
            },
            table_state: TableState::default(),
            refresh_interval: Duration::from_secs(30),
            last_update: Instant::now(),
            should_quit: false,
            error_message: None,
        };

        app.fetch_data().await;
        Ok(app)
    }

    pub async fn run(&mut self) -> Result<()> {
        // Setup terminal
        enable_raw_mode()?;
        let mut stdout = io::stdout();
        execute!(stdout, EnterAlternateScreen, EnableMouseCapture)?;
        let backend = CrosstermBackend::new(stdout);
        let mut terminal = Terminal::new(backend)?;

        let result = self.run_app(&mut terminal).await;

        // Restore terminal
        disable_raw_mode()?;
        execute!(
            terminal.backend_mut(),
            LeaveAlternateScreen,
            DisableMouseCapture
        )?;
        terminal.show_cursor()?;

        result
    }

    async fn run_app<B: Backend>(&mut self, terminal: &mut Terminal<B>) -> Result<()> {
        let mut last_refresh = Instant::now();

        loop {
            terminal.draw(|f| self.ui(f))?;

            // Handle input
            if event::poll(Duration::from_millis(100))? {
                match event::read()? {
                    Event::Key(key) => {
                        match key.code {
                            KeyCode::Char('q') => self.should_quit = true,
                            KeyCode::Char('r') | KeyCode::Char(' ') => {
                                self.fetch_data().await;
                            },
                            KeyCode::Char('b') => {
                                self.current_partition = "batch".to_string();
                                self.fetch_data().await;
                            },
                            KeyCode::Char('m') => {
                                self.current_partition = "highmem_q".to_string(); 
                                self.fetch_data().await;
                            },
                            KeyCode::Char('g') => {
                                self.current_partition = "gpu_q".to_string();
                                self.fetch_data().await;
                            },
                            KeyCode::Down | KeyCode::Char('j') => {
                                self.next_node();
                            },
                            KeyCode::Up | KeyCode::Char('k') => {
                                self.previous_node();
                            },
                            _ => {},
                        }
                    },
                    Event::Mouse(mouse) => {
                        match mouse.kind {
                            MouseEventKind::Down(_) => {
                                // Handle mouse clicks for table selection
                                if mouse.row >= 6 && mouse.row < (6 + self.nodes.len() as u16) {
                                    let selected_index = (mouse.row - 6) as usize;
                                    if selected_index < self.nodes.len() {
                                        self.table_state.select(Some(selected_index));
                                    }
                                }
                            },
                            MouseEventKind::ScrollDown => {
                                self.next_node();
                            },
                            MouseEventKind::ScrollUp => {
                                self.previous_node();
                            },
                            _ => {},
                        }
                    },
                    _ => {},
                }
            }

            // Auto refresh
            if last_refresh.elapsed() >= self.refresh_interval {
                self.fetch_data().await;
                last_refresh = Instant::now();
            }

            if self.should_quit {
                break;
            }
        }

        Ok(())
    }

    async fn fetch_data(&mut self) {
        self.error_message = None;
        
        match self.scheduler.get_nodes(&self.current_partition).await {
            Ok(mut nodes) => {
                // Sort nodes: IDLE first, then by available resources
                nodes.sort_by(|a, b| {
                    // Available nodes first
                    if a.is_available() != b.is_available() {
                        return b.is_available().cmp(&a.is_available());
                    }
                    
                    // Among available, sort by power (cores + memory)
                    if a.is_available() && b.is_available() {
                        let a_power = a.available_cores() * 1000 + a.available_mem_gb();
                        let b_power = b.available_cores() * 1000 + b.available_mem_gb();
                        return b_power.cmp(&a_power);
                    }
                    
                    // State ordering for unavailable nodes
                    use std::cmp::Ordering;
                    match (&a.state, &b.state) {
                        (NodeState::Running, _) => Ordering::Less,
                        (_, NodeState::Running) => Ordering::Greater,
                        (NodeState::Busy, _) => Ordering::Less,
                        (_, NodeState::Busy) => Ordering::Greater,
                        _ => Ordering::Equal,
                    }
                });
                
                self.stats = self.calculate_stats(&nodes);
                self.nodes = nodes;
            },
            Err(e) => {
                self.error_message = Some(format!("Failed to get nodes: {}", e));
            }
        }
        
        // Get jobs (don't fail on error)
        if let Ok(jobs) = self.scheduler.get_jobs(&self.current_partition).await {
            self.jobs = jobs;
        }
        
        // Get user jobs (don't fail on error)
        if let Ok(user_jobs) = self.scheduler.get_user_jobs(&self.current_user).await {
            self.user_jobs = user_jobs;
        }
        
        self.last_update = Instant::now();
    }

    fn calculate_stats(&self, nodes: &[Node]) -> ClusterStats {
        let mut stats = ClusterStats {
            total_nodes: nodes.len() as u32,
            avail_nodes: 0,
            total_cores: 0,
            used_cores: 0,
            avail_cores: 0,
            total_memory_gb: 0,
            used_memory_gb: 0,
            avail_memory_gb: 0,
        };
        
        for node in nodes {
            stats.total_cores += node.total_cores;
            stats.used_cores += node.used_cores;
            stats.total_memory_gb += node.total_mem_gb();
            stats.used_memory_gb += node.used_mem_gb();
            
            if node.is_available() {
                stats.avail_nodes += 1;
            }
        }
        
        stats.avail_cores = stats.total_cores.saturating_sub(stats.used_cores);
        stats.avail_memory_gb = stats.total_memory_gb.saturating_sub(stats.used_memory_gb);
        
        stats
    }

    fn next_node(&mut self) {
        let i = match self.table_state.selected() {
            Some(i) => {
                if i >= self.nodes.len() - 1 {
                    0
                } else {
                    i + 1
                }
            }
            None => 0,
        };
        self.table_state.select(Some(i));
    }

    fn previous_node(&mut self) {
        let i = match self.table_state.selected() {
            Some(i) => {
                if i == 0 {
                    self.nodes.len() - 1
                } else {
                    i - 1
                }
            }
            None => 0,
        };
        self.table_state.select(Some(i));
    }

    fn user_has_jobs_on_node(&self, node_id: &str) -> bool {
        self.user_jobs.iter().any(|job| {
            job.state == JobState::Running && job.node_list.iter().any(|n| n == node_id)
        })
    }

    fn ui(&mut self, f: &mut Frame) {
        let chunks = Layout::default()
            .direction(Direction::Vertical)
            .margin(1)
            .constraints([
                Constraint::Length(1), // Title
                Constraint::Length(1), // Spacing
                Constraint::Length(1), // Header
                Constraint::Length(5), // Stats
                Constraint::Length(1), // Spacing
                Constraint::Min(10),   // Table
                Constraint::Length(1), // Jobs
                Constraint::Length(1), // Help
            ])
            .split(f.size());

        // Title
        let title = Paragraph::new("🖥️  NodeStat - Cluster Monitor")
            .style(Style::default().fg(Color::Magenta).add_modifier(Modifier::BOLD))
            .alignment(Alignment::Center);
        f.render_widget(title, chunks[0]);

        // Error message
        if let Some(ref error) = self.error_message {
            let error_msg = Paragraph::new(format!("Error: {}", error))
                .style(Style::default().fg(Color::Red).add_modifier(Modifier::BOLD));
            f.render_widget(error_msg, chunks[2]);
        } else {
            // Header
            let elapsed_secs = self.last_update.elapsed().as_secs();
            let header = format!("Partition: {}    Last update: {}s ago", 
                                self.current_partition, 
                                elapsed_secs);
            let header_widget = Paragraph::new(header)
                .style(Style::default().fg(Color::Cyan));
            f.render_widget(header_widget, chunks[2]);
        }

        // Stats
        self.render_stats(f, chunks[3]);

        // Table
        self.render_table(f, chunks[5]);

        // Jobs summary
        let jobs_summary = format!("Jobs: {} running ({} yours)", 
                                  self.jobs.len(), 
                                  self.user_jobs.len());
        let jobs_widget = Paragraph::new(jobs_summary)
            .style(Style::default().fg(Color::Yellow));
        f.render_widget(jobs_widget, chunks[6]);

        // Help
        let help = Paragraph::new("b: batch | m: highmem | g: gpu | r: refresh | q: quit | mouse: click/scroll")
            .style(Style::default().fg(Color::Gray));
        f.render_widget(help, chunks[7]);
    }

    fn render_stats(&self, f: &mut Frame, area: Rect) {
        let cpu_ratio = if self.stats.total_cores > 0 {
            self.stats.used_cores as f64 / self.stats.total_cores as f64
        } else {
            0.0
        };

        let mem_ratio = if self.stats.total_memory_gb > 0 {
            self.stats.used_memory_gb as f64 / self.stats.total_memory_gb as f64
        } else {
            0.0
        };

        let stats_layout = Layout::default()
            .direction(Direction::Vertical)
            .constraints([
                Constraint::Length(1),
                Constraint::Length(1),
                Constraint::Length(1),
            ])
            .split(area);

        // CPU gauge
        let cpu_gauge = Gauge::default()
            .block(Block::default().borders(Borders::NONE))
            .gauge_style(Style::default().fg(Color::Red))
            .percent((cpu_ratio * 100.0) as u16)
            .label(format!("CPU  {}/{}", self.stats.used_cores, self.stats.total_cores));
        f.render_widget(cpu_gauge, stats_layout[0]);

        // Memory gauge  
        let mem_gauge = Gauge::default()
            .block(Block::default().borders(Borders::NONE))
            .gauge_style(Style::default().fg(Color::Blue))
            .percent((mem_ratio * 100.0) as u16)
            .label(format!("MEM  {}GB/{}GB", self.stats.used_memory_gb, self.stats.total_memory_gb));
        f.render_widget(mem_gauge, stats_layout[1]);

        // Node summary
        let node_summary = Paragraph::new(format!("Nodes: {} total, {} available", 
                                                 self.stats.total_nodes, 
                                                 self.stats.avail_nodes));
        f.render_widget(node_summary, stats_layout[2]);
    }

    fn render_table(&mut self, f: &mut Frame, area: Rect) {
        let header_cells = ["Node", "CPU", "Memory", "Avail CPU", "Avail Mem", "State", "Jobs"]
            .iter()
            .map(|h| Cell::from(*h).style(Style::default().fg(Color::White).add_modifier(Modifier::BOLD)));
        let header = Row::new(header_cells).height(1).bottom_margin(1);

        let rows = self.nodes.iter().map(|node| {
            let user_has_jobs = self.user_has_jobs_on_node(&node.id);
            
            let node_name = if user_has_jobs {
                format!("★ {}", node.id)
            } else {
                node.id.clone()
            };

            let cpu_bar = self.create_progress_bar(node.used_cores, node.total_cores);
            let mem_bar = self.create_progress_bar(node.used_mem_gb(), node.total_mem_gb());
            
            let state_style = match node.state {
                NodeState::Idle => Style::default().fg(Color::Green),
                NodeState::Running => Style::default().fg(Color::Yellow),
                NodeState::Down | NodeState::Offline => Style::default().fg(Color::Gray),
                _ => Style::default().fg(Color::Red),
            };

            Row::new(vec![
                Cell::from(node_name).style(if user_has_jobs { Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD) } else { Style::default() }),
                Cell::from(cpu_bar),
                Cell::from(mem_bar),
                Cell::from(node.available_cores().to_string()),
                Cell::from(format!("{} GB", node.available_mem_gb())),
                Cell::from(node.state.to_string()).style(state_style),
                Cell::from(node.jobs.len().to_string()),
            ])
        });

        let table = Table::new(rows, [
            Constraint::Percentage(15),
            Constraint::Percentage(20),
            Constraint::Percentage(20),
            Constraint::Percentage(10),
            Constraint::Percentage(10),
            Constraint::Percentage(15),
            Constraint::Percentage(10),
        ])
            .header(header)
            .block(Block::default().borders(Borders::ALL).title("Nodes"))
            .highlight_style(Style::default().bg(Color::DarkGray).add_modifier(Modifier::BOLD))
            .highlight_symbol(">> ");

        f.render_stateful_widget(table, area, &mut self.table_state);
    }

    fn create_progress_bar(&self, used: u32, total: u32) -> String {
        if total == 0 {
            return "░░░░░░░░░░░░░░░░░░░░ 0/0".to_string();
        }

        let ratio = used as f64 / total as f64;
        let bar_length = 20;
        let filled_length = (ratio * bar_length as f64) as usize;
        
        let filled = "█".repeat(filled_length);
        let empty = "░".repeat(bar_length - filled_length);
        
        format!("{}{} {}/{}", filled, empty, used, total)
    }
}