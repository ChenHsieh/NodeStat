# NodeStat TUI 🖥️

A modern, interactive Terminal User Interface for monitoring HPC cluster nodes and jobs. Built with Go and the [Charm.sh](https://charm.sh) ecosystem for a beautiful, responsive experience.

![NodeStat TUI Demo](demo.gif)

## ✨ Features

- **🎯 Real-time Monitoring**: Auto-refresh every 30 seconds with manual refresh (r/space)
- **🏗️ Smart Node Sorting**: IDLE nodes first, then sorted by available resources (most powerful first)
- **⚡ Partition Switching**: Quick hotkeys to switch between batch (b), highmem (m), and gpu (g) partitions
- **📊 Visual Resource Bars**: Beautiful CPU/Memory usage visualization with progress bars
- **⭐ User Job Highlighting**: Your running jobs are highlighted with ★ on the node list
- **📈 Cluster Overview**: Real-time statistics showing total/used/available resources
- **⌨️ Vim-like Navigation**: hjkl/arrow keys for navigation, familiar keybindings
- **🎨 Modern UI**: Clean, colorful interface built with Bubble Tea and Lipgloss

## 🚀 Installation

### Prerequisites
- Go 1.20 or later
- SLURM or Torque scheduler (or use mock mode for testing)

### Build from Source
```bash
git clone <repository-url>
cd NodeStat
go mod tidy
go build -o nodestat .
```

### Quick Test (Demo Mode)
```bash
./nodestat -s mock -q batch
```

## 🎮 Usage

### Basic Commands
```bash
# Monitor batch partition (default)
./nodestat

# Monitor specific partition
./nodestat -q highmem_q

# Use with Torque/PBS
./nodestat -s torque -q batch

# Demo mode (no cluster required)
./nodestat -s mock
```

### Keyboard Shortcuts
| Key | Action |
|-----|--------|
| `b` | Switch to batch partition |
| `m` | Switch to highmem partition |
| `g` | Switch to gpu partition |
| `r` / `space` | Refresh data |
| `↑/k` `↓/j` | Navigate table |
| `q` | Quit |
| `?` | Show help |

## 📊 Interface Layout

```
┌─ NodeStat - Cluster Monitor ─────────────────────────────────────────┐
│                                                                       │
│ Partition: batch    Last update: 14:32:15                           │
│                                                                       │
│ ┌─ Cluster Overview ─────────────────────────────────────────────┐   │
│ │ CPU  ████████████████░░░░░░░░ 2156/3840                        │   │
│ │ MEM  ██████████████░░░░░░░░░░ 32TB/48TB                        │   │
│ │ Nodes: 156 total, 89 available                                 │   │
│ └─────────────────────────────────────────────────────────────────┘   │
│                                                                       │
│ Nodes                                                                 │
│ ┌─────────────────────────────────────────────────────────────────┐   │
│ │ Node     CPU                    Memory                Avail ... │   │
│ │ ★ batch001 ░░░░░░░░░░░░░░░░░░░░ ░░░░░░░░░░░░░░░░░░░░    48    ... │   │
│ │   batch002 ████████░░░░░░░░░░░░ ██████░░░░░░░░░░░░░░    24    ... │   │
│ │   ...                                                           │   │
│ └─────────────────────────────────────────────────────────────────┘   │
│                                                                       │
│ Jobs: 142 running (3 yours)                                          │
│                                                                       │
│ b: batch | m: highmem | g: gpu | r: refresh | q: quit                │
└───────────────────────────────────────────────────────────────────────┘
```

## 🔧 Configuration

### Command Line Options
```
  -q string    Partition/queue to display (default: batch)
  -s string    Scheduler system: slurm, torque, or mock (default: slurm)
               Use 'mock' for testing/demo without a real cluster
  -h          Show this help message
  -v          Show version information
```

### Common Partitions
| Partition | Description |
|-----------|-------------|
| `batch` | Standard compute nodes |
| `highmem_q` | High memory nodes |
| `gpu_q` | GPU-enabled nodes |
| `s_interq` | Interactive queue |

## 🔄 Migrating from Python Version

The new Go TUI version provides the same core functionality as the original Python script but with significant improvements:

### Key Differences
| Python Version | Go TUI Version |
|----------------|----------------|
| Static text output | Interactive TUI with navigation |
| Manual refresh only | Auto-refresh + manual |
| Individual node focus | Partition-centric view |
| Basic colored bars | Rich visual progress bars |
| Command-line filtering | Real-time interactive filtering |
| Simple job list | Integrated job highlighting |

### Command Migration
```bash
# Old Python approach
python3 node_stat.py -q batch --jobs --avail

# New Go TUI approach  
./nodestat -q batch
# Then use 'b', 'm', 'g' keys to switch partitions
# Jobs and availability shown automatically
```

## 🏗️ Architecture

Built with modern Go practices and the Charm.sh ecosystem:

- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)**: Reactive TUI framework
- **[Bubbles](https://github.com/charmbracelet/bubbles)**: Pre-built UI components
- **[Lipgloss](https://github.com/charmbracelet/lipgloss)**: Styling and layout
- **Scheduler Abstraction**: Clean interfaces for SLURM/Torque/Mock

### Project Structure
```
├── main.go                    # CLI and application entry
├── internal/
│   ├── models/               # Data models (Node, Job, etc.)
│   ├── scheduler/            # Scheduler interfaces & implementations
│   │   ├── interface.go      # Common interface
│   │   ├── slurm.go         # SLURM implementation  
│   │   ├── torque.go        # Torque implementation
│   │   └── mock.go          # Mock data for testing
│   └── ui/                  # Bubble Tea TUI components
│       └── app.go           # Main application model
└── go.mod                   # Go dependencies
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test with mock mode: `./nodestat -s mock`
5. Submit a pull request

## 📝 License

MIT License - see original Python version for details.

## 🙏 Acknowledgments

- Original Python version for the foundation
- [Charm.sh](https://charm.sh) for the amazing TUI toolkit
- HPC community for inspiration and requirements

---

*Built with ❤️ using Go and the Charm.sh ecosystem*