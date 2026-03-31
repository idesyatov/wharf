package ui

import (
	"encoding/base64"
	"fmt"
	"os"
)

func CopyToClipboard(text string) {
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	fmt.Fprintf(os.Stderr, "\033]52;c;%s\a", encoded)
}
