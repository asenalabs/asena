# ============================================
# File: scripts/install.sh
# Purpose: Install Asena binary, create system user, create config files,
#          install systemd unit, and set correct permissions.
# Usage: sudo ./scripts/install.sh
# ============================================

#!/usr/bin/env bash
set -euo pipefail

#CONFIGURABLE PARAMETERS
SERVICE_USER="asena"
BINARY_SRC="./bin/asena"
BINARY_DEST="/usr/local/bin/asena"
CONFIG_DIR="/etc/asena"
LIB_DIR="/var/lib/asena"
LOG_DIR="/var/log/asena"
SERVICE_FILE_DEST="/etc/systemd/system/asena.service"
TMPFILES_DEST="/etc/tmpfiles.d/asena.conf"

# Helper: print a prefixed message
info() { echo "[INFO] $*"; }
error() { echo "[ERROR] $*" >&2; }

# ---------------------------
# Preflight: must be root
# ---------------------------
if [ "$(id -u)" -ne 0 ]; then
    error "This installer must be run as root. Use sudo ./scripts/install.sh"
    exit 1
fi

info "Starting Asena installation..."

# ---------------------------
# Ensure binary exists
# ---------------------------
if [ ! -f "$BINARY_SRC" ]; then
    error "Compiled binary not found at '$BINARY_SRC'. Please build it first.\n"
    exit 1
fi

# ---------------------------
# Install binary
# ---------------------------
info "Installing binary to $BINARY_DEST"
install -m 0755 "$BINARY_SRC" "$BINARY_DEST"

# ---------------------------
# Create service user
# ---------------------------
if id "$SERVICE_USER" >/dev/null 2>&1; then
    info "User '$SERVICE_USER' already exists"
else
    info "Creating system user '$SERVICE_USER' (no home, nologin)"
    useradd --system --no-create-home --shell /usr/sbin/nologin --user-group "$SERVICE_USER"
fi

# ---------------------------
# Create directories
# ---------------------------
info "Creating directories: $CONFIG_DIR, $LIB_DIR, $LOG_DIR"
mkdir -p "$CONFIG_DIR" "$LIB_DIR" "$LOG_DIR"
chown -R root:"$SERVICE_USER" "$CONFIG_DIR"
chown -R "$SERVICE_USER":"$SERVICE_USER" "$LIB_DIR" "$LOG_DIR"
chmod 0750 "$CONFIG_DIR"

# ---------------------------
# Create default config files (if missing)
# ---------------------------
# Create a default static config (owned by root:asena)
if [ ! -f "$CONFIG_DIR/asena.yaml" ]; then
    info "Craeting default static config: $CONFIG_DIR/asena.yaml"
    touch "$CONFIG_DIR/asena.yaml" 
    chown root:"$SERVICE_USER" "$CONFIG_DIR/asena.yaml"
    chmod 0640 "$CONFIG_DIR/asena.yaml"
else
    info "Static config already exists, skipping: $CONFIG_DIR/asena.yaml"
fi

# Creating a default dynamic config (owned by asena)
if [ ! -f "$CONFIG_DIR/dynamic.yaml" ]; then
    info "Creating default dynamic config: $CONFIG_DIR/dynamic.yaml"
    touch "$CONFIG_DIR/dynamic.yaml"
    chown "$SERVICE_USER":"$SERVICE_USER" "$CONFIG_DIR/dynamic.yaml"
    chmod 0640 "$CONFIG_DIR/dynamic.yaml"
else
    info "Dynamic config already exists, skipping: $CONFIG_DIR/dynamic.yaml"
fi

# ---------------------------
# Install systemd unit file
# ---------------------------
if [ ! -f "./systemd/asena.service" ]; then
    error "Missing './systemd/asena.service' in repo. Please include it before installing."
    exit 1
fi

info "Installing systemd service unit to $SERVICE_FILE_DEST"
install -m 0644 ./systemd/asena.service "$SERVICE_FILE_DEST"

# ---------------------------
# Install tmpfiles config
# ---------------------------
if [ -f "./systemd/asena.conf" ]; then
    info "Installing tmpfiles.d entry to $TMPFILES_DEST"
    install -m 0644 ./systemd/asena.conf "$TMPFILES_DEST"

    if command -v systemd-tmpfiles >/dev/null 2>&1; then
        info "Running systemd-tmpfiles to create directories"
        systemd-tmpfiles --create "$TMPFILES_DEST"
    else
        info "systemd-tmpfiles not found, creating dirs manually"
        mkdir -p /run/asena /var/lib/asena /var/log/asena
        chown -R "$SERVICE_USER":"$SERVICE_USER" /run/asena /var/lib/asena /var/log/asena
    fi
else
    info "No tmpfiles config found (./systemd/asena.conf), skipping."
fi

# ---------------------------
# Reload systemd and enable service
# ---------------------------
info "Reloading systemd deamon"
systemctl daemon-reload

info "Enabling and starting asena service"
systemctl enable --now asena || {
    error "Failedto enable/start asena. Check 'journalctl -u asena' for details."
    exit 1
}

info "Instalation complete."
info "Config: $CONFIG_DIR"
info "Data: $LIB_DIR"
info "Logs: $LOG_DIR"
info "Binary: $BINARY_DEST"

# End of install.sh