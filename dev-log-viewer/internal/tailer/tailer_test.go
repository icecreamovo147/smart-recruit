package tailer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotBoundsAndNoFinalNewline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "service.log")
	write(t, path, "one\ntwo\nthree")

	tailer := NewFileTailer("svc", path)
	event := tailer.Snapshot(2)
	if event.Err != nil {
		t.Fatal(event.Err)
	}
	if event.Type != EventSnapshot {
		t.Fatalf("type = %q", event.Type)
	}
	assertLines(t, event.Lines, []string{"two", "three"})

	empty := filepath.Join(t.TempDir(), "empty.log")
	write(t, empty, "")
	event = NewFileTailer("svc", empty).Snapshot(MaxSnapshotLines + 100)
	if event.Err != nil {
		t.Fatal(event.Err)
	}
	assertLines(t, event.Lines, nil)
}

func TestPollAppendHalfLineAndUTF8(t *testing.T) {
	path := filepath.Join(t.TempDir(), "service.log")
	write(t, path, "ready\n")
	tailer := NewFileTailer("svc", path)
	if event := tailer.Snapshot(10); event.Err != nil {
		t.Fatal(event.Err)
	}

	appendFile(t, path, "half")
	if events := tailer.Poll(); len(events) != 0 {
		t.Fatalf("half line produced events: %+v", events)
	}
	appendFile(t, path, "🙂\nnext\n")
	events := tailer.Poll()
	if len(events) != 1 || events[0].Type != EventLines {
		t.Fatalf("events = %+v", events)
	}
	assertLines(t, events[0].Lines, []string{"half🙂", "next"})
}

func TestPollUTF8SplitAcrossPolls(t *testing.T) {
	path := filepath.Join(t.TempDir(), "service.log")
	write(t, path, "")
	tailer := NewFileTailer("svc", path)
	if event := tailer.Snapshot(10); event.Err != nil {
		t.Fatal(event.Err)
	}

	appendBytes(t, path, []byte{'h', 'i', ' ', 0xF0, 0x9F})
	if events := tailer.Poll(); len(events) != 0 {
		t.Fatalf("partial utf8 produced events: %+v", events)
	}
	appendBytes(t, path, []byte{0x99, 0x82, '\n'})
	events := tailer.Poll()
	if len(events) != 1 {
		t.Fatalf("events = %+v", events)
	}
	assertLines(t, events[0].Lines, []string{"hi 🙂"})
}

func TestPollTruncateDeleteAndRecreateReset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "service.log")
	write(t, path, "a\nb\n")
	tailer := NewFileTailer("svc", path)
	if event := tailer.Snapshot(10); event.Err != nil {
		t.Fatal(event.Err)
	}

	write(t, path, "n\n")
	events := tailer.Poll()
	if len(events) != 2 || events[0].Type != EventReset || events[1].Type != EventLines {
		t.Fatalf("truncate events = %+v", events)
	}
	if events[0].Generation != 1 || events[1].Generation != 1 {
		t.Fatalf("generation after truncate = %+v", events)
	}
	assertLines(t, events[1].Lines, []string{"n"})

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if events := tailer.Poll(); len(events) != 0 {
		t.Fatalf("delete events = %+v", events)
	}
	write(t, path, "again\n")
	events = tailer.Poll()
	if len(events) != 2 || events[0].Type != EventReset || events[1].Type != EventLines {
		t.Fatalf("recreate events = %+v", events)
	}
	if events[0].Generation != 2 {
		t.Fatalf("generation after recreate = %d", events[0].Generation)
	}
	assertLines(t, events[1].Lines, []string{"again"})
}

func TestOneFileErrorDoesNotAffectAnotherTailer(t *testing.T) {
	root := t.TempDir()
	bad := NewFileTailer("bad", filepath.Join(root, "missing", "bad.log"))
	goodPath := filepath.Join(root, "good.log")
	write(t, goodPath, "ok\n")
	good := NewFileTailer("good", goodPath)

	if events := bad.Poll(); len(events) != 0 {
		t.Fatalf("missing file should wait without busy error: %+v", events)
	}
	event := good.Snapshot(10)
	if event.Err != nil {
		t.Fatal(event.Err)
	}
	assertLines(t, event.Lines, []string{"ok"})
}

func write(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func appendFile(t *testing.T, path string, content string) {
	t.Helper()
	appendBytes(t, path, []byte(content))
}

func appendBytes(t *testing.T, path string, content []byte) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := file.Write(content); err != nil {
		t.Fatal(err)
	}
}

func assertLines(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("lines = %#v, want %#v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("lines = %#v, want %#v", got, want)
		}
	}
}
