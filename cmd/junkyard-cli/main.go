package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	Version = "1.0.0"
	Build   = "dev"
)

func main() {
	version := flag.Bool("version", false, "Show version")
	flag.Parse()

	if *version {
		fmt.Printf("junk v%s (build: %s)\n", Version, Build)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	command := args[0]

	switch command {
	case "logs":
		cmdLogs(args[1:])
	case "stream":
		cmdStream(args[1:])
	case "stats":
		cmdStats(args[1:])
	case "search":
		cmdSearch(args[1:])
	case "graph":
		cmdGraph(args[1:])
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`🗑️  JUNKyard CLI (junk) v` + Version)
	fmt.Print(`
Usage: junk <command> [options]

Commands:
  logs    - Display recent logs
  stream  - Stream logs in real-time
  stats   - Show log statistics
  search  - Search logs
  graph   - Display log graphs

Options:
  --host      Filter by hostname
  --source    Filter by source (syslog, ssh, firewall, etc.)
  --level     Filter by level (debug, info, warning, error)
  --limit     Limit number of results (default: 100)
  --hours     Time range in hours (default: 24)

Examples:
  junk logs
  junk logs --host s1-app --level error
  junk search "database"
  junk graph --hours 24
`)
}

func cmdLogs(args []string) {
	fmt.Println("TODO: Implement logs command")
}

func cmdStream(args []string) {
	fmt.Println("TODO: Implement stream command")
}

func cmdStats(args []string) {
	fmt.Println("TODO: Implement stats command")
}

func cmdSearch(args []string) {
	fmt.Println("TODO: Implement search command")
}

func cmdGraph(args []string) {
	fmt.Println("TODO: Implement graph command")
}
