package resumeparser

import (
	"context"
	"fmt"
	"strings"
)

// ResumeParser defines the strategy interface for extracting text from resume files.
// Each supported file format provides its own implementation.
type ResumeParser interface {
	ExtractText(ctx context.Context, data []byte) (string, error)
	SupportedExtensions() []string
}

// Registry maps file extensions to their corresponding parsers.
// Use NewRegistry() to get a pre-configured instance with all supported parsers.
type Registry struct {
	parsers map[string]ResumeParser
}

// NewRegistry creates a Registry with all supported resume parsers registered.
// To add support for a new format, create a new ResumeParser implementation
// and register it here.
func NewRegistry() *Registry {
	r := &Registry{parsers: make(map[string]ResumeParser)}
	r.register(&PDFParser{})
	r.register(&DOCXParser{})
	return r
}

func (r *Registry) register(p ResumeParser) {
	for _, ext := range p.SupportedExtensions() {
		r.parsers[strings.ToLower(ext)] = p
	}
}

// GetParser returns the parser for the given file extension, or an error if unsupported.
func (r *Registry) GetParser(extension string) (ResumeParser, error) {
	ext := strings.TrimPrefix(strings.ToLower(extension), ".")
	p, ok := r.parsers[ext]
	if !ok {
		return nil, fmt.Errorf("unsupported resume format: %s", extension)
	}
	return p, nil
}

// IsParsable returns true if the extension has a registered parser that can extract text.
func (r *Registry) IsParsable(extension string) bool {
	ext := strings.TrimPrefix(strings.ToLower(extension), ".")
	_, ok := r.parsers[ext]
	return ok
}

// DefaultRegistry is the global parser registry pre-configured with all supported formats.
var DefaultRegistry = NewRegistry()
