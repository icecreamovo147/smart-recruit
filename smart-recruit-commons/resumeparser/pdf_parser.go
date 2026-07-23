package resumeparser

import (
	"context"
	"fmt"
	"strings"

	"github.com/gen2brain/go-fitz"
)

// PDFParser extracts text from PDF resume files using MuPDF (via go-fitz).
type PDFParser struct{}

func (p *PDFParser) SupportedExtensions() []string {
	return []string{"pdf"}
}

func (p *PDFParser) ExtractText(ctx context.Context, data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("pdf data is empty")
	}
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	doc, err := fitz.NewFromMemory(data)
	if err != nil {
		return "", err
	}
	defer doc.Close()

	var builder strings.Builder
	for pageNumber := 0; pageNumber < doc.NumPage(); pageNumber++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		pageText, err := doc.Text(pageNumber)
		if err != nil {
			return "", err
		}
		pageText = strings.TrimSpace(pageText)
		if pageText == "" {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString(pageText)
	}

	text := limitText(cleanText(builder.String()))
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("pdf does not contain extractable text")
	}
	if !isDocumentCoherent(text) {
		return "", fmt.Errorf("pdf text appears garbled (broken font encoding), OCR may be required")
	}
	return text, nil
}
