package tailer

import (
	"errors"
	"io"
	"os"
	"strings"
	"syscall"
	"unicode/utf8"
)

const (
	DefaultSnapshotLines = 300
	MaxSnapshotLines     = 1000
	DefaultMaxReadBytes  = 1024 * 1024

	EventSnapshot = "snapshot"
	EventLines    = "lines"
	EventReset    = "reset"
	EventError    = "error"
)

type Event struct {
	Type       string
	Service    string
	Generation uint64
	Lines      []string
	Err        error
}

type FileTailer struct {
	service      string
	path         string
	maxReadBytes int64
	offset       int64
	identity     identity
	generation   uint64
	partial      []byte
	missing      bool
}

func NewFileTailer(service string, path string) *FileTailer {
	return &FileTailer{
		service:      service,
		path:         path,
		maxReadBytes: DefaultMaxReadBytes,
		missing:      true,
	}
}

func (t *FileTailer) Generation() uint64 {
	return t.generation
}

func (t *FileTailer) Snapshot(lines int) Event {
	limit := clampLines(lines)
	content, info, id, err := readTail(t.path, limit, t.maxReadBytes)
	if err != nil {
		t.missing = os.IsNotExist(err)
		return Event{Type: EventError, Service: t.service, Generation: t.generation, Err: err}
	}
	t.identity = id
	t.offset = info.Size()
	t.partial = nil
	t.missing = false
	return Event{Type: EventSnapshot, Service: t.service, Generation: t.generation, Lines: content}
}

func (t *FileTailer) Poll() []Event {
	info, err := os.Stat(t.path)
	if err != nil {
		if os.IsNotExist(err) {
			t.missing = true
			t.partial = nil
			return nil
		}
		return []Event{{Type: EventError, Service: t.service, Generation: t.generation, Err: err}}
	}
	if info.IsDir() {
		return []Event{{Type: EventError, Service: t.service, Generation: t.generation, Err: errors.New("tail path is a directory")}}
	}

	id := fileIdentity(info)
	events := []Event{}
	if t.missing || t.identity != (identity{}) && t.identity != id || info.Size() < t.offset {
		t.generation++
		t.offset = 0
		t.partial = nil
		events = append(events, Event{Type: EventReset, Service: t.service, Generation: t.generation})
	}
	t.identity = id
	t.missing = false

	if info.Size() == t.offset {
		return events
	}
	lines, err := t.readFromOffset(info.Size())
	if err != nil {
		return append(events, Event{Type: EventError, Service: t.service, Generation: t.generation, Err: err})
	}
	if len(lines) > 0 {
		events = append(events, Event{Type: EventLines, Service: t.service, Generation: t.generation, Lines: lines})
	}
	return events
}

func (t *FileTailer) readFromOffset(size int64) ([]string, error) {
	readTo := size
	if readTo-t.offset > t.maxReadBytes {
		readTo = t.offset + t.maxReadBytes
	}
	file, err := os.Open(t.path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	length := readTo - t.offset
	buf := make([]byte, length)
	n, err := file.ReadAt(buf, t.offset)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	t.offset += int64(n)
	return splitCompleteLines(&t.partial, buf[:n]), nil
}

func readTail(path string, lines int, maxBytes int64) ([]string, os.FileInfo, identity, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, nil, identity{}, err
	}
	if info.IsDir() {
		return nil, nil, identity{}, errors.New("tail path is a directory")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, identity{}, err
	}
	defer file.Close()

	size := info.Size()
	start := int64(0)
	if size > maxBytes {
		start = size - maxBytes
	}
	buf := make([]byte, size-start)
	n, err := file.ReadAt(buf, start)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, nil, identity{}, err
	}
	text := sanitizeUTF8(buf[:n])
	if start > 0 {
		if index := strings.IndexByte(text, '\n'); index >= 0 {
			text = text[index+1:]
		} else {
			text = ""
		}
	}
	physical := strings.Split(text, "\n")
	if len(physical) > 0 && physical[len(physical)-1] == "" {
		physical = physical[:len(physical)-1]
	}
	if len(physical) > lines {
		physical = physical[len(physical)-lines:]
	}
	return physical, info, fileIdentity(info), nil
}

func splitCompleteLines(partial *[]byte, buf []byte) []string {
	combined := make([]byte, 0, len(*partial)+len(buf))
	combined = append(combined, (*partial)...)
	combined = append(combined, buf...)
	lines := []string{}
	start := 0
	for index, value := range combined {
		if value != '\n' {
			continue
		}
		line := combined[start:index]
		lines = append(lines, sanitizeUTF8(line))
		start = index + 1
	}
	if start < len(combined) {
		*partial = append((*partial)[:0], combined[start:]...)
	} else {
		*partial = nil
	}
	return lines
}

func sanitizeUTF8(buf []byte) string {
	if utf8.Valid(buf) {
		return string(buf)
	}
	return strings.ToValidUTF8(string(buf), "\uFFFD")
}

func clampLines(lines int) int {
	if lines <= 0 {
		return DefaultSnapshotLines
	}
	if lines > MaxSnapshotLines {
		return MaxSnapshotLines
	}
	return lines
}

type identity struct {
	dev uint64
	ino uint64
}

func fileIdentity(info os.FileInfo) identity {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return identity{}
	}
	return identity{dev: uint64(stat.Dev), ino: uint64(stat.Ino)}
}
