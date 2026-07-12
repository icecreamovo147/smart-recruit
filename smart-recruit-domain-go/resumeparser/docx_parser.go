package resumeparser

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// DOCXParser extracts text from DOCX resume files.
// DOCX is a ZIP archive containing XML; text lives in word/document.xml inside <w:t> elements.
type DOCXParser struct{}

func (p *DOCXParser) SupportedExtensions() []string {
	return []string{"docx"}
}

func (p *DOCXParser) ExtractText(ctx context.Context, data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("docx data is empty")
	}
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("docx: failed to open zip: %w", err)
	}

	docFile, err := openZipFile(zipReader, "word/document.xml")
	if err != nil {
		return "", fmt.Errorf("docx: %w", err)
	}
	defer docFile.Close()

	docBytes, err := io.ReadAll(docFile)
	if err != nil {
		return "", fmt.Errorf("docx: failed to read document.xml: %w", err)
	}

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	text, err := extractTextFromDocxXML(docBytes)
	if err != nil {
		return "", fmt.Errorf("docx: failed to parse xml: %w", err)
	}

	cleaned := limitText(cleanText(text))
	if strings.TrimSpace(cleaned) == "" {
		return "", fmt.Errorf("docx does not contain extractable text")
	}
	return cleaned, nil
}

func openZipFile(r *zip.Reader, name string) (io.ReadCloser, error) {
	for _, f := range r.File {
		if f.Name == name {
			return f.Open()
		}
	}
	return nil, fmt.Errorf("file %q not found in archive", name)
}

// extractTextFromDocxXML parses the simplified Office Open XML and extracts all <w:t> text,
// inserting newlines between paragraphs (<w:p>).
func extractTextFromDocxXML(data []byte) (string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var builder strings.Builder
	inParagraph := false
	hadTextInParagraph := false

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		switch se := tok.(type) {
		case xml.StartElement:
			if se.Name.Local == "p" {
				inParagraph = true
				hadTextInParagraph = false
			}
		case xml.EndElement:
			if se.Name.Local == "p" {
				if inParagraph && hadTextInParagraph {
					builder.WriteString("\n")
				}
				inParagraph = false
			}
		case xml.CharData:
			if inParagraph {
				text := strings.TrimSpace(string(se))
				if text != "" {
					builder.WriteString(text)
					hadTextInParagraph = true
				}
			}
		}
	}

	return builder.String(), nil
}
