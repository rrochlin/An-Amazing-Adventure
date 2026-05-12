package dialogue

import (
	"encoding/csv"
	"fmt"
	"io"
	"io/fs"
	"strings"
)

// OpeningLine returns the first authored string row for a dialogue asset.
// This is intentionally minimal: enough to surface authored dialogue integration
// before the full Yarn runtime is wired into ws-chat.
func OpeningLine(fsys fs.FS, stringsPath string) (string, error) {
	if fsys == nil {
		return "", fmt.Errorf("dialogue source fs is not set")
	}
	if stringsPath == "" {
		return "", fmt.Errorf("dialogue strings path is required")
	}
	file, err := fsys.Open(stringsPath)
	if err != nil {
		return "", fmt.Errorf("open dialogue strings %q: %w", stringsPath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	rows, err := reader.ReadAll()
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read dialogue strings %q: %w", stringsPath, err)
	}
	for i, row := range rows {
		if len(row) < 2 {
			continue
		}
		if i == 0 && strings.EqualFold(strings.TrimSpace(row[0]), "id") {
			continue
		}
		text := strings.TrimSpace(row[1])
		if text != "" {
			return text, nil
		}
	}
	return "", nil
}
