package resumeparser

import (
	"bytes"
	"strings"
	"unicode"
)

const maxResumeTextLength = 6000

// AnalysisTextStats holds metrics from the text cleaning pipeline.
type AnalysisTextStats struct {
	OriginalChars int
	CleanedChars  int
	KeptLines     int
	RemovedLines  int
}

// Magic bytes for supported resume formats.
var (
	magicPDF = []byte("%PDF-")
	magicZIP = []byte("PK\x03\x04") // DOCX (ZIP-based)
)

// ValidateMagicBytes checks whether the file header matches the claimed extension.
// ext should be the lowercase file extension (without dot).
func ValidateMagicBytes(ext string, header []byte) bool {
	switch ext {
	case "pdf":
		return bytes.HasPrefix(header, magicPDF)
	case "docx":
		return bytes.HasPrefix(header, magicZIP)
	default:
		return false
	}
}

func PrepareForAnalysis(text string) (string, AnalysisTextStats) {
	stats := AnalysisTextStats{OriginalChars: len([]rune(text))}
	lines := strings.Split(text, "\n")
	cleaned := make([]string, 0, len(lines))
	seen := make(map[string]int)
	blank := false
	for _, line := range lines {
		line = normalizeResumeLine(line)
		if line == "" {
			if len(cleaned) > 0 && !blank {
				cleaned = append(cleaned, "")
				blank = true
			}
			continue
		}
		if !isUsefulResumeLine(line) {
			stats.RemovedLines++
			continue
		}
		key := strings.ToLower(strings.Join(strings.Fields(line), " "))
		if seen[key] >= 2 {
			stats.RemovedLines++
			continue
		}
		seen[key]++
		cleaned = append(cleaned, line)
		stats.KeptLines++
		blank = false
	}
	result := limitText(cleanText(strings.Join(cleaned, "\n")))
	stats.CleanedChars = len([]rune(result))
	return result, stats
}

func IsAnalysisTextUseful(text string, stats AnalysisTextStats) bool {
	if strings.TrimSpace(text) == "" {
		return false
	}
	if stats.CleanedChars < 120 || stats.KeptLines < 3 {
		return false
	}
	if stats.RemovedLines >= 20 && stats.CleanedChars < 800 {
		return false
	}
	if stats.OriginalChars > 0 && float64(stats.CleanedChars)/float64(stats.OriginalChars) < 0.2 {
		return false
	}
	return isDocumentCoherent(text)
}

func cleanText(text string) string {
	lines := strings.Split(text, "\n")
	cleaned := make([]string, 0, len(lines))
	blank := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if !blank {
				cleaned = append(cleaned, "")
				blank = true
			}
			continue
		}
		cleaned = append(cleaned, line)
		blank = false
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

func limitText(text string) string {
	runes := []rune(text)
	if len(runes) <= maxResumeTextLength {
		return text
	}
	return string(runes[:maxResumeTextLength]) + "\n...（简历文本过长，已截断）"
}

func normalizeResumeLine(line string) string {
	line = strings.Map(func(r rune) rune {
		switch r {
		case '\u00a0', '\u2002', '\u2003', '\u2009':
			return ' '
		case '\u200b', '\ufeff', unicode.ReplacementChar:
			return -1
		default:
			if r == '\t' {
				return ' '
			}
			if isPrivateUse(r) {
				return -1
			}
			if unicode.IsControl(r) {
				return -1
			}
			return r
		}
	}, line)
	return strings.TrimSpace(strings.Join(strings.Fields(line), " "))
}

func isUsefulResumeLine(line string) bool {
	runes := []rune(line)
	if len(runes) == 0 {
		return false
	}
	useful := 0
	core := 0
	bad := 0
	for _, r := range runes {
		switch {
		case r == unicode.ReplacementChar:
			bad += 2
		case isPrivateUse(r):
			bad += 2
		case isCJK(r), unicode.IsLetter(r), unicode.IsDigit(r):
			useful++
			core++
		case isCommonResumePunctuation(r):
			useful++
		case unicode.IsSpace(r):
			useful++
		case unicode.IsSymbol(r):
			bad++
		default:
			if unicode.IsPunct(r) {
				useful++
			} else {
				bad++
			}
		}
	}
	total := len(runes)
	if bad > 0 && bad*3 >= total {
		return false
	}
	if core == 0 {
		return false
	}
	if total <= 4 {
		return useful >= total-bad
	}
	return float64(useful)/float64(total) >= 0.68 && float64(core)/float64(total) >= 0.25
}

func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		(r >= 0x3040 && r <= 0x30ff) ||
		(r >= 0xac00 && r <= 0xd7af)
}

func isPrivateUse(r rune) bool {
	return (r >= 0xe000 && r <= 0xf8ff) ||
		(r >= 0xf0000 && r <= 0xffffd) ||
		(r >= 0x100000 && r <= 0x10fffd)
}

// isDocumentCoherent detects garbled output from PDFs with broken font encodings.
// Such PDFs produce valid-looking ASCII characters (e.g. "Y3", "NMs", "979;G7@")
// that pass line-level filters but make no semantic sense.
// A coherent Chinese resume has CJK ratio >= 8%; a coherent English resume has avg word length >= 4.0.
func isDocumentCoherent(text string) bool {
	runes := []rune(text)
	if len(runes) < 60 {
		return true
	}
	totalNonSpace := 0
	cjkCount := 0
	for _, r := range runes {
		if !unicode.IsSpace(r) {
			totalNonSpace++
			if isCJK(r) {
				cjkCount++
			}
		}
	}
	if totalNonSpace == 0 {
		return false
	}
	if float64(cjkCount)/float64(totalNonSpace) >= 0.08 {
		return true
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return false
	}
	totalLen := 0
	for _, w := range words {
		totalLen += len([]rune(w))
	}
	return float64(totalLen)/float64(len(words)) >= 4.0
}

func isCommonResumePunctuation(r rune) bool {
	return strings.ContainsRune(" ,.;:!?()[]{}<>-/\\+_=*@#%&|\"'`~，。；：！？（）【】《》、-—·•￥", r)
}
