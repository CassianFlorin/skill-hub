package audit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRecordAndList(t *testing.T) {
	t.Setenv("SKILLHUB_HOME", t.TempDir())
	events, err := List(0)
	if err != nil {
		t.Fatalf("List on empty log: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected no events, got %d", len(events))
	}

	if err := Record(Event{Command: "install", Identity: "acme/demo", Version: "1.0.0"}); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if err := Record(Event{Command: "update", Identity: "acme/demo", FromVersion: "1.0.0", Version: "1.1.0"}); err != nil {
		t.Fatalf("Record: %v", err)
	}
	RecordOutcome(Event{Command: "deploy", Identity: "acme/demo", Runtime: "codex"}, os.ErrPermission)

	events, err = List(0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[0].Command != "install" || events[0].Result != ResultOK || events[0].Time == "" {
		t.Errorf("install event = %+v", events[0])
	}
	if events[1].FromVersion != "1.0.0" || events[1].Version != "1.1.0" {
		t.Errorf("update event = %+v", events[1])
	}
	if events[2].Result != ResultError || events[2].Detail == "" {
		t.Errorf("deploy error event = %+v", events[2])
	}

	limited, err := List(2)
	if err != nil {
		t.Fatalf("List limited: %v", err)
	}
	if len(limited) != 2 || limited[0].Command != "update" {
		t.Errorf("limited events = %+v", limited)
	}
}

func TestListSkipsMalformedLines(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SKILLHUB_HOME", home)
	content := `{"command":"install","identity":"a/x","result":"ok","time":"t"}
not json
{"command":"uninstall","identity":"a/x","result":"ok","time":"t"}
`
	if err := os.WriteFile(filepath.Join(home, LogFileName), []byte(content), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}
	events, err := List(0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(events) != 2 || events[1].Command != "uninstall" {
		t.Errorf("events = %+v", events)
	}
}
