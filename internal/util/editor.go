package util

import "os"

// DetectEditor returns the user's preferred editor.
// Priority: VISUAL > EDITOR > vi fallback.
func DetectEditor() string {
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	return "vi"
}
