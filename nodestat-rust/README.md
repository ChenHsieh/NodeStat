# NodeStat TUI - Rust Version for PyPI

A fast, modern Terminal User Interface for monitoring HPC cluster nodes, built in Rust and packaged for easy installation via pip/pipx.

## 🚀 Quick Install

```bash
# Install with pipx (recommended)
pipx install nodestat-tui

# Or install with pip
pip install nodestat-tui
```

## ✨ Features

- **⚡ Lightning Fast**: Written in Rust for maximum performance
- **🎯 Partition-Centric**: Switch between batch/highmem/gpu with hotkeys
- **📊 Visual Bars**: Beautiful CPU/Memory usage visualization  
- **⭐ User Jobs**: Your jobs highlighted with ★ in node list
- **🔄 Real-time**: Auto-refresh with smooth updates
- **⌨️ Vim Keys**: hjkl navigation, familiar shortcuts

## 🎮 Usage

```bash
# Start monitoring (defaults to batch partition)
nodestat

# Monitor specific partition  
nodestat -q highmem_q

# Demo mode (no cluster needed)
nodestat -s mock
```

### Keyboard Controls
- `b/m/g` - Switch to batch/highmem/gpu partitions
- `j/k` or `↑↓` - Navigate node table
- `r/space` - Manual refresh
- `q` - Quit

## 🏗️ Why Rust + PyPI?

This approach gives you the best of both worlds:
- **Rust Performance**: Native speed, low resource usage
- **Easy Installation**: `pipx install` works on any cluster
- **No Dependencies**: Single binary, no runtime requirements
- **HPC Friendly**: Works great on SLURM/Torque systems

Perfect for environments like Sapelo2 where you want easy installation but maximum performance.

## 🔧 Requirements

- Python 3.8+ (for installation only)
- SLURM or Torque scheduler
- Linux x86_64

## 📊 Interface Preview

```
🖥️  NodeStat - Cluster Monitor

Partition: batch    Last update: 2s ago

CPU  ████████████████░░░░ 2156/3840
MEM  ██████████████░░░░░░ 32GB/48GB  
Nodes: 156 total, 89 available

┌─ Nodes ──────────────────────────────────────────────┐
│ Node     CPU              Memory           Avail ... │
│ ★ batch001 ░░░░░░░░░░░░░░░░░░░░ ░░░░░░░░░░░░░░░░   48  │
│   batch002 ████████░░░░░░░░░░░░ ██████░░░░░░░░░░   24  │
│ >> batch003 ░░░░░░░░░░░░░░░░░░░░ ░░░░░░░░░░░░░░░░   56  │
└───────────────────────────────────────────────────────┘

Jobs: 142 running (3 yours)
b: batch | m: highmem | g: gpu | r: refresh | q: quit
```

## 🛠️ Development

Built with modern Rust ecosystem:
- [Ratatui](https://ratatui.rs/) for the TUI interface
- [Tokio](https://tokio.rs/) for async operations  
- [Clap](https://clap.rs/) for CLI parsing
- [Crossterm](https://crates.io/crates/crossterm) for terminal control

See the [GitHub repository](https://github.com/pbasting/NodeStat) for source code and contributing guidelines.