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
    db * storage.DB
    listener net.Listener
    done chan struct {}
}

func NewSyslogServer(db * storage.DB, address string)( * SyslogServer, error) {
    listener, err: = net.Listen("tcp", address)
    if err != nil {
        return nil, fmt.Errorf("failed to start syslog server: %w", err)
    }

    log.Printf("🗑️  JUNKyard Syslog server listening on %s", address)

    s: = & SyslogServer {
        db: db,
        listener: listener,
        done: make(chan struct {}),
    }

    go func() {
        udpAddr, err: = net.ResolveUDPAddr("udp", address)
        if err != nil {
            log.Printf("Failed to resolve UDP address: %v", err)
            return
        }
        conn, err: = net.ListenUDP("udp", udpAddr)
        if err != nil {
            log.Printf("Failed to start UDP syslog listener: %v", err)
            return
        }
        defer conn.Close()
        log.Printf("🗑️  JUNKyard Syslog UDP listener on %s", address)
        buf: = make([] byte, 65535)
        for {
            n, remoteAddr, err: = conn.ReadFromUDP(buf)
            if err != nil {
                continue
            }
            line: = strings.TrimSpace(string(buf[: n]))
            if line == "" {
                continue
            }
            entry: = s.parseSyslogMessage(line, remoteAddr.String())
            if entry != nil {
                if err: = s.db.Insert(entry);
                err != nil {
                    log.Printf("Failed to store UDP syslog entry: %v", err)
                }
            }
        }
    }()

    return s, nil
}

func(s * SyslogServer) Start() error {
    for {
        select {
            case <-s.done:
                return nil
            default:
                conn, err: = s.listener.Accept()
                if err != nil {
                    continue
                }
                go s.handleConnection(conn)
        }
    }
}

func(s * SyslogServer) handleConnection(conn net.Conn) {
    defer conn.Close()

    remoteAddr: = conn.RemoteAddr().String()
    log.Printf("Syslog connection from %s", remoteAddr)

    scanner: = bufio.NewScanner(conn)
    for scanner.Scan() {
        line: = scanner.Text()
        if line == "" {
            continue
        }
        entry: = s.parseSyslogMessage(line, remoteAddr)
        if entry != nil {
            if err: = s.db.Insert(entry);
            err != nil {
                log.Printf("Failed to store syslog entry: %v", err)
            }
        }
    }
}

var hostMap = map[string] string {
    "192.168.20.254": "s2-fw",
    "192.168.20.1": "s2-mt",
    "192.168.10.10": "s2-js",
    "10.0.10.254": "s1-fw",
    "10.0.10.1": "s1-app",
    "10.0.20.1": "s1-db",
}

func resolveHost(ip string) string {
    if name, ok: = hostMap[ip];
    ok {
        return name
    }
    return ip
}

func(s * SyslogServer) parseSyslogMessage(msg string, remoteAddr string) * storage.LogEntry {
    original: = msg

        var level string
    priorityRegex: = regexp.MustCompile(`^<(\d+)>`)
    matches: = priorityRegex.FindStringSubmatch(msg)

    if len(matches) > 1 {
        priority: = matches[1]
        severityMap: = map[rune] string {
            '0': "error",
            '1': "error",
            '2': "error",
            '3': "error",
            '4': "warning",
            '5': "info",
            '6': "info",
            '7': "debug",
        }
        if len(priority) > 0 {
            lastDigit: = rune(priority[len(priority) - 1])
            if sev,
            ok: = severityMap[lastDigit];ok {
                level = sev
            }
        }
        msg = priorityRegex.ReplaceAllString(msg, "")
    }

    if level == "" {
        level = "info"
    }

    parts: = strings.Fields(msg)
    var host, message, source string

    if len(parts) >= 4 {
        // If parts[3] contains [ or / or : it's a process name, not a hostname
        if strings.ContainsAny(parts[3], "[/:") {
            host = resolveHost(strings.Split(remoteAddr, ":")[0])
            message = strings.Join(parts[3: ], " ")
        } else {
            host = parts[3]
            message = strings.Join(parts[4: ], " ")
        }
        if len(parts) >= 5 {
            processMatch: = regexp.MustCompile(`^([a-zA-Z0-9_-]+)(\[\d+\])?:`).FindStringSubmatch(parts[4])
            if len(processMatch) > 1 {
                source = processMatch[1]
            }
        }
    } else {
        host = resolveHost(strings.Split(remoteAddr, ":")[0])
        message = msg
    }

    if source == "" {
        source = "syslog"
    }

    if strings.Contains(message, "sshd") || strings.Contains(message, "SSH") {
        source = "ssh"
    } else if strings.Contains(message, "firewall") || strings.Contains(message, "pf:") {
        source = "firewall"
    } else if strings.Contains(message, "openvpn") || strings.Contains(message, "VPN") {
        source = "vpn"
    }

    return &storage.LogEntry {
        Timestamp: time.Now(),
        Host: host,
        Source: source,
        Level: level,
        Message: message,
        Raw: original,
    }
}

func(s * SyslogServer) Stop() error {
    close(s.done)
    return s.listener.Close()
}
