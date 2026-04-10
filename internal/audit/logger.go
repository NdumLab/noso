package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/NdumLab/noso/pkg/models"
)

type Logger struct {
	path string
}

func NewLogger(path string) Logger {
	return Logger{path: path}
}

func (l Logger) Append(query string, response models.Response) error {
	if l.path == "" {
		return nil
	}

	// 0o700 dir: only the owning user can list or enter it.
	dir := filepath.Dir(l.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	// MkdirAll does not chmod existing directories; enforce permissions explicitly.
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}

	// 0o600 file: only the owning user can read or write audit records.
	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	// Exclusive lock so concurrent CLI invocations don't interleave records.
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN) //nolint:errcheck

	record := models.AuditRecord{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Query:     query,
		Response:  response,
	}
	return json.NewEncoder(f).Encode(record)
}
