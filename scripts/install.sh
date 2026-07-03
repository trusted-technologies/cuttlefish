#!/usr/bin/env bash
set -e

# Cuttlefish interactive installer.
# Supports: master install, slave install, update, reinstall, uninstall.
#
# Prompts are read from /dev/tty so the script can be piped from curl and
# still remain interactive.

if [[ ! -e /dev/tty ]]; then
    echo "This installer requires a terminal (/dev/tty)." >&2
    exit 1
fi

INSTALL_DIR="/opt/cuttlefish"
ENV_FILE="${INSTALL_DIR}/.env"
COMPOSE_FILE="${INSTALL_DIR}/docker-compose.yml"
NGINX_AVAIL="/etc/nginx/sites-available/cuttlefish"
NGINX_ENABLED="/etc/nginx/sites-enabled/cuttlefish"

REPO="trusted-technologies/cuttlefish"
MASTER_IMAGE="ghcr.io/${REPO}-master:main"
SLAVE_IMAGE="ghcr.io/${REPO}-slave:main"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

log() { echo -e "\033[1;34m[cuttlefish]\033[0m $*"; }
warn() { echo -e "\033[1;33m[cuttlefish]\033[0m $*" >&2; }
err() { echo -e "\033[1;31m[cuttlefish]\033[0m $*" >&2; }

_read_tty() {
    if [[ -t 0 ]]; then
        read -r "$@"
    else
        read -r "$@" < /dev/tty
    fi
}

ask() {
    local prompt="$1"
    local default="${2:-}"
    local value
    if [[ -n $default ]]; then
        _read_tty -p "${prompt} [${default}]: " value
        echo "${value:-$default}"
    else
        _read_tty -p "${prompt}: " value
        echo "$value"
    fi
}

ask_yesno() {
    local prompt="$1"
    local default="${2:-y}"
    local value
    while true; do
        _read_tty -p "${prompt} [${default}]: " value
        value="${value:-$default}"
        case "$value" in
            [Yy]|[Yy][Ee][Ss]) return 0 ;;
            [Nn]|[Nn][Oo]) return 1 ;;
            *) echo "Please answer y or n." ;;
        esac
    done
}

require_root() {
    if [[ $EUID -ne 0 ]]; then
        err "This script must be run as root. Try: sudo $0"
        exit 1
    fi
}

generate_token() {
    openssl rand -hex 32
}

detect_public_ipv4() {
    local ip=""
    if command -v curl >/dev/null 2>&1; then
        ip=$(curl -4s --max-time 5 https://ifconfig.me 2>/dev/null || true)
    fi
    if [[ -z $ip ]] && command -v wget >/dev/null 2>&1; then
        ip=$(wget -qO- --timeout=5 https://ifconfig.me 2>/dev/null || true)
    fi
    echo "$ip"
}

detect_public_ipv6() {
    local ip=""
    if command -v curl >/dev/null 2>&1; then
        ip=$(curl -6s --max-time 5 https://ifconfig.me 2>/dev/null || true)
    fi
    if [[ -z $ip ]] && command -v wget >/dev/null 2>&1; then
        ip=$(wget -qO- --timeout=5 -6 https://ifconfig.me 2>/dev/null || true)
    fi
    echo "$ip"
}

parse_url_port() {
    local url="$1"
    local default_port="${2:-8080}"
    local host_port="${url#*://}"
    local host="${host_port%%:*}"
    local port="${host_port##*:}"
    if [[ "$host" == "$host_port" || "$port" == "$host_port" ]]; then
        echo "$default_port"
    else
        echo "$port"
    fi
}

# ---------------------------------------------------------------------------
# Docker
# ---------------------------------------------------------------------------

ensure_docker() {
    if command -v docker >/dev/null 2>&1; then
        log "Docker already installed."
        return 0
    fi
    log "Installing Docker..."
    curl -sSL https://get.docker.com/ | CHANNEL=stable bash
    systemctl enable --now docker
    log "Docker installed."
}

pull_images() {
    log "Pulling latest images..."
    docker pull "$MASTER_IMAGE"
    docker pull "$SLAVE_IMAGE"
}

# ---------------------------------------------------------------------------
# Master install
# ---------------------------------------------------------------------------

install_nginx() {
    log "Installing Nginx..."
    if command -v apt-get >/dev/null 2>&1; then
        apt-get update -qq
        apt-get install -y -qq nginx
    elif command -v dnf >/dev/null 2>&1; then
        dnf install -y nginx
    elif command -v yum >/dev/null 2>&1; then
        yum install -y nginx
    else
        warn "Could not install Nginx automatically. Please install it manually."
        return 1
    fi
    systemctl enable --now nginx
}

configure_nginx_http() {
    local domain="$1"
    cat > "$NGINX_AVAIL" <<EOF
server {
    listen 80;
    listen [::]:80;
    server_name ${domain};

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF
    rm -f "$NGINX_ENABLED"
    ln -s "$NGINX_AVAIL" "$NGINX_ENABLED"
    nginx -t && systemctl reload nginx
}

configure_nginx_selfsigned() {
    local domain="$1"
    local ssl_dir="/etc/ssl/cuttlefish"
    mkdir -p "$ssl_dir"
    if [[ ! -f "${ssl_dir}/${domain}.crt" ]]; then
        log "Generating self-signed SSL certificate for ${domain}..."
        openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
            -keyout "${ssl_dir}/${domain}.key" \
            -out "${ssl_dir}/${domain}.crt" \
            -subj "/CN=${domain}" >/dev/null 2>&1
    fi
    cat > "$NGINX_AVAIL" <<EOF
server {
    listen 80;
    listen [::]:80;
    server_name ${domain};
    return 301 https://\$host\$request_uri;
}

server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name ${domain};

    ssl_certificate ${ssl_dir}/${domain}.crt;
    ssl_certificate_key ${ssl_dir}/${domain}.key;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF
    rm -f "$NGINX_ENABLED"
    ln -s "$NGINX_AVAIL" "$NGINX_ENABLED"
    nginx -t && systemctl reload nginx
}

configure_certbot() {
    local domain="$1"
    log "Installing Certbot..."
    if command -v apt-get >/dev/null 2>&1; then
        apt-get update -qq
        apt-get install -y -qq certbot python3-certbot-nginx
    elif command -v dnf >/dev/null 2>&1; then
        dnf install -y certbot python3-certbot-nginx
    elif command -v yum >/dev/null 2>&1; then
        yum install -y certbot python3-certbot-nginx
    else
        warn "Could not install Certbot. Falling back to self-signed SSL."
        configure_nginx_selfsigned "$domain"
        return
    fi
    local email
    email=$(ask "Enter email for Let's Encrypt notifications")
    configure_nginx_http "$domain"
    certbot --nginx --non-interactive --agree-tos -m "$email" -d "$domain" || true
    systemctl reload nginx
}

install_master() {
    require_root
    ensure_docker

    mkdir -p "$INSTALL_DIR"

    local domain port token use_nginx ssl_choice
    domain=$(ask "Enter domain for the Looking Glass (e.g. lg.example.com)")
    if [[ -z $domain ]]; then
        err "Domain cannot be empty."
        exit 1
    fi
    port=$(ask "Internal master port" "8080")
    token=$(generate_token)

    log "Generated MASTER_TOKEN: ${token}"
    log "Write it down if you plan to add slaves."

    if ask_yesno "Install and configure Nginx as a reverse proxy?" "y"; then
        use_nginx="true"
        echo "SSL options:"
        echo "  1) Let's Encrypt (requires domain pointed to this server)"
        echo "  2) Self-signed certificate"
        echo "  3) HTTP only"
        ssl_choice=$(ask "Choose SSL option" "1")
        install_nginx
        case "$ssl_choice" in
            1) configure_certbot "$domain" ;;
            2) configure_nginx_selfsigned "$domain" ;;
            *) configure_nginx_http "$domain" ;;
        esac
    else
        use_nginx="false"
    fi

    cat > "$ENV_FILE" <<EOF
CUTTLEFISH_ROLE="master"
MASTER_DOMAIN="${domain}"
MASTER_PORT="${port}"
MASTER_TOKEN="${token}"
USE_NGINX="${use_nginx}"
EOF

    if [[ $use_nginx == "true" ]]; then
        cat > "$COMPOSE_FILE" <<EOF
services:
  master:
    image: ${MASTER_IMAGE}
    container_name: cuttlefish-master
    restart: unless-stopped
    ports:
      - "127.0.0.1:${port}:8080"
    environment:
      MASTER_ADDR: ":8080"
      MASTER_TOKEN: "${token}"
    volumes:
      - cuttlefish-master-data:/data
volumes:
  cuttlefish-master-data:
EOF
    else
        # No nginx: expose master directly.
        cat > "$COMPOSE_FILE" <<EOF
services:
  master:
    image: ${MASTER_IMAGE}
    container_name: cuttlefish-master
    restart: unless-stopped
    ports:
      - "${port}:8080"
    environment:
      MASTER_ADDR: ":8080"
      MASTER_TOKEN: "${token}"
    volumes:
      - cuttlefish-master-data:/data
volumes:
  cuttlefish-master-data:
EOF
    fi

    log "Starting master container..."
    docker compose -f "$COMPOSE_FILE" up -d --pull always

    log "Master installed."
    log "URL: http://${domain} (or https://${domain} if SSL configured)"
    log "MASTER_TOKEN: ${token}"
}

# ---------------------------------------------------------------------------
# Slave install
# ---------------------------------------------------------------------------

install_slave() {
    require_root
    ensure_docker

    mkdir -p "$INSTALL_DIR"

    local master_url token slave_id slave_name location ipv4 ipv6 public_url iperf_port files_sizes stats_interfaces
    master_url=$(ask "Enter master URL (e.g. https://lg.example.com)")
    token=$(ask "Enter MASTER_TOKEN")
    slave_id=$(ask "Slave ID" "$(hostname -f 2>/dev/null || hostname)")
    slave_name=$(ask "Slave display name" "$slave_id")
    location=$(ask "Slave location" "Unknown")

    ipv4=$(detect_public_ipv4)
    ipv4=$(ask "Public IPv4" "$ipv4")

    ipv6=$(detect_public_ipv6)
    if [[ -n $ipv6 ]]; then
        ipv6=$(ask "Public IPv6 (leave empty if none)" "$ipv6")
    else
        ipv6=$(ask "Public IPv6 (leave empty if none)")
    fi

    public_url=$(ask "Public URL for this slave" "http://${ipv4}:8080")
    local public_port
    public_port=$(parse_url_port "$public_url" 8080)
    iperf_port=$(ask "iPerf3 server port" "5201")
    files_sizes=$(ask "Test file sizes (comma-separated, empty for all)" "")
    stats_interfaces=$(ask "Network interfaces to monitor (comma-separated, empty for auto)" "")

    cat > "$ENV_FILE" <<EOF
CUTTLEFISH_ROLE="slave"
MASTER_URL="${master_url}"
SLAVE_TOKEN="${token}"
SLAVE_ID="${slave_id}"
SLAVE_NAME="${slave_name}"
SLAVE_PUBLIC_URL="${public_url}"
SLAVE_IPV4="${ipv4}"
SLAVE_IPV6="${ipv6}"
SLAVE_LOCATION="${location}"
IPERF_PORT="${iperf_port}"
FILES_SIZES="${files_sizes}"
STATS_INTERFACES="${stats_interfaces}"
FILES_DIR="/data/files"
EOF

    cat > "$COMPOSE_FILE" <<EOF
services:
  slave:
    image: ${SLAVE_IMAGE}
    container_name: cuttlefish-slave
    restart: unless-stopped
    ports:
      - "${public_port}:8080"
      - "${iperf_port}:${iperf_port}"
    cap_add:
      - NET_RAW
    environment:
      SLAVE_ID: "${slave_id}"
      SLAVE_NAME: "${slave_name}"
      SLAVE_PUBLIC_URL: "${public_url}"
      SLAVE_IPV4: "${ipv4}"
      SLAVE_IPV6: "${ipv6}"
      MASTER_URL: "${master_url}"
      SLAVE_TOKEN: "${token}"
      SLAVE_LOCATION: "${location}"
      SLAVE_ADDR: ":8080"
      IPERF_PORT: "${iperf_port}"
      FILES_SIZES: "${files_sizes}"
      STATS_INTERFACES: "${stats_interfaces}"
      FILES_DIR: "/data/files"
    volumes:
      - cuttlefish-slave-files:/data/files
volumes:
  cuttlefish-slave-files:
EOF

    log "Starting slave container..."
    docker compose -f "$COMPOSE_FILE" up -d --pull always

    log "Slave installed."
}

# ---------------------------------------------------------------------------
# Update / reinstall / uninstall
# ---------------------------------------------------------------------------

update_cuttlefish() {
    require_root
    if [[ ! -f "$ENV_FILE" ]]; then
        err "No existing installation found at ${INSTALL_DIR}."
        exit 1
    fi
    # shellcheck source=/dev/null
    source "$ENV_FILE"
    log "Updating Cuttlefish (${CUTTLEFISH_ROLE})..."
    docker compose -f "$COMPOSE_FILE" pull
    docker compose -f "$COMPOSE_FILE" up -d
    log "Updated."
}

reinstall_cuttlefish() {
    require_root
    if [[ ! -f "$ENV_FILE" ]]; then
        warn "No existing installation found. Proceeding with fresh install."
    else
        # shellcheck source=/dev/null
        source "$ENV_FILE"
        log "Current role: ${CUTTLEFISH_ROLE}"
        if ! ask_yesno "Do you want to reconfigure and reinstall?" "y"; then
            exit 0
        fi
    fi
    uninstall_cuttlefish
    echo
    log "Starting reinstallation..."
    show_menu
}

uninstall_cuttlefish() {
    require_root
    if [[ -f "$COMPOSE_FILE" ]]; then
        log "Stopping and removing containers..."
        docker compose -f "$COMPOSE_FILE" down -v || true
    fi
    if [[ -f "$NGINX_ENABLED" ]]; then
        rm -f "$NGINX_ENABLED" "$NGINX_AVAIL"
        systemctl reload nginx 2>/dev/null || true
    fi
    if [[ -d "$INSTALL_DIR" ]]; then
        if ask_yesno "Remove configuration directory ${INSTALL_DIR}?" "y"; then
            rm -rf "$INSTALL_DIR"
        fi
    fi
    log "Uninstalled."
}

# ---------------------------------------------------------------------------
# Menu
# ---------------------------------------------------------------------------

show_menu() {
    echo
    echo "=================================="
    echo "   Cuttlefish installer"
    echo "=================================="
    echo "1) Install master"
    echo "2) Install slave"
    echo "3) Update existing installation"
    echo "4) Reinstall / reconfigure"
    echo "5) Uninstall"
    echo "6) Exit"
    echo
    local choice
    choice=$(ask "Choose an option" "1")
    case "$choice" in
        1) install_master ;;
        2) install_slave ;;
        3) update_cuttlefish ;;
        4) reinstall_cuttlefish ;;
        5) uninstall_cuttlefish ;;
        6) exit 0 ;;
        *) warn "Invalid option"; show_menu ;;
    esac
}

# ---------------------------------------------------------------------------
# Entrypoint
# ---------------------------------------------------------------------------

case "${1:-}" in
    master) install_master ;;
    slave) install_slave ;;
    update) update_cuttlefish ;;
    reinstall) reinstall_cuttlefish ;;
    uninstall) uninstall_cuttlefish ;;
    *) show_menu ;;
esac
