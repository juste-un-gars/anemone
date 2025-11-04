// Anemone SMB Config Generator - Force SMB config regeneration
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/juste-un-gars/anemone/internal/smb"
)

func main() {
	// Get data directory
	dataDir := os.Getenv("ANEMONE_DATA_DIR")
	if dataDir == "" {
		dataDir = "/srv/anemone"
	}

	dbPath := filepath.Join(dataDir, "db", "anemone.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		fmt.Printf("❌ Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Use system-wide dfree wrapper
	dfreePath := "/usr/local/bin/anemone-dfree-wrapper.sh"

	smbCfg := &smb.Config{
		ConfigPath: filepath.Join(dataDir, "smb", "smb.conf"),
		WorkGroup:  "ANEMONE",
		ServerName: "Anemone NAS",
		SharesDir:  filepath.Join(dataDir, "shares"),
		DfreePath:  dfreePath,
	}

	fmt.Println("🔧 Regenerating SMB configuration...")
	if err := smb.GenerateConfig(db, smbCfg); err != nil {
		fmt.Printf("❌ Failed to generate config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ SMB config regenerated successfully")
	fmt.Printf("   Config: %s\n", smbCfg.ConfigPath)
	fmt.Printf("   Copied to: /etc/samba/smb.conf\n")

	fmt.Println("\n🔄 Reloading Samba...")
	if err := smb.ReloadConfig(); err != nil {
		fmt.Printf("⚠️  Failed to reload Samba: %v\n", err)
		fmt.Println("   Run manually: sudo systemctl reload smbd")
		os.Exit(1)
	}

	fmt.Println("✅ Samba reloaded successfully")
}
