// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/juste-un-gars/anemone/internal/config"
	"github.com/juste-un-gars/anemone/internal/database"
	"github.com/juste-un-gars/anemone/internal/logger"
	"github.com/juste-un-gars/anemone/internal/scheduler"
	"github.com/juste-un-gars/anemone/internal/serverbackup"
	"github.com/juste-un-gars/anemone/internal/setup"
	syncpkg "github.com/juste-un-gars/anemone/internal/sync"
	"github.com/juste-un-gars/anemone/internal/sysconfig"
	"github.com/juste-un-gars/anemone/internal/rclone"
	"github.com/juste-un-gars/anemone/internal/tls"
	"github.com/juste-un-gars/anemone/internal/trash"
	"github.com/juste-un-gars/anemone/internal/updater"
	"github.com/juste-un-gars/anemone/internal/usbbackup"
	"github.com/juste-un-gars/anemone/internal/usermanifest"
	"github.com/juste-un-gars/anemone/internal/web"
	wgpkg "github.com/juste-un-gars/anemone/internal/wireguard"
)

func main() {
	// Load configuration first
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger early (with env var level or default)
	// Will be updated with DB level after database init
	logLevel := logger.ParseLevel(cfg.LogLevel)
	if cfg.LogLevel == "" {
		logLevel = logger.DefaultLevel // WARN
	}
	if err := logger.Init(&logger.Config{
		Level:  logLevel,
		LogDir: cfg.LogDir,
	}); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Close()

	logger.Info("Starting Anemone NAS...")

	// Check if setup is needed
	if setup.IsSetupNeeded(cfg.DataDir) {
		logger.Info("Setup mode detected - starting setup wizard...")
		runSetupMode(cfg)
		return
	}

	// Validate directories exist and are writable
	if warnings := cfg.ValidateDirs(); len(warnings) > 0 {
		for _, w := range warnings {
			logger.Warn("Directory warning", "message", w)
		}
	}

	// Initialize database
	db, err := database.Init(cfg.DatabasePath)
	if err != nil {
		logger.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Run migrations
	if err := database.Migrate(db); err != nil {
		logger.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Update log level from DB (unless overridden by env var)
	if cfg.LogLevel == "" {
		if dbLevel, err := sysconfig.GetLogLevel(db); err == nil {
			logger.SetLevel(logger.ParseLevel(dbLevel))
			logger.Info("Log level set from database", "level", dbLevel)
		}
	}

	// Sync current version in database with code version
	if err := updater.SyncVersionWithDB(db); err != nil {
		logger.Warn("Failed to sync version with DB", "error", err)
	}

	// Cleanup zombie syncs (syncs stuck in "running" state)
	if err := syncpkg.CleanupZombieSyncs(db); err != nil {
		logger.Warn("Failed to cleanup zombie syncs", "error", err)
	}

	// Start automatic synchronization scheduler
	scheduler.Start(db)

	// Start automatic server backup scheduler (daily at 4 AM)
	serverbackup.StartScheduler(db, cfg.DataDir)

	// Start automatic USB backup scheduler
	usbbackup.StartScheduler(db, cfg.DataDir)

	// Start automatic rclone (cloud) backup scheduler
	rclone.StartScheduler(db, cfg.DataDir)

	// Auto-connect WireGuard VPN if configured
	if err := wgpkg.AutoConnect(db); err != nil {
		logger.Warn("WireGuard auto-connect failed", "error", err)
	}

	// Start automatic trash cleanup scheduler (daily at 3 AM)
	trash.StartCleanupScheduler(db, func() (int, error) {
		return sysconfig.GetTrashRetentionDays(db)
	})

	// Start automatic update checker (daily)
	updater.StartUpdateChecker(db)

	// Start user manifest watcher (real-time updates via inotify)
	// Monitors share directories and regenerates manifests when files change
	manifestWatcher, err := usermanifest.NewWatcher(db)
	if err != nil {
		logger.Warn("Failed to create manifest watcher, falling back to scheduled generation", "error", err)
	} else {
		if err := manifestWatcher.Start(); err != nil {
			logger.Warn("Failed to start manifest watcher", "error", err)
		}
		defer manifestWatcher.Stop()
	}

	// Start user manifest scheduler as backup (every 30 minutes)
	// Catches any changes that might be missed by the watcher
	usermanifest.StartScheduler(db, cfg.SharesDir, 30)

	// Initialize web server
	router := web.NewRouter(db, cfg)

	// WaitGroup to wait for all servers
	var wg sync.WaitGroup

	// Start HTTPS server if enabled
	if cfg.EnableHTTPS {
		wg.Add(1)
		go func() {
			defer wg.Done()
			startHTTPSServer(cfg, router)
		}()
	}

	// Start HTTP server if enabled (disabled by default for security)
	if cfg.EnableHTTP {
		wg.Add(1)
		go func() {
			defer wg.Done()
			startHTTPServer(cfg, router)
		}()
	}

	// If neither is enabled, this shouldn't happen due to config validation
	if !cfg.EnableHTTPS && !cfg.EnableHTTP {
		logger.Error("No server protocol enabled. Set ENABLE_HTTPS=true or ENABLE_HTTP=true")
		os.Exit(1)
	}

	// Wait for all servers
	wg.Wait()
}

func startHTTPSServer(cfg *config.Config, router http.Handler) {
	// Generate or load TLS certificate
	certCfg := &tls.CertConfig{
		CertPath: cfg.TLSCertPath,
		KeyPath:  cfg.TLSKeyPath,
		DataDir:  cfg.DataDir,
	}

	if err := tls.GenerateOrLoadCertificate(certCfg); err != nil {
		logger.Error("Failed to setup TLS certificate", "error", err)
		os.Exit(1)
	}

	addr := fmt.Sprintf(":%s", cfg.HTTPSPort)
	logger.Info("HTTPS server listening", "address", fmt.Sprintf("https://localhost%s", addr))
	logger.Info("Self-signed certificate warning is normal for local/private use")

	if err := http.ListenAndServeTLS(addr, cfg.TLSCertPath, cfg.TLSKeyPath, router); err != nil {
		logger.Error("HTTPS server failed", "error", err)
		os.Exit(1)
	}
}

func startHTTPServer(cfg *config.Config, router http.Handler) {
	addr := fmt.Sprintf(":%s", cfg.Port)
	logger.Warn("HTTP server listening (insecure)", "address", fmt.Sprintf("http://localhost%s", addr))
	logger.Warn("HTTP transmits credentials in clear text. Consider ENABLE_HTTPS=true")

	if err := http.ListenAndServe(addr, router); err != nil {
		logger.Error("HTTP server failed", "error", err)
		os.Exit(1)
	}
}

// runSetupMode runs the setup wizard server
func runSetupMode(cfg *config.Config) {
	// Create setup wizard router
	router := web.NewSetupRouter(cfg)

	// WaitGroup to wait for all servers
	var wg sync.WaitGroup

	// Start HTTPS server if enabled
	if cfg.EnableHTTPS {
		wg.Add(1)
		go func() {
			defer wg.Done()
			startHTTPSServer(cfg, router)
		}()
	}

	// Start HTTP server if enabled
	if cfg.EnableHTTP {
		wg.Add(1)
		go func() {
			defer wg.Done()
			startHTTPServer(cfg, router)
		}()
	}

	// If neither is enabled, default to HTTP for setup
	if !cfg.EnableHTTPS && !cfg.EnableHTTP {
		logger.Warn("No server protocol enabled, defaulting to HTTP for setup wizard")
		startHTTPServer(cfg, router)
		return
	}

	// Wait for all servers
	wg.Wait()
}
