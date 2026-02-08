// v2preview - Standalone preview server for the v2 UI prototype.
// Serves templates with mock data, no database or setup required.
// Usage: go run cmd/v2preview/main.go
package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
)

// Mock data structs matching what the v2 templates expect.

type Session struct {
	Username string
	IsAdmin  bool
}

type DashboardStats struct {
	UserCount       int
	StorageUsed     string
	PeerStorageUsed string
	StorageQuota    string
	StoragePercent  int
	LastBackup      string
	PeerCount       int
	TrashCount      int
}

type Activity struct {
	Description string
	Time        string
	Status      string
}

type USBBackup struct {
	ID         int
	Name       string
	DevicePath string
	IsMounted  bool
	LastSync   string
	LastStatus string
}

type Drive struct {
	Name string
	Size string
}

type RcloneConfig struct {
	ID         int
	Name       string
	Host       string
	RemotePath string
	Enabled    bool
	LastSync   string
}

type SyncEntry struct {
	PeerName    string
	Username    string
	Status      string
	FilesSynced int
	BytesSynced string
	CompletedAt string
}

type IncomingBackup struct {
	SourceServer       string
	UserID             int
	ShareName          string
	FileCount          int
	TotalSizeFormatted string
	Path               string
}

type ServerBackup struct {
	Name          string
	SizeFormatted string
	Date          string
}

type DashboardData struct {
	Lang           string
	Title          string
	ActivePage     string
	Session        *Session
	Stats          *DashboardStats
	RecentActivity []Activity
}

type BackupsData struct {
	Lang       string
	Title      string
	ActivePage string
	Session    *Session

	USBBackups    []USBBackup
	USBDrives     []Drive
	RcloneConfigs []RcloneConfig
	SSHKeyExists  bool

	SyncEnabled  bool
	SyncInterval string
	RecentSyncs  []SyncEntry

	IncomingBackups []IncomingBackup
	IncomingCount   int
	IncomingSize    string

	ServerBackups []ServerBackup
}

var funcMap = template.FuncMap{
	"T": func(lang, key string, args ...interface{}) string {
		// Simple i18n mock - return key translations for common keys
		translations := map[string]string{
			"v2.nav.dashboard":              "Tableau de bord",
			"v2.nav.storage":                "Stockage",
			"v2.nav.backups":                "Sauvegardes",
			"v2.nav.network":                "Réseau",
			"v2.nav.peers":                  "Pairs",
			"v2.nav.wireguard":              "WireGuard",
			"v2.nav.system":                 "Système",
			"v2.nav.users":                  "Utilisateurs",
			"v2.nav.shares":                 "Partages",
			"v2.nav.settings":               "Paramètres",
			"v2.nav.trash":                  "Corbeille",
			"v2.nav.logs":                   "Journaux",
			"v2.nav.updates":                "Mises à jour",
			"v2.nav.theme":                  "Thème",
			"v2.nav.logout":                 "Déconnexion",
			"v2.dashboard.users":            "Utilisateurs",
			"v2.dashboard.active_users":     "utilisateurs actifs",
			"v2.dashboard.storage":          "Stockage",
			"v2.dashboard.peer_storage":     "Stockage pairs :",
			"v2.dashboard.peers":            "Pairs",
			"v2.dashboard.connected_peers":  "pairs connectés",
			"v2.dashboard.last_backup":      "Dernière sauvegarde",
			"v2.dashboard.configure_backups": "Configurer les sauvegardes",
			"v2.dashboard.recent_activity":  "Activité récente",
			"v2.dashboard.no_activity":      "Aucune activité récente",
			"v2.dashboard.backup_status":    "État des sauvegardes",
			"v2.dashboard.quick_actions":    "Actions rapides",
			"v2.dashboard.action_users":     "Utilisateurs",
			"v2.dashboard.active_label":     "actifs",
			"v2.dashboard.action_trash":     "Corbeille",
			"v2.dashboard.manage":           "Gérer",
			"v2.dashboard.action_peers":     "Pairs",
			"v2.dashboard.connected_label":  "connectés",
			"v2.dashboard.action_logs":      "Journaux système",
			"v2.dashboard.view_logs":        "Voir les journaux",
			"v2.backups.add":                "Ajouter",
			"v2.backups.edit":               "Modifier",
			"v2.backups.delete":             "Supprimer",
			"v2.backups.actions":            "Actions",
			"v2.backups.confirm_delete":     "Êtes-vous sûr ?",
			"v2.backups.never":              "Jamais",
			"v2.backups.status.success":     "Succès",
			"v2.backups.status.error":       "Erreur",
			"v2.backups.usb.title":          "Sauvegardes USB",
			"v2.backups.usb.name":           "Nom",
			"v2.backups.usb.device":         "Périphérique",
			"v2.backups.usb.status":         "Statut",
			"v2.backups.usb.last_sync":      "Dernière synchro",
			"v2.backups.usb.mounted":        "Monté",
			"v2.backups.usb.unmounted":       "Non monté",
			"v2.backups.usb.empty":          "Aucune sauvegarde USB configurée",
			"v2.backups.usb.available_drives": "Lecteurs disponibles",
			"v2.backups.cloud.title":        "Sauvegardes Cloud (SFTP)",
			"v2.backups.cloud.name":         "Nom",
			"v2.backups.cloud.host":         "Hôte",
			"v2.backups.cloud.status":       "Statut",
			"v2.backups.cloud.last_sync":    "Dernière synchro",
			"v2.backups.cloud.enabled":      "Activé",
			"v2.backups.cloud.disabled":     "Désactivé",
			"v2.backups.cloud.empty":        "Aucune destination cloud configurée",
			"v2.backups.cloud.ssh_key":      "Clé SSH",
			"v2.backups.cloud.configured":   "Configurée",
			"v2.backups.cloud.ssh_hint":     "La clé SSH est configurée pour les connexions SFTP.",
			"v2.backups.p2p.title":          "Synchronisation P2P",
			"v2.backups.p2p.configure":      "Configurer",
			"v2.backups.p2p.force_sync":     "Forcer la synchro",
			"v2.backups.p2p.status_label":   "Synchronisation",
			"v2.backups.p2p.enabled":        "Activée",
			"v2.backups.p2p.disabled":       "Désactivée",
			"v2.backups.p2p.interval":       "Intervalle",
			"v2.backups.p2p.peer":           "Pair",
			"v2.backups.p2p.user":           "Utilisateur",
			"v2.backups.p2p.files":          "Fichiers",
			"v2.backups.p2p.size":           "Taille",
			"v2.backups.p2p.date":           "Date",
			"v2.backups.p2p.empty":          "Aucune synchronisation récente",
			"v2.backups.incoming.title":       "Sauvegardes entrantes",
			"v2.backups.incoming.total_backups": "Total sauvegardes",
			"v2.backups.incoming.total_size":   "Taille totale",
			"v2.backups.incoming.source":       "Serveur source",
			"v2.backups.incoming.user":         "Utilisateur",
			"v2.backups.incoming.share":        "Partage",
			"v2.backups.incoming.files":        "Fichiers",
			"v2.backups.incoming.empty":        "Aucune sauvegarde entrante",
			"v2.backups.server.title":          "Sauvegardes serveur",
			"v2.backups.server.create":         "Créer une sauvegarde",
			"v2.backups.server.filename":       "Fichier",
			"v2.backups.server.download":       "Télécharger",
			"v2.backups.server.empty":          "Aucune sauvegarde serveur",
		}
		if v, ok := translations[key]; ok {
			return v
		}
		return key
	},
	"ServerName": func() string { return "Anemone NAS" },
	"FormatBytes": func(b int64) string { return fmt.Sprintf("%d B", b) },
	"slice": func(s string, start, end int) string {
		if end > len(s) {
			end = len(s)
		}
		if start >= len(s) {
			return ""
		}
		return s[start:end]
	},
}

func loadPage(page string) *template.Template {
	base := filepath.Join("web", "templates", "v2", "v2_base.html")
	pageFile := filepath.Join("web", "templates", "v2", page)
	return template.Must(
		template.New("v2_base.html").Funcs(funcMap).ParseFiles(base, pageFile),
	)
}

func main() {
	session := &Session{Username: "admin", IsAdmin: true}

	// Static files
	fs := http.FileServer(http.Dir("web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Dashboard
	http.HandleFunc("/v2/dashboard", func(w http.ResponseWriter, r *http.Request) {
		data := DashboardData{
			Lang:       "fr",
			Title:      "Tableau de bord",
			ActivePage: "dashboard",
			Session:    session,
			Stats: &DashboardStats{
				UserCount:       3,
				StorageUsed:     "175.85 GB",
				PeerStorageUsed: "42.3 GB",
				StorageQuota:    "2 TB",
				StoragePercent:  9,
				LastBackup:      "Il y a 2 h",
				PeerCount:       2,
				TrashCount:      5,
			},
			RecentActivity: []Activity{
				{Description: "Sync nas-bureau → alice ✓", Time: "Il y a 2 h", Status: "success"},
				{Description: "Sync nas-bureau → bob ✓", Time: "Il y a 2 h", Status: "success"},
				{Description: "User 'Charlie' created", Time: "Il y a 1 j", Status: "success"},
				{Description: "Sync nas-maison → alice ✗", Time: "Il y a 3 j", Status: "error"},
			},
		}
		tmpl := loadPage("v2_dashboard.html")
		if err := tmpl.ExecuteTemplate(w, "v2_base", data); err != nil {
			http.Error(w, err.Error(), 500)
		}
	})

	// Backups
	http.HandleFunc("/v2/backups", func(w http.ResponseWriter, r *http.Request) {
		data := BackupsData{
			Lang:       "fr",
			Title:      "Sauvegardes",
			ActivePage: "backups",
			Session:    session,
			USBBackups: []USBBackup{
				{ID: 1, Name: "Backup USB Principal", DevicePath: "/dev/sdb1", IsMounted: true, LastSync: "Il y a 2 h", LastStatus: "success"},
				{ID: 2, Name: "Backup USB Mensuel", DevicePath: "/dev/sdc1", IsMounted: false, LastSync: "Il y a 15 j", LastStatus: "success"},
			},
			USBDrives: []Drive{
				{Name: "Samsung T7", Size: "931.5 GB"},
				{Name: "WD Elements", Size: "1.8 TB"},
			},
			RcloneConfigs: []RcloneConfig{
				{ID: 1, Name: "Hetzner SFTP", Host: "sftp.example.com", RemotePath: "/backups/anemone", Enabled: true, LastSync: "Il y a 6 h"},
			},
			SSHKeyExists:  true,
			SyncEnabled:   true,
			SyncInterval:  "2h",
			RecentSyncs: []SyncEntry{
				{PeerName: "nas-bureau", Username: "alice", Status: "success", FilesSynced: 47, BytesSynced: "128.4 MB", CompletedAt: "Il y a 2 h"},
				{PeerName: "nas-bureau", Username: "bob", Status: "success", FilesSynced: 12, BytesSynced: "3.2 MB", CompletedAt: "Il y a 2 h"},
				{PeerName: "nas-maison", Username: "alice", Status: "error", FilesSynced: 0, BytesSynced: "0 B", CompletedAt: "Il y a 3 j"},
			},
			IncomingBackups: []IncomingBackup{
				{SourceServer: "nas-bureau", UserID: 1, ShareName: "documents", FileCount: 342, TotalSizeFormatted: "1.2 GB", Path: "nas-bureau/1_documents"},
				{SourceServer: "nas-bureau", UserID: 2, ShareName: "photos", FileCount: 1205, TotalSizeFormatted: "38.7 GB", Path: "nas-bureau/2_photos"},
			},
			IncomingCount: 2,
			IncomingSize:  "39.9 GB",
			ServerBackups: []ServerBackup{
				{Name: "anemone-backup-2026-02-08.enc", SizeFormatted: "4.2 MB", Date: "2026-02-08 04:00"},
				{Name: "anemone-backup-2026-02-07.enc", SizeFormatted: "4.1 MB", Date: "2026-02-07 04:00"},
			},
		}
		tmpl := loadPage("v2_backups.html")
		if err := tmpl.ExecuteTemplate(w, "v2_base", data); err != nil {
			http.Error(w, err.Error(), 500)
		}
	})

	// Root redirect
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/v2/dashboard", http.StatusSeeOther)
	})

	fmt.Println("Preview server: http://localhost:9090/v2/dashboard")
	log.Fatal(http.ListenAndServe(":9090", nil))
}
