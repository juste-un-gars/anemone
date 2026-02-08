// Anemone - Multi-user NAS with P2P encrypted synchronization
// Copyright (C) 2025 juste-un-gars
// Licensed under the GNU Affero General Public License v3.0

package web

import (
	"fmt"
	"net/http"
	"os/exec"

	"github.com/juste-un-gars/anemone/internal/auth"
	"github.com/juste-un-gars/anemone/internal/logger"
	"github.com/juste-un-gars/anemone/internal/wireguard"
)

// handleAdminWireGuard displays the WireGuard configuration page.
func (s *Server) handleAdminWireGuard(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	lang := s.getLang(r)

	// Check if WireGuard is installed
	installed := isWireGuardInstalled()

	// Get current configuration
	cfg, err := wireguard.Get(s.db)
	if err != nil {
		logger.Error("Error getting WireGuard config", "error", err)
	}

	// Get detailed VPN status
	var status *wireguard.Status
	if cfg != nil {
		status = wireguard.GetStatus(cfg.Name)
	}

	s.renderWireGuardPageV2(w, session, lang, installed, cfg, status, "", "")
}

// isWireGuardInstalled checks if wg-quick is available.
func isWireGuardInstalled() bool {
	_, err := exec.LookPath("wg-quick")
	return err == nil
}

// handleAdminWireGuardImport handles POST to import a .conf file.
func (s *Server) handleAdminWireGuardImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/wireguard", http.StatusSeeOther)
		return
	}

	configContent := r.FormValue("config")

	// Parse the configuration
	cfg, err := wireguard.ParseConfig(configContent)
	if err != nil {
		logger.Warn("Failed to parse WireGuard config", "error", err)
		s.renderWireGuardPage(w, r, "", "Failed to parse config: "+err.Error())
		return
	}

	// Save to database
	if err := wireguard.Save(s.db, cfg); err != nil {
		logger.Error("Failed to save WireGuard config", "error", err)
		s.renderWireGuardPage(w, r, "", "Failed to save configuration")
		return
	}

	logger.Info("WireGuard configuration imported", "address", cfg.Address, "endpoint", cfg.PeerEndpoint)
	s.renderWireGuardPage(w, r, "Configuration imported successfully", "")
}

// handleAdminWireGuardEdit handles POST to edit the configuration.
func (s *Server) handleAdminWireGuardEdit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/wireguard", http.StatusSeeOther)
		return
	}

	// Get existing config
	cfg, err := wireguard.Get(s.db)
	if err != nil || cfg == nil {
		s.renderWireGuardPage(w, r, "", "No configuration found")
		return
	}

	// Update fields from form
	if name := r.FormValue("name"); name != "" {
		cfg.Name = name
	}
	if address := r.FormValue("address"); address != "" {
		cfg.Address = address
	}
	cfg.DNS = r.FormValue("dns") // Can be empty
	if endpoint := r.FormValue("peer_endpoint"); endpoint != "" {
		cfg.PeerEndpoint = endpoint
	}
	if allowedIPs := r.FormValue("allowed_ips"); allowedIPs != "" {
		cfg.AllowedIPs = allowedIPs
	}
	if keepalive := r.FormValue("persistent_keepalive"); keepalive != "" {
		var k int
		fmt.Sscanf(keepalive, "%d", &k)
		cfg.PersistentKeepalive = k
	}

	// Save updated config
	if err := wireguard.Save(s.db, cfg); err != nil {
		logger.Error("Failed to save WireGuard config", "error", err)
		s.renderWireGuardPage(w, r, "", "Failed to save configuration")
		return
	}

	logger.Info("WireGuard configuration updated", "address", cfg.Address, "endpoint", cfg.PeerEndpoint)
	s.renderWireGuardPage(w, r, "Configuration updated successfully", "")
}

// handleAdminWireGuardDelete handles POST to delete the configuration.
func (s *Server) handleAdminWireGuardDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/wireguard", http.StatusSeeOther)
		return
	}

	// Disconnect if connected
	cfg, _ := wireguard.Get(s.db)
	if cfg != nil && cfg.Enabled {
		disconnectWireGuard(cfg.Name)
	}

	// Delete from database
	if err := wireguard.Delete(s.db); err != nil {
		logger.Error("Failed to delete WireGuard config", "error", err)
		s.renderWireGuardPage(w, r, "", "Failed to delete configuration")
		return
	}

	logger.Info("WireGuard configuration deleted")
	s.renderWireGuardPage(w, r, "Configuration deleted", "")
}

// handleAdminWireGuardOptions handles POST to update options (auto_start).
func (s *Server) handleAdminWireGuardOptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/wireguard", http.StatusSeeOther)
		return
	}

	autoStart := r.FormValue("auto_start") == "1"

	if err := wireguard.SetAutoStart(s.db, autoStart); err != nil {
		logger.Error("Failed to update auto_start", "error", err)
	} else {
		logger.Info("WireGuard auto_start updated", "auto_start", autoStart)
	}

	http.Redirect(w, r, "/admin/wireguard", http.StatusSeeOther)
}

// handleAdminWireGuardConnect handles POST to connect the VPN.
func (s *Server) handleAdminWireGuardConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/wireguard", http.StatusSeeOther)
		return
	}

	cfg, err := wireguard.Get(s.db)
	if err != nil || cfg == nil {
		s.renderWireGuardPage(w, r, "", "No configuration found")
		return
	}

	// Write the config file to /etc/wireguard/
	if err := wireguard.WriteConfFile(cfg); err != nil {
		logger.Error("Failed to write WireGuard config file", "error", err)
		s.renderWireGuardPage(w, r, "", "Failed to write config file: "+err.Error())
		return
	}

	// Start the VPN
	if err := connectWireGuard(cfg.Name); err != nil {
		logger.Error("Failed to connect WireGuard", "error", err)
		s.renderWireGuardPage(w, r, "", "Failed to connect: "+err.Error())
		return
	}

	// Update enabled status in DB
	if err := wireguard.SetEnabled(s.db, true); err != nil {
		logger.Error("Failed to update enabled status", "error", err)
	}

	logger.Info("WireGuard VPN connected", "interface", cfg.Name)
	s.renderWireGuardPage(w, r, "VPN connected", "")
}

// handleAdminWireGuardDisconnect handles POST to disconnect the VPN.
func (s *Server) handleAdminWireGuardDisconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/wireguard", http.StatusSeeOther)
		return
	}

	cfg, err := wireguard.Get(s.db)
	if err != nil || cfg == nil {
		http.Redirect(w, r, "/admin/wireguard", http.StatusSeeOther)
		return
	}

	// Stop the VPN
	if err := disconnectWireGuard(cfg.Name); err != nil {
		logger.Warn("Failed to disconnect WireGuard", "error", err)
		// Continue anyway - interface might already be down
	}

	// Update enabled status in DB
	if err := wireguard.SetEnabled(s.db, false); err != nil {
		logger.Error("Failed to update enabled status", "error", err)
	}

	logger.Info("WireGuard VPN disconnected", "interface", cfg.Name)
	s.renderWireGuardPage(w, r, "VPN disconnected", "")
}

// connectWireGuard starts the WireGuard interface.
func connectWireGuard(name string) error {
	cmd := exec.Command("sudo", "wg-quick", "up", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(output))
	}
	return nil
}

// disconnectWireGuard stops the WireGuard interface.
func disconnectWireGuard(name string) error {
	cmd := exec.Command("sudo", "wg-quick", "down", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(output))
	}
	return nil
}

// renderWireGuardPage is a helper to render the page with success/error messages.
func (s *Server) renderWireGuardPage(w http.ResponseWriter, r *http.Request, success, errorMsg string) {
	session, _ := auth.GetSessionFromContext(r)
	lang := s.getLang(r)
	installed := isWireGuardInstalled()
	cfg, _ := wireguard.Get(s.db)

	var status *wireguard.Status
	if cfg != nil {
		status = wireguard.GetStatus(cfg.Name)
	}

	s.renderWireGuardPageV2(w, session, lang, installed, cfg, status, success, errorMsg)
}

// renderWireGuardPageV2 renders the v2 WireGuard page.
func (s *Server) renderWireGuardPageV2(w http.ResponseWriter, session *auth.Session, lang string, installed bool, cfg *wireguard.Config, status *wireguard.Status, success, errorMsg string) {
	data := struct {
		V2TemplateData
		Installed bool
		Config    *wireguard.Config
		Status    *wireguard.Status
		Success   string
		Error     string
	}{
		V2TemplateData: V2TemplateData{
			Lang:       lang,
			Title:      "WireGuard VPN",
			ActivePage: "wireguard",
			Session:    session,
		},
		Installed: installed,
		Config:    cfg,
		Status:    status,
		Success:   success,
		Error:     errorMsg,
	}

	tmpl := s.loadV2Page("v2_wireguard.html", s.funcMap)
	if err := tmpl.ExecuteTemplate(w, "v2_base", data); err != nil {
		logger.Error("Error rendering WireGuard template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
