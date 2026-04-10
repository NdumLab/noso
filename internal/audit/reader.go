package audit

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/NdumLab/noso/pkg/models"
)

func ReadAll(path string) ([]models.AuditRecord, error) {
	if path == "" {
		return nil, nil
	}

	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var records []models.AuditRecord
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record models.AuditRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			// Skip malformed lines rather than aborting — a corrupt or
			// truncated record at the end of the log should not prevent
			// all history from being read.
			continue
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func Filter(records []models.AuditRecord, query string, limit int) []models.AuditRecord {
	query = strings.ToLower(strings.TrimSpace(query))
	var filtered []models.AuditRecord
	for i := len(records) - 1; i >= 0; i-- {
		record := records[i]
		if query != "" {
			blob := strings.ToLower(record.Query + "\n" + record.Response.IntentID + "\n" + record.Response.Command)
			if !strings.Contains(blob, query) {
				continue
			}
		}
		filtered = append(filtered, record)
		if limit > 0 && len(filtered) >= limit {
			break
		}
	}
	return filtered
}
