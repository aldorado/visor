# visor starten auf ubuntu 24.04 lts (deppen-sicher, deutsch)

ja, wirklich deppen-sicher.
einfach 1:1 block für block copy/pasten.

wenn ein step fehlschlägt: *nicht weitermachen*, erst den fehler lösen.

m12 update:
- visor hat jetzt einen interaktiven setup-flow (first-run mode)
- der wird aktiv, wenn bootstrap noch fehlt (z.b. keine `.env` + keine basis-envs)
- diese anleitung bleibt der manuelle, stabile fallback

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

## 5) `.env` anlegen

im ordner `/root/code/visor`:

```bash
cat > .env <<'EOF'
TELEGRAM_BOT_TOKEN=PASTE_BOT_TOKEN_HERE
USER_PHONE_NUMBER=PASTE_CHAT_ID_HERE
PORT=8080

# fürs erste super einfach starten
AGENT_BACKEND=echo
AGENT_BACKENDS=echo

# empfohlen
TELEGRAM_WEBHOOK_SECRET=change-me-random-secret

# logs
LOG_LEVEL=info
LOG_VERBOSE=false

# runtime daten
DATA_DIR=data
EOF
```

wichtig:
- zuerst mit `echo` testen
- `echo` ist nur ein dummy-smoke-test backend und antwortet mit `echo: <deine nachricht>`
- `pi` erst danach

---

## 6) lokaler smoke test

```bash
cd /root/code/visor
set -a
source .env
set +a
go run .
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
go run .
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

m12-hinweis (wichtig):
- mit `AGENT_BACKEND=echo` gibt es *keinen* echten setup-assistenten (echo ist nur smoke-test)
- der interaktive m12 setup-flow braucht ein echtes backend (`pi` oder `claude`)

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

optional (m12 guided setup nutzen):
- wenn du setup lieber im chat führen willst, starte visor mit echtem backend (`pi`/`claude`) und folge den setup-fragen
- m12 kann jetzt zusätzlich:
  - openai key validieren (`validate_openai`)
  - levelups gesammelt auswählen (`none` | `recommended` | explizite liste)
  - levelups in reihenfolge anwenden (env -> enable -> validate -> start -> health)
  - am ende test-message + setup-summary schreiben

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
ExecStart=/usr/bin/env bash -lc 'set -a; source /root/code/visor/.env; set +a; /usr/bin/go run .'
Restart=always
RestartSec=3
User=root

[Install]
WantedBy=multi-user.target
EOF
```

start + enable:

```bash
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

- `.levelup.env` aus `.levelup.env.example` bauen
- levelups aktivieren/validieren
- openai + elevenlabs keys für voice setzen
- otel/sigNoz: `docs/signoz-setup.md`

---

wenn du willst, kann ich als nächstes noch eine *single-script install.sh* version bauen (one-shot installer).