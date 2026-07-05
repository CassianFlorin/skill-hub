package audit

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/CassianFlorin/skill-hub/internal/config"
)

const LogFileName = "audit.jsonl"

const (
	ResultOK    = "ok"
	ResultError = "error"
)

type Event struct {
	Time         string `json:"time"`
	Command      string `json:"command"`
	Identity     string `json:"identity,omitempty"`
	Version      string `json:"version,omitempty"`
	FromVersion  string `json:"from_version,omitempty"`
	SourceRef    string `json:"source_ref,omitempty"`
	SourceCommit string `json:"source_commit,omitempty"`
	Checksum     string `json:"checksum,omitempty"`
	Runtime      string `json:"runtime,omitempty"`
	Registry     string `json:"registry,omitempty"`
	Result       string `json:"result"`
	Detail       string `json:"detail,omitempty"`
}

func LogPath() (string, error) {
	home, err := config.DefaultHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, LogFileName), nil
}

// Record appends one event to the local audit log. Auditing is
// best-effort: callers ignore the returned error so a failed write
// never blocks the operation being audited.
func Record(event Event) error {
	if event.Time == "" {
		event.Time = time.Now().UTC().Format(time.RFC3339)
	}
	if event.Result == "" {
		event.Result = ResultOK
	}
	path, err := LogPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(data)
	return err
}

// RecordOutcome records an event whose result depends on err and
// returns nothing; it is safe to call with a nil error.
func RecordOutcome(event Event, err error) {
	if err != nil {
		event.Result = ResultError
		event.Detail = err.Error()
	}
	_ = Record(event)
}

// List returns the most recent events, newest last. A limit of 0
// returns all events. Malformed lines are skipped.
func List(limit int) ([]Event, error) {
	path, err := LogPath()
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()
	var events []Event
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var event Event
		if err := json.Unmarshal(line, &event); err != nil {
			continue
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if limit > 0 && len(events) > limit {
		events = events[len(events)-limit:]
	}
	return events, nil
}
