package manager

import (
	"os"
	"path/filepath"
)

// FindGollamaUnit returns the path to the installed gollama systemd unit and
// whether it is a user unit. It checks the system unit first, then the user
// unit. found is false when no unit is installed. This is the single source of
// truth for unit location, shared by the `restart` CLI command (main.go) and
// the server's restart handler (P5-T6 dedup).
func FindGollamaUnit() (path string, userUnit bool, found bool) {
	const systemUnit = "/etc/systemd/system/gollama.service"
	if _, err := os.Stat(systemUnit); err == nil {
		return systemUnit, false, true
	}
	if home, err := os.UserHomeDir(); err == nil {
		if p := filepath.Join(home, ".config", "systemd", "user", "gollama.service"); serviceFileExists(p) {
			return p, true, true
		}
	}
	return "", false, false
}

func serviceFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
