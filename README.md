# NodeStat TUI 🖥️

A modern, interactive Terminal User Interface for monitoring HPC cluster nodes and jobs. Available in two implementations for different deployment needs.

## 🚀 Quick Start

### Option 1: Install via PyPI (Recommended for HPC environments)
```bash
# Install with pipx (recommended)
pipx install nodestat-tui

# Start monitoring
nodestat
```

### Option 2: Run Go Version
```bash
# Build from source
go build -o nodestat .

# Start monitoring
./nodestat
```

## ✨ Features

- **🎯 Real-time Monitoring**: Auto-refresh with manual refresh (r/space)
- **🏗️ Smart Node Sorting**: IDLE nodes first, sorted by available resources
- **⚡ Partition Switching**: Quick hotkeys for batch (b), highmem (m), gpu (g)
- **📊 Visual Resource Bars**: Beautiful CPU/Memory usage visualization
- **⭐ User Job Highlighting**: Your running jobs highlighted with ★
- **🖱️ Mouse Support**: Click to navigate and scroll through nodes
- **⌨️ Vim-like Navigation**: hjkl/arrow keys, familiar shortcuts

## 📦 Project Structure

This project provides two equivalent implementations:

### 1. **Rust Version** (`nodestat-rust/`)
- **Target**: PyPI distribution for easy installation
- **Best for**: Production HPC environments (install with pipx)
- **Framework**: Ratatui + Crossterm
- **Install**: `pipx install nodestat-tui`

### 2. **Go Version** (root directory)
- **Target**: Standalone binary distribution  
- **Best for**: Development and direct deployment
- **Framework**: Bubble Tea + Lipgloss (Charm.sh ecosystem)
- **Install**: `go build -o nodestat .`

Both versions provide identical functionality and user experience.

## 🎮 Usage

```bash
# Monitor batch partition (default)
nodestat

# Monitor specific partition
nodestat -q highmem_q

# Demo mode (no cluster required)
nodestat -s mock -q batch

# Switch partitions with hotkeys: b=batch, m=highmem, g=gpu
# Navigate with: hjkl or arrow keys
# Refresh with: r or space
# Mouse: click to select, scroll to navigate
```

## 🏗️ Supported Schedulers

- **SLURM**: Production HPC clusters
- **Torque/PBS**: Legacy HPC systems  
- **Mock**: Testing and development

## 📋 Legacy Python Version

The original Python script (`node_stat.py`) is preserved for reference but is superseded by the modern TUI versions above.
