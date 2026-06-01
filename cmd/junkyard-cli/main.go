package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
)

var Version = "1.0.0"

var (
	apiBase      string
	errColor     = color.New(color.FgRed)
	warnColor    = color.New(color.FgYellow)
	infoColor    = color.New(color.FgCyan)
	debugColor   = color.New(color.FgWhite, color.Faint)
	successColor = color.New(color.FgGreen)
)

type LogEntry struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Host      string    `json:"host"`
	Source    string    `json:"source"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Raw       string    `json:"raw"`
}

type Stats struct {
	Total    int64            `json:"total"`
	ByLevel  map[string]int64 `json:"by_level"`
	ByHost   map[string]int64 `json:"by_host"`
	BySource map[string]int64 `json:"by_source"`
	DBSizeMB float64          `json:"db_size_mb"`
	LastHour int64            `json:"last_hour"`
	Last24h  int64            `json:"last_24h"`
}

type TimeSeriesData struct {
	Interval string `json:"interval"`
	Hours    int    `json:"hours"`
	Data     []struct {
		Timestamp time.Time `json:"timestamp"`
		Count     int64     `json:"count"`
	} `json:"data"`
}

func init() {
	apiBase = os.Getenv("JUNKYARD_SERVER")
	if apiBase == "" {
		apiBase = "http://localhost:8080"
	}
	if !strings.HasPrefix(apiBase, "http") {
		apiBase = "http://" + apiBase
	}
}

func main() {
	version := flag.Bool("version", false, "Show version")
	flag.Parse()

	if *version {
		fmt.Printf("JUNKyard CLI v%s\n", Version)
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
	case "health":
		cmdHealth(args[1:])
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(color.GreenString("🗑️  JUNKyard CLI v" + Version + "\n\n"))
	fmt.Print(`Usage: junk <command> [options]

Commands:
  logs    - Display recent logs with filtering
  stream  - Stream logs in real-time (polls every 5s)
  stats   - Show log statistics and breakdown
  search  - Full-text search across logs
  graph   - Display log trends as ASCII graph
  health  - Check server health

Global Options:
  --host      Filter by hostname
  --source    Filter by source (syslog, http, etc.)
  --level     Filter by level (debug, info, warning, error)
  --limit     Limit number of results (default: 100)
  --hours     Time range in hours (default: 24)
  --server    API server address (default: http://localhost:8080)

Examples:
  junk logs                              # Last 100 logs
  junk logs --host S1-APP --level error  # Errors on S1-APP
  junk search "database"                 # Full-text search
  junk graph --hours 24                  # Last 24 hours
  junk stream                            # Real-time streaming
  junk stats                             # Statistics
  junk health                            # Server status
`)
}

func cmdLogs(args []string) {
	fs := flag.NewFlagSet("logs", flag.ExitOnError)
	host := fs.String("host", "", "Filter by host")
	source := fs.String("source", "", "Filter by source")
	level := fs.String("level", "", "Filter by level")
	limit := fs.Int("limit", 100, "Limit results")
	fs.Parse(args)

	params := url.Values{}
	if *host != "" {
		params.Add("host", *host)
	}
	if *source != "" {
		params.Add("source", *source)
	}
	if *level != "" {
		params.Add("level", *level)
	}
	params.Add("limit", fmt.Sprintf("%d", *limit))

	resp, err := http.Get(apiBase + "/api/logs?" + params.Encode())
	if err != nil {
		errColor.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var logs []LogEntry
	if err := json.NewDecoder(resp.Body).Decode(&logs); err != nil {
		errColor.Printf("Error decoding response: %v\n", err)
		os.Exit(1)
	}

	if len(logs) == 0 {
		fmt.Println("No logs found")
		return
	}

	fmt.Printf("%-20s %-15s %-12s %-10s %s\n", "TIME", "HOST", "SOURCE", "LEVEL", "MESSAGE")
	fmt.Println(strings.Repeat("-", 120))

	for _, log := range logs {
		levelStr := strings.ToUpper(log.Level)
		var levelColor color.Attribute
		switch log.Level {
		case "error":
			levelColor = color.FgRed
		case "warning":
			levelColor = color.FgYellow
		case "info":
			levelColor = color.FgCyan
		case "debug":
			levelColor = color.FgWhite
		default:
			levelColor = color.FgWhite
		}

		msg := log.Message
		if len(msg) > 50 {
			msg = msg[:47] + "..."
		}

		fmt.Printf("%-20s %-15s %-12s %-10s %s\n",
			log.Timestamp.Format("15:04:05"),
			color.New(color.FgCyan).Sprint(log.Host),
			color.New(color.FgMagenta).Sprint(log.Source),
			color.New(levelColor).Sprint(levelStr),
			msg,
		)
	}
}

func cmdStream(args []string) {
	fs := flag.NewFlagSet("stream", flag.ExitOnError)
	host := fs.String("host", "", "Filter by host")
	level := fs.String("level", "", "Filter by level")
	limit := fs.Int("limit", 50, "Limit results")
	fs.Parse(args)

	fmt.Println(successColor.Sprint("Starting stream (Ctrl+C to stop)..."))
	fmt.Println()

	var lastID int64 = 0

	for {
		params := url.Values{}
		if *host != "" {
			params.Add("host", *host)
		}
		if *level != "" {
			params.Add("level", *level)
		}
		params.Add("limit", fmt.Sprintf("%d", *limit))

		resp, err := http.Get(apiBase + "/api/logs?" + params.Encode())
		if err != nil {
			errColor.Printf("Connection error: %v\n", err)
			time.Sleep(5 * time.Second)
			continue
		}

		var logs []LogEntry
		if err := json.NewDecoder(resp.Body).Decode(&logs); err != nil {
			resp.Body.Close()
			time.Sleep(5 * time.Second)
			continue
		}
		resp.Body.Close()

		for _, log := range logs {
			if log.ID > lastID {
				lastID = log.ID
				levelColor := color.FgWhite
				switch log.Level {
				case "error":
					levelColor = color.FgRed
				case "warning":
					levelColor = color.FgYellow
				case "info":
					levelColor = color.FgCyan
				}

				fmt.Printf("%s [%s] %s %s: %s\n",
					log.Timestamp.Format("15:04:05"),
					color.New(levelColor).Sprint(strings.ToUpper(log.Level)),
					color.New(color.FgCyan).Sprint(log.Host),
					color.New(color.FgMagenta).Sprint(log.Source),
					log.Message,
				)
			}
		}

		time.Sleep(5 * time.Second)
	}
}

func cmdStats(args []string) {
	resp, err := http.Get(apiBase + "/api/stats")
	if err != nil {
		errColor.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var stats Stats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		errColor.Printf("Error decoding response: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Printf("%-20s %s\n", color.GreenString("Total Logs:"), successColor.Sprintf("%d", stats.Total))
	fmt.Printf("%-20s %s\n", color.GreenString("Last Hour:"), infoColor.Sprintf("%d", stats.LastHour))
	fmt.Printf("%-20s %s\n", color.GreenString("Last 24 Hours:"), infoColor.Sprintf("%d", stats.Last24h))
	fmt.Printf("%-20s %s MB\n", color.GreenString("Database Size:"), infoColor.Sprintf("%.1f", stats.DBSizeMB))
	fmt.Println()

	if len(stats.ByLevel) > 0 {
		fmt.Println(color.BlueString("Log Levels:"))
		fmt.Printf("%-12s %-10s %s\n", "LEVEL", "COUNT", "PERCENTAGE")
		fmt.Println(strings.Repeat("-", 40))
		for level, count := range stats.ByLevel {
			pct := float64(count) / float64(stats.Total) * 100
			fmt.Printf("%-12s %-10d %.1f%%\n", level, count, pct)
		}
		fmt.Println()
	}

	if len(stats.ByHost) > 0 {
		fmt.Println(color.BlueString("Top Hosts:"))
		fmt.Printf("%-15s %-10s %s\n", "HOST", "COUNT", "PERCENTAGE")
		fmt.Println(strings.Repeat("-", 40))
		for host, count := range stats.ByHost {
			pct := float64(count) / float64(stats.Total) * 100
			fmt.Printf("%-15s %-10d %.1f%%\n", host, count, pct)
		}
		fmt.Println()
	}

	if len(stats.BySource) > 0 {
		fmt.Println(color.BlueString("Log Sources:"))
		fmt.Printf("%-15s %-10s %s\n", "SOURCE", "COUNT", "PERCENTAGE")
		fmt.Println(strings.Repeat("-", 40))
		for source, count := range stats.BySource {
			pct := float64(count) / float64(stats.Total) * 100
			fmt.Printf("%-15s %-10d %.1f%%\n", source, count, pct)
		}
	}
}

func cmdSearch(args []string) {
	if len(args) == 0 {
		errColor.Println("Usage: junk search <query>")
		os.Exit(1)
	}

	query := args[0]
	params := url.Values{}
	params.Add("search", query)
	params.Add("limit", "200")

	resp, err := http.Get(apiBase + "/api/logs?" + params.Encode())
	if err != nil {
		errColor.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var logs []LogEntry
	if err := json.NewDecoder(resp.Body).Decode(&logs); err != nil {
		errColor.Printf("Error decoding response: %v\n", err)
		os.Exit(1)
	}

	if len(logs) == 0 {
		fmt.Printf("No results found for: %s\n", query)
		return
	}

	fmt.Printf("Found %d results for: %s\n\n", len(logs), color.YellowString(query))

	fmt.Printf("%-20s %-15s %-10s %s\n", "TIME", "HOST", "LEVEL", "MESSAGE")
	fmt.Println(strings.Repeat("-", 100))

	for _, log := range logs {
		levelStr := strings.ToUpper(log.Level)
		var levelColor color.Attribute
		switch log.Level {
		case "error":
			levelColor = color.FgRed
		case "warning":
			levelColor = color.FgYellow
		case "info":
			levelColor = color.FgCyan
		default:
			levelColor = color.FgWhite
		}

		msg := log.Message
		if len(msg) > 50 {
			msg = msg[:47] + "..."
		}

		fmt.Printf("%-20s %-15s %-10s %s\n",
			log.Timestamp.Format("15:04:05"),
			color.New(color.FgCyan).Sprint(log.Host),
			color.New(levelColor).Sprint(levelStr),
			msg,
		)
	}
}

func cmdGraph(args []string) {
	fs := flag.NewFlagSet("graph", flag.ExitOnError)
	hours := fs.Int("hours", 24, "Time range in hours")
	fs.Parse(args)

	params := url.Values{}
	params.Add("interval", "hour")
	params.Add("hours", fmt.Sprintf("%d", *hours))

	resp, err := http.Get(apiBase + "/api/timeseries?" + params.Encode())
	if err != nil {
		errColor.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var data TimeSeriesData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		errColor.Printf("Error decoding response: %v\n", err)
		os.Exit(1)
	}

	if len(data.Data) == 0 {
		fmt.Println("No data available for the specified time range")
		return
	}

	fmt.Printf("Log volume over the last %d hours:\n\n", *hours)

	// Simple ASCII graph
	maxCount := int64(0)
	for _, point := range data.Data {
		if point.Count > maxCount {
			maxCount = point.Count
		}
	}

	for _, point := range data.Data {
		barLength := int(point.Count * 50 / maxCount)
		bar := strings.Repeat("█", barLength)
		fmt.Printf("%s %s %d\n", point.Timestamp.Format("15:04"), infoColor.Sprint(bar), point.Count)
	}

	fmt.Printf("\nTotal: %d logs\n", len(data.Data))
}

func cmdHealth(args []string) {
	resp, err := http.Get(apiBase + "/health")
	if err != nil {
		errColor.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var health map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		errColor.Printf("Error decoding response: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Printf("%-20s %s\n", color.GreenString("Status:"), successColor.Sprint(health["status"]))
	fmt.Printf("%-20s %s\n", color.GreenString("Version:"), health["version"])
	fmt.Printf("%-20s %s\n", color.GreenString("Hostname:"), health["hostname"])
	fmt.Printf("%-20s %v\n", color.GreenString("Uptime:"), health["uptime_seconds"])

	if db, ok := health["database"].(map[string]interface{}); ok {
		fmt.Println()
		fmt.Println(color.BlueString("Database:"))
		if path, ok := db["path"].(string); ok {
			fmt.Printf("  Path: %s\n", path)
		}
		if size, ok := db["size_mb"].(float64); ok {
			fmt.Printf("  Size: %.1f MB\n", size)
		}
		if total, ok := db["total_logs"].(float64); ok {
			fmt.Printf("  Total Logs: %d\n", int64(total))
		}
	}
	fmt.Println()
}
