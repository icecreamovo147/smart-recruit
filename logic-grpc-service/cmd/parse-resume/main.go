package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"logic-grpc-service/resumeparser"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: go run ./cmd/parse-resume /path/to/resume.{pdf,docx}\n")
		os.Exit(2)
	}

	path := os.Args[1]
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	parser, err := resumeparser.DefaultRegistry.GetParser(ext)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unsupported format: %s (supported: pdf, docx)\n", ext)
		os.Exit(2)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read file: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	text, err := parser.ExtractText(ctx, data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse %s: %v\n", ext, err)
		os.Exit(1)
	}

	fmt.Printf("Parsed %s: %s\n", strings.ToUpper(ext), path)
	fmt.Printf("Extracted characters: %d\n", len([]rune(text)))
	fmt.Println("----- BEGIN TEXT -----")
	fmt.Println(text)
	fmt.Println("----- END TEXT -----")

	analysisText, stats := resumeparser.PrepareForAnalysis(text)
	fmt.Printf("Analysis text characters: %d\n", stats.CleanedChars)
	fmt.Printf("Removed noisy lines: %d\n", stats.RemovedLines)
	fmt.Println("----- BEGIN ANALYSIS TEXT -----")
	fmt.Println(analysisText)
	fmt.Println("----- END ANALYSIS TEXT -----")
}
