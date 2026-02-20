#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
UNIT_PATH="/etc/systemd/system/visor.service"

cd "$ROOT"
mkdir -p bin
go build -o bin/visor .

cat > "$UNIT_PATH" <<'EOF'
[Unit]
Description=visor ai runtime
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
Environment=PATH=/root/.nvm/versions/node/v24.13.1/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
WorkingDirectory=/root/code/visor
ExecStart=/bin/bash -lc 'set -a; source /root/code/visor/.env; set +a; exec /root/code/visor/bin/visor'
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now visor
systemctl status visor --no-pager
