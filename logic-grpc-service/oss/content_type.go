package oss

import "strings"

// ContentTypeFromFileType maps a logical file_type (e.g. "pdf", "docx") to a
// canonical MIME type used when signing PUT URLs. Returns
// "application/octet-stream" for unknown types so the signed Content-Type
// always has a deterministic value the frontend can match.
func ContentTypeFromFileType(fileType string) string {
	switch strings.ToLower(strings.TrimSpace(fileType)) {
	case "pdf":
		return "application/pdf"
	case "docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case "doc":
		return "application/msword"
	default:
		return "application/octet-stream"
	}
}
