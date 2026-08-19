package services

// ExposedSanitizeGeminiLatex exposes the internal sanitizeGeminiLatex function for unit testing.
func ExposedSanitizeGeminiLatex(rawOutput string) string {
	return sanitizeGeminiLatex(rawOutput)
}
