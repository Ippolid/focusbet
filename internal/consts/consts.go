package consts

import (
	"os"
	"path/filepath"
)

func GetStorageDir() string {
	dir, _ := os.UserConfigDir()
	dbDir := filepath.Join(dir, "focusbet")

	return dbDir
}
