package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"nodestat/internal/scheduler"
	"nodestat/internal/ui"
)

func main() {
	// Command line flags
	var (
		partition   = flag.String("q", "batch", "partition/queue to display (e.g., batch, highmem_q, gpu_q)")
		system      = flag.String("s", "slurm", "scheduler system to use (slurm or torque)")
		showHelp    = flag.Bool("h", false, "show help")
		showVersion = flag.Bool("v", false, "show version")
	)
	flag.Parse()

	if *showHelp {
		printHelp()
		return
	}

	if *showVersion {
		fmt.Println("NodeStat TUI v1.0.0")
		fmt.Println("A modern terminal UI for cluster monitoring")
		return
	}

	// Validate scheduler type
	var schedulerType scheduler.SchedulerType
	switch *system {
	case "slurm":
		schedulerType = scheduler.SLURM
	case "torque":
		schedulerType = scheduler.Torque
	case "mock":
		schedulerType = scheduler.Mock
	default:
		fmt.Fprintf(os.Stderr, "Error: Invalid scheduler type '%s'. Use 'slurm', 'torque', or 'mock'\n", *system)
		os.Exit(1)
	}

	// Create and run the application
	app := ui.NewApp(schedulerType, *partition)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func printHelp() {
	fmt.Println("NodeStat TUI - Modern cluster monitoring interface")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  nodestat [options]")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("  -q string    Partition/queue to display (default: batch)")
	fmt.Println("  -s string    Scheduler system: slurm, torque, or mock (default: slurm)")
	fmt.Println("               Use 'mock' for testing/demo without a real cluster")
	fmt.Println("  -h          Show this help message")
	fmt.Println("  -v          Show version information")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  nodestat -q batch")
	fmt.Println("  nodestat -q highmem_q -s slurm")
	fmt.Println("  nodestat -q gpu_q")
	fmt.Println("  nodestat -s mock          # Demo mode for testing")
	fmt.Println()
	fmt.Println("KEYBOARD SHORTCUTS:")
	fmt.Println("  b           Switch to batch partition")
	fmt.Println("  m           Switch to highmem partition")
	fmt.Println("  g           Switch to gpu partition")
	fmt.Println("  r/space     Refresh data")
	fmt.Println("  ↑/k ↓/j     Navigate table")
	fmt.Println("  q           Quit")
	fmt.Println()
	fmt.Println("FEATURES:")
	fmt.Println("  • Real-time node monitoring with 30s auto-refresh")
	fmt.Println("  • Nodes sorted by availability (IDLE first, then by power)")
	fmt.Println("  • Visual CPU/Memory usage bars")
	fmt.Println("  • User's jobs highlighted with ★")
	fmt.Println("  • Partition switching with hotkeys")
	fmt.Println("  • Cluster overview statistics")
}
