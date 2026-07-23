package tailer

import (
	"context"
	"path/filepath"
	"time"

	"smart-recruit/dev-log-viewer/internal/catalog"
	"smart-recruit/dev-log-viewer/internal/parser"
	"smart-recruit/dev-log-viewer/internal/stream"
)

const DefaultPollInterval = 250 * time.Millisecond

type Coordinator struct {
	root     string
	hub      *stream.Hub
	parser   parser.Parser
	tailers  map[string]*FileTailer
	services map[string]catalog.ServiceDefinition
	interval time.Duration
}

func NewCoordinator(root string, definitions []catalog.ServiceDefinition, hub *stream.Hub) *Coordinator {
	tailers := map[string]*FileTailer{}
	services := map[string]catalog.ServiceDefinition{}
	for _, definition := range definitions {
		path := filepath.Join(root, filepath.FromSlash(definition.LogFile))
		tailers[definition.ID] = NewFileTailer(definition.ID, path)
		services[definition.ID] = definition
	}
	return &Coordinator{
		root:     root,
		hub:      hub,
		parser:   parser.New(),
		tailers:  tailers,
		services: services,
		interval: DefaultPollInterval,
	}
}

func (c *Coordinator) Start(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.PollOnce(time.Now().UTC())
		}
	}
}

func (c *Coordinator) PollOnce(observedAt time.Time) {
	for service, tailer := range c.tailers {
		for _, event := range tailer.Poll() {
			switch event.Type {
			case EventReset:
				c.hub.Publish(stream.Envelope{Type: stream.TypeReset, Service: service, Recoverable: false, Payload: map[string]uint64{"generation": event.Generation}})
			case EventLines:
				records := c.parser.ParseLines(service, event.Generation, event.Lines, observedAt)
				for _, record := range records {
					c.hub.Publish(stream.Envelope{Type: stream.TypeLog, Service: service, Payload: record})
				}
			}
		}
	}
}

func (c *Coordinator) Snapshot(serviceIDs []string, tailLines int, observedAt time.Time) []stream.Envelope {
	ids := serviceIDs
	if len(ids) == 0 {
		ids = make([]string, 0, len(c.services))
		for _, definition := range catalog.DefaultDefinitions() {
			if _, ok := c.services[definition.ID]; ok {
				ids = append(ids, definition.ID)
			}
		}
	}
	envelopes := []stream.Envelope{}
	for _, service := range ids {
		tailer, ok := c.tailers[service]
		if !ok {
			continue
		}
		event := tailer.Snapshot(tailLines)
		if event.Err != nil {
			envelopes = append(envelopes, stream.Envelope{Type: stream.TypeReset, Service: service, Recoverable: false})
			continue
		}
		records := c.parser.ParseLines(service, event.Generation, event.Lines, observedAt)
		for _, record := range records {
			envelopes = append(envelopes, stream.Envelope{Type: stream.TypeLog, Service: service, Payload: record})
		}
	}
	return envelopes
}
