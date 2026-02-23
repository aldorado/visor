# visor starten auf ubuntu 24.04 lts (deppen-sicher, deutsch)

ja, wirklich deppen-sicher.
einfach 1:1 block für block copy/pasten.

wenn ein step fehlschlägt: *nicht weitermachen*, erst den fehler lösen.

setup update:
- nutze das interaktive setup-wizard script für `.env`
- der wizard updated keys sicher (kein blindes überschreiben)
- diese anleitung zeigt den kompletten flow rundherum

---

## 0) was du brauchst

- frischer ubuntu 24.04 lts server (minimal reicht)
- sudo rechte
- telegram account
- ca. 15–30 minuten

empfohlen für test:
- 2 vcpu
- 2 gb ram
- 10 gb disk

---

## 1) grundpakete installieren

```bash
sudo apt update
sudo apt install -y git curl jq ca-certificates gnupg unzip tmux
sudo apt install -y golang-go
sudo apt install -y docker.io docker-compose-v2
```

prüfen:

```bash
go version
docker --version
docker compose version
git --version
```

optional (damit docker ohne sudo läuft):

```bash
sudo usermod -aG docker $USER
newgrp docker
```

---

## 2) visor code holen

```bash
sudo mkdir -p /root/code
cd /root/code
# falls schon geklont: nächste zeile skippen
# git clone <DEIN_REPO_URL> visor
cd visor
```

falls noch kein git repo:

```bash
git init
```

---

## 3) telegram bot token erstellen

1. in telegram: `@BotFather` öffnen
2. `/newbot` ausführen
3. namen + username setzen
4. token kopieren (`123456:ABC...`)

---

## 4) telegram chat id holen

schreib deinem bot zuerst eine nachricht (`hi`).

dann:

```bash
curl -s "https://api.telegram.org/botYOUR_BOT_TOKEN/getUpdates" | jq
```

gesucht ist:
- `.result[].message.chat.id`

diese nummer kommt später in `USER_PHONE_NUMBER`.
(ja, name ist legacy. wert ist chat id.)

---

## 5) setup wizard für `.env` ausführen

im ordner `/root/code/visor`:

```bash
./scripts/setup-wizard.sh
```

der wizard fragt:
- `TELEGRAM_BOT_TOKEN`
- `USER_PHONE_NUMBER`
- `AGENT_BACKEND` (`pi` oder `echo`)
- `OPENAI_API_KEY` (optional)
- `TZ` (default `Europe/Vienna`)

wichtig:
- der wizard updated keys in `.env` sicher (kein blindes überschreiben)
- zuerst mit `echo` testen
- danach auf `pi` wechseln

---

## 6) lokaler smoke test

```bash
cd /root/code/visor
set -a
source .env
set +a
mkdir -p bin
go build -o bin/visor .
./bin/visor
```

terminal offen lassen.

in zweitem terminal:

```bash
curl -s http://127.0.0.1:8080/health
```

soll sein:

```json
{"status":"ok"}
```

du solltest auch das visor ascii startup banner im terminal sehen. das ist normal.

wenn ja: `ctrl+c` im visor terminal.

---

## 7) public https url machen (telegram webhook braucht das)

schnellster weg: cloudflared tunnel.

cloudflared installieren:

```bash
curl -fsSL https://pkg.cloudflare.com/cloudflare-main.gpg | sudo gpg --dearmor -o /usr/share/keyrings/cloudflare-main.gpg
echo 'deb [signed-by=/usr/share/keyrings/cloudflare-main.gpg] https://pkg.cloudflare.com/cloudflared noble main' | sudo tee /etc/apt/sources.list.d/cloudflared.list
sudo apt update
sudo apt install -y cloudflared
```

visor wieder starten (terminal 1):

```bash
cd /root/code/visor
set -a
source .env
set +a
mkdir -p bin
go build -o bin/visor .
./bin/visor
```

tunnel starten (terminal 2):

```bash
cloudflared tunnel --url http://127.0.0.1:8080
```

du bekommst eine url wie:
- `https://something.trycloudflare.com`

die url kopieren.

---

## 8) webhook setzen

```bash
curl -s "https://api.telegram.org/botYOUR_BOT_TOKEN/setWebhook" \
  -d "url=https://YOUR_PUBLIC_HOST/webhook" \
  -d "secret_token=YOUR_TELEGRAM_WEBHOOK_SECRET"
```

status prüfen:

```bash
curl -s "https://api.telegram.org/botYOUR_BOT_TOKEN/getWebhookInfo" | jq
```

wenn da fehler stehen: *erst fixen*, dann weiter.

branding-hinweis:
- offizieller logo-dateipfad im repo: `docs/assets/visor-logo.png`

---

## 9) echter test in telegram

dem bot schreiben.

mit `AGENT_BACKEND=echo` muss direkt eine `echo: ...` antwort kommen.

wenn ja: basis setup passt ✅


---

## 10) von echo auf pi wechseln

nur wenn step 9 funktioniert.

1) pi cli installieren und einloggen
2) prüfen:

```bash
pi --version
pi auth status
```

3) `.env` ändern:

```bash
AGENT_BACKEND=pi
AGENT_BACKENDS=pi,echo
```

4) visor neu starten und testen.


---

## 11) als service laufen lassen (damit es nach logout weiterläuft)

systemd file:

```bash
sudo tee /etc/systemd/system/visor.service >/dev/null <<'EOF'
[Unit]
Description=Visor Agent Runtime
After=network.target

[Service]
Type=simple
WorkingDirectory=/root/code/visor
EnvironmentFile=/root/code/visor/.env
ExecStart=/usr/bin/env bash -lc 'set -a; source /root/code/visor/.env; set +a; exec /root/code/visor/bin/visor'
Restart=always
RestartSec=3
User=root

[Install]
WantedBy=multi-user.target
EOF
```

build + start + enable:

```bash
cd /root/code/visor
mkdir -p bin
go build -o bin/visor .
sudo systemctl daemon-reload
sudo systemctl enable --now visor
sudo systemctl status visor --no-pager
```

logs live:

```bash
journalctl -u visor -f
```

---

## 12) klassische fehler

### bot antwortet nicht
- webhook falsch
- public url down
- falsche chat id in `USER_PHONE_NUMBER`

check:

```bash
curl -s "https://api.telegram.org/botYOUR_BOT_TOKEN/getWebhookInfo" | jq
```

### `TELEGRAM_BOT_TOKEN is required`
- `.env` fehlt oder wurde nicht geladen
- key falsch geschrieben

### `USER_PHONE_NUMBER is required`
- chat id fehlt

### läuft, aber keine antwort

```bash
journalctl -u visor -n 200 --no-pager
```

---

## 13) danach (optional)

- openai + elevenlabs keys für voice setzen
- otel/sigNoz: `docs/signoz-setup.md`

---

wenn du willst, kann ich als nächstes noch eine *single-script install.sh* version bauen (one-shot installer).