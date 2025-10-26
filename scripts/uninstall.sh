# ============================================
# File: scripts/uninstall.sh
# Purpose: Uninstall Asena safely
# Usage: sudo ./scripts/uninstall.sh
# ============================================

#!/usr/bin/env bash
set -euo pipefail

SERVICE_USER="asena"
BINARY_DEST="/usr/local/bin/asena"
CONFIG_DIR="/etc/asena"
LIB_DIR="/var/lib/asena"
LOG_DIR="/var/log/asena"
SERVICE_FILE="/etc/systemd/system/asena.service"
TMPFILES_DEST="/etc/tmpfiles.d/asena.conf"

info() { echo "[INFO] $*"; }
error() { echo "[ERROR] $*" >&2; }

if [ "$(id -u)" -ne 0 ]; then
  error "Run as root: sudo ./scripts/uninstall.sh"
  exit 1
fi

info "Stopping and disabling Asena service if present"
if systemctl is-active --quiet asena; then
  systemctl stop asena || true
  info "Service stopped"
fi

if systemctl is-enabled --quiet asena; then
  systemctl disable asena || true
  info "Service disabled"
fi

if [ -f "$SERVICE_FILE" ]; then
  rm -f "$SERVICE_FILE"
  systemctl daemon-reload || true
  info "Removed systemd unit $SERVICE_FILE"
fi

if [ -f "$BINARY_DEST" ]; then
  rm -f "$BINARY_DEST"
  info "Removed binary: $BINARY_DEST"
fi

# Ask before deleting config/data/logs
read -r -p "Do you want to delete config directory '$CONFIG_DIR'? (y/N): " delconf
if [[ "$delconf" =~ ^[Yy]$ ]]; then
  rm -rf "$CONFIG_DIR"
  info "Deleted $CONFIG_DIR"
else
  info "Keeping $CONFIG_DIR"
fi

read -r -p "Do you want to delete data directory '$LIB_DIR'? (y/N): " deldata
if [[ "$deldata" =~ ^[Yy]$ ]]; then
  rm -rf "$LIB_DIR"
  info "Deleted $LIB_DIR"
else
  info "Keeping $LIB_DIR"
fi

read -r -p "Do you want to delete log directory '$LOG_DIR'? (y/N): " dellog
if [[ "$dellog" =~ ^[Yy]$ ]]; then
  rm -rf "$LOG_DIR"
  info "Deleted $LOG_DIR"
else
  info "Keeping $LOG_DIR"
fi

# Remove tmpfiles entry if exists
if [ -f "$TMPFILES_DEST" ]; then
  rm -f "$TMPFILES_DEST"
  info "Removed tmpfiles entry: $TMPFILES_DEST"
fi

# Optionally remove user
if id "$SERVICE_USER" >/dev/null 2>&1; then
  read -r -p "Delete system user '$SERVICE_USER'? (y/N): " deluser
  if [[ "$deluser" =~ ^[Yy]$ ]]; then
    userdel "$SERVICE_USER" || true
    info "Removed user $SERVICE_USER"
  else
    info "Keeping user $SERVICE_USER"
  fi
fi

info "Uninstall complete."

# End of uninstall.sh