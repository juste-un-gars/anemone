#!/bin/bash

# Anemone Auto-Update Script
# This script performs automatic updates of Anemone from the web interface
# It runs independently and continues even after the server restarts

set -e  # Exit on error

# Configuration
LOG_FILE="/tmp/anemone-update.log"
BACKUP_DIR="/tmp/anemone-backup-$(date +%Y%m%d-%H%M%S)"
PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TARGET_VERSION="$1"

# Setup Go environment (required when running in background with nohup)
if [ -f "/etc/profile.d/go.sh" ]; then
    source /etc/profile.d/go.sh
fi
# Fallback: set PATH manually if go.sh doesn't exist
if ! command -v go &> /dev/null; then
    export PATH=$PATH:/usr/local/go/bin
    export GOPATH=$HOME/go
    export PATH=$PATH:$GOPATH/bin
fi

# Logging function
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

log_error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $1" | tee -a "$LOG_FILE" >&2
}

# Start update process
log "========================================="
log "Starting Anemone auto-update process"
log "Project directory: $PROJECT_DIR"
log "Target version: v$TARGET_VERSION"
log "========================================="

# Step 1: Create backup directory
log "Step 1/6: Creating backup directory..."
mkdir -p "$BACKUP_DIR"
log "Backup directory created: $BACKUP_DIR"

# Step 2: Backup current binaries
log "Step 2/6: Backing up current binaries..."
if [ -f "$PROJECT_DIR/anemone" ]; then
    cp "$PROJECT_DIR/anemone" "$BACKUP_DIR/anemone"
    log "Backed up: anemone"
fi
if [ -f "$PROJECT_DIR/anemone-dfree" ]; then
    cp "$PROJECT_DIR/anemone-dfree" "$BACKUP_DIR/anemone-dfree"
    log "Backed up: anemone-dfree"
fi

# Step 3: Fetch tags and checkout target version
log "Step 3/6: Fetching tags and checking out version v$TARGET_VERSION..."
cd "$PROJECT_DIR"

# Fetch all tags from GitHub (--force to allow updating existing tags)
if ! git fetch --tags --force >> "$LOG_FILE" 2>&1; then
    log_error "Failed to fetch tags from GitHub"
    log "Update failed. Binaries not modified."
    exit 1
fi

# Checkout the target version tag
if ! git checkout "tags/v$TARGET_VERSION" >> "$LOG_FILE" 2>&1; then
    log_error "Failed to checkout version v$TARGET_VERSION"
    log "Update failed. Binaries not modified."
    exit 1
fi
log "Successfully checked out version v$TARGET_VERSION"

# Step 4: Build main binary
log "Step 4/6: Building anemone binary..."
if ! go build -o anemone cmd/anemone/main.go >> "$LOG_FILE" 2>&1; then
    log_error "Failed to build anemone binary"
    log "Restoring backup..."
    if [ -f "$BACKUP_DIR/anemone" ]; then
        cp "$BACKUP_DIR/anemone" "$PROJECT_DIR/anemone"
        log "Restored anemone binary from backup"
    fi
    exit 1
fi
log "Successfully built anemone binary"

# Step 5: Build anemone-dfree binary
log "Step 5/6: Building anemone-dfree binary..."
if ! go build -o anemone-dfree cmd/anemone-dfree/main.go >> "$LOG_FILE" 2>&1; then
    log_error "Failed to build anemone-dfree binary"
    log "Restoring backup..."
    if [ -f "$BACKUP_DIR/anemone" ]; then
        cp "$BACKUP_DIR/anemone" "$PROJECT_DIR/anemone"
    fi
    if [ -f "$BACKUP_DIR/anemone-dfree" ]; then
        cp "$BACKUP_DIR/anemone-dfree" "$PROJECT_DIR/anemone-dfree"
    fi
    log "Restored binaries from backup"
    exit 1
fi
log "Successfully built anemone-dfree binary"

# Step 6: Restart service
log "Step 6/6: Restarting anemone service..."
# Wait 3 seconds to allow the HTTP response to be sent
sleep 3

# Restart the service using sudo (requires NOPASSWD configuration)
sudo systemctl restart anemone >> "$LOG_FILE" 2>&1

if [ $? -eq 0 ]; then
    log "Successfully restarted anemone service"
    log "========================================="
    log "Update completed successfully!"
    log "Backup kept in: $BACKUP_DIR"
    log "========================================="
else
    log_error "Failed to restart service"
    log "You may need to restart manually: sudo systemctl restart anemone"
    exit 1
fi

exit 0
