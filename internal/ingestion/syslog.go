package ingestion

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/mr-andrej/junkyard/internal/storage"
)

type SyslogServer struct {
	db       *storage.DB
	listener net.Listener
	done     chan struct{}
}

func NewSyslogServer(db *storage.DB, address string) (*SyslogServer, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to start syslog server: %w", err)
	}

	log.Printf("🗑️  JUNKyard Syslog server listening on %s", address)

	return &SyslogServer{
		db:       db,
		listener: listener,
		done:     make(chan struct{}),
	}, nil
}

func (s *SyslogServer) Start() error {
	for {
		select {
		case <-s.done:
			return nil
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				continue
			}

			go s.handleConnection(conn)
		}
	}
}

func (s *SyslogServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().String()
	log.Printf("Syslog connection from %s", remoteAddr)

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		entry := s.parseSyslogMessage(line, remoteAddr)
		if entry != nil {
			if err := s.db.Insert(entry); err != nil {
				log.Printf("Failed to store syslog entry: %v", err)
			}
		}
	}
}

func (s *SyslogServer) parseSyslogMessage(msg string, remoteAddr string) *storage.LogEntry {
	// Simple RFC 3164 parser
	// Format: <priority>timestamp hostname tag: message
	// Example: <34>May  7 14:32:11 s1-app sshd[1234]: Failed password for admin

	original := msg

	// Extract priority
	var level string
	priorityRegex := regexp.MustCompile(`^<(\d+)>`)
	matches := priorityRegex.FindStringSubmatch(msg)

	if len(matches) > 1 {
		// Extract severity from priority
		// priority = facility * 8 + severity
		// severity: 0=emerg, 1=alert, 2=crit, 3=error, 4=warning, 5=notice, 6=info, 7=debug
		priority := matches[1]

		severityMap := map[rune]string{
			'0': "error",
			'1': "error",
			'2': "error",
			'3': "error",
			'4': "warning",
			'5': "info",
			'6': "info",
			'7': "debug",
		}

		// Get last digit for severity
		if len(priority) > 0 {
			lastDigit := rune(priority[len(priority)-1])
			if sev, ok := severityMap[lastDigit]; ok {
				level = sev
			}
		}

		msg = priorityRegex.ReplaceAllString(msg, "")
	}

	if level == "" {
		level = "info"
	}

	// Try to extract hostname from the message
	// Typical format: "May  7 14:32:11 hostname process: message"
	parts := strings.Fields(msg)
	var host, message, source string

	if len(parts) >= 4 {
		// parts[0-2] = timestamp (e.g., "May 7 14:32:11")
		// parts[3] = hostname
		// parts[4+] = process and message
		host = parts[3]
		message = strings.Join(parts[4:], " ")

		// Try to extract source/process name
		// Format: "sshd[1234]:" or "kernel:"
		if len(parts) >= 5 {
			processMatch := regexp.MustCompile(`^([a-zA-Z0-9_-]+)(\[\d+\])?:`).FindStringSubmatch(parts[4])
			if len(processMatch) > 1 {
				source = processMatch[1]
			}
		}
	} else {
		// Fallback: use remote IP as host
		host = strings.Split(remoteAddr, ":")[0]
		message = msg
	}

	if source == "" {
		source = "syslog"
	}

	// Detect specific sources based on message content
	if strings.Contains(message, "sshd") || strings.Contains(message, "SSH") {
		source = "ssh"
	} else if strings.Contains(message, "firewall") || strings.Contains(message, "pf:") {
		source = "firewall"
	} else if strings.Contains(message, "openvpn") || strings.Contains(message, "VPN") {
		source = "vpn"
	}

	return &storage.LogEntry{
		Timestamp: time.Now(),
		Host:      host,
		Source:    source,
		Level:     level,
		Message:   message,
		Raw:       original,
	}
}

func (s *SyslogServer) Stop() error {
	close(s.done)
	return s.listener.Close()
}
