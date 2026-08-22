package workspace

import (
	"encoding/json"
	"io/fs"
	"path/filepath"
	"strings"
)

func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func truncate(value string, maxBytes int) (string, bool) {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value, false
	}
	return value[:maxBytes], true
}

func relativeWorkingDirOrDot(path string) string {
	if path == "" {
		return "."
	}
	return path
}

func mergeMetadata(base map[string]any, extra map[string]any) map[string]any {
	result := make(map[string]any, len(base)+len(extra))
	for key, value := range base {
		result[key] = value
	}
	for key, value := range extra {
		result[key] = value
	}
	return result
}

func trimStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func scanQuota(root string) (quotaStats, error) {
	var stats quotaStats
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		stats.FileCount++
		stats.UsedDiskBytes += info.Size()
		return nil
	})
	return stats, err
}

func mustJSON(value any) json.RawMessage {
	raw, _ := json.Marshal(value)
	return raw
}

func stringPtr(value string) *string {
	return &value
}
