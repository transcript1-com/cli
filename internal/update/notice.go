package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	checkFile     = "update-check.json"
	checkInterval = 24 * time.Hour

	msgAvailable = "transcript1 %s is available. Run transcript1 update."
)

type checkRecord struct {
	CheckedAt string `json:"checked_at"`
}

// record the time first so a failed check also waits a day
func Notice(ctx context.Context, hc *http.Client, base, current, cacheDir string, now time.Time) string {
	pub, err := PublicKey()
	if err != nil {
		return ""
	}

	if !IsVersion(current) {
		return ""
	}

	if isCheckedRecently(cacheDir, now) {
		return ""
	}

	recordCheck(cacheDir, now)
	m, err := Fetch(ctx, hc, base, pub)
	if err != nil {
		return ""
	}

	if !Newer(m.Version, current) {
		return ""
	}

	return fmt.Sprintf(msgAvailable, m.Version)
}

func isCheckedRecently(cacheDir string, now time.Time) bool {
	data, err := os.ReadFile(filepath.Join(cacheDir, checkFile))
	if err != nil {
		return false
	}

	var rec checkRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return false
	}

	at, err := time.Parse(time.RFC3339, rec.CheckedAt)
	if err != nil {
		return false
	}

	age := now.Sub(at)
	if age < 0 {
		return false
	}

	return age < checkInterval
}

func recordCheck(cacheDir string, now time.Time) {
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return
	}

	data, _ := json.Marshal(checkRecord{CheckedAt: now.UTC().Format(time.RFC3339)})
	os.WriteFile(filepath.Join(cacheDir, checkFile), append(data, '\n'), 0o600)
}
