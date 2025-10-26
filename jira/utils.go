package jira

import (
	"strings"
)

// ExtractDescription extracts text from Jira description field (handles both string and ADF format)
func ExtractDescription(desc any) string {
	if desc == nil {
		return ""
	}

	// If it's already a string, return it
	if str, ok := desc.(string); ok {
		return str
	}

	// If it's an ADF object, extract text from content
	if adf, ok := desc.(map[string]any); ok {
		content, ok := adf["content"].([]any)
		if !ok {
			return ""
		}

		var text strings.Builder
		for _, item := range content {
			if itemMap, ok := item.(map[string]any); ok {
				extractTextFromADF(itemMap, &text)
			}
		}
		return strings.TrimSpace(text.String())
	}

	return ""
}

// extractTextFromADF recursively extracts text from ADF nodes
func extractTextFromADF(node map[string]any, builder *strings.Builder) {
	if text, ok := node["text"].(string); ok {
		builder.WriteString(text)
	}

	if content, ok := node["content"].([]any); ok {
		for _, item := range content {
			if itemMap, ok := item.(map[string]any); ok {
				extractTextFromADF(itemMap, builder)
			}
		}
		// Add newline after paragraph-like nodes
		if nodeType, ok := node["type"].(string); ok {
			if nodeType == "paragraph" || nodeType == "heading" {
				builder.WriteString("\n")
			}
		}
	}
}
