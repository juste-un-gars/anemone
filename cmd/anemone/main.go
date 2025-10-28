// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/juste-un-gars/anemone/internal/config"
	"github.com/juste-un-gars/anemone/internal/database"
	"github.com/juste-un-gars/anemone/internal/tls"
	"github.com/juste-un-gars/anemone/internal/web"
)

func main() {
	log.Println("🪸 Starting Anemone NAS...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := database.Init(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

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
		log.Fatal("❌ No server protocol enabled. Set ENABLE_HTTPS=true or ENABLE_HTTP=true")
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
		log.Fatalf("Failed to setup TLS certificate: %v", err)
	}

	addr := fmt.Sprintf(":%s", cfg.HTTPSPort)
	log.Printf("🔒 HTTPS server listening on https://localhost%s", addr)
	log.Printf("   ⚠️  If using a self-signed certificate, your browser will show a security warning")
	log.Printf("   ✓  This is normal and safe for local/private use")

	if err := http.ListenAndServeTLS(addr, cfg.TLSCertPath, cfg.TLSKeyPath, router); err != nil {
		log.Fatalf("HTTPS server failed: %v", err)
	}
}

func startHTTPServer(cfg *config.Config, router http.Handler) {
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("⚠️  HTTP server listening on http://localhost%s", addr)
	log.Printf("   ⚠️  WARNING: HTTP is not secure. Credentials and data are transmitted in clear text!")
	log.Printf("   ✓  Consider using HTTPS instead by setting ENABLE_HTTPS=true")

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}
