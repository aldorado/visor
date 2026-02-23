# visor setup for ubuntu 24.04 lts (noob edition)

this guide is intentionally very explicit.
copy/paste one block at a time.

if something fails, stop there and fix that step first.

m12 update:
- visor now has an interactive first-run setup mode
- it activates when bootstrap is missing (for example no `.env` + missing base envs)
- this guide is still the manual, stable fallback path

---

## 0) what you need

- a fresh ubuntu 24.04 lts server (minimal is fine)
- sudo access
- a telegram account
- 15–30 minutes

recommended machine for testing:
- 2 vcpu
- 2 gb ram
- 10 gb disk

---

## 1) install system packages

```bash
sudo apt update
sudo apt install -y git curl jq ca-certificates gnupg unzip tmux
sudo apt install -y golang-go
sudo apt install -y docker.io docker-compose-v2
```

verify:

```bash
go version
docker --version
docker compose version
git --version
```

optional (lets your normal user run docker without sudo):

```bash
sudo usermod -aG docker $USER
newgrp docker
```

---

## 2) get the visor code

```bash
sudo mkdir -p /root/code
cd /root/code
# if already cloned, skip next line
# git clone <YOUR_REPO_URL> visor
cd visor
```

if this is not a git repo yet, initialize now:

```bash
git init
```

---

## 3) create telegram bot token

1. in telegram, open `@BotFather`
2. run `/newbot`
3. choose name + username
4. copy the token (looks like `123456:ABC...`)

save it somewhere for next step.

---

## 4) get your telegram chat id

send any message to your new bot first (for example: `hi`).

then run (replace `YOUR_BOT_TOKEN`):

```bash
curl -s "https://api.telegram.org/botYOUR_BOT_TOKEN/getUpdates" | jq
```

find the number at:
- `.result[].message.chat.id`

that numeric value is your `USER_PHONE_NUMBER` env value (name is legacy, value is chat id).

---

## 5) create `.env`

in `/root/code/visor`, create this file:

```bash
cat > .env <<'EOF'
TELEGRAM_BOT_TOKEN=PASTE_BOT_TOKEN_HERE
USER_PHONE_NUMBER=PASTE_CHAT_ID_HERE
PORT=8080

# start simple first
AGENT_BACKEND=echo
AGENT_BACKENDS=echo

# optional but recommended
TELEGRAM_WEBHOOK_SECRET=change-me-to-a-random-string

# logging
LOG_LEVEL=info
LOG_VERBOSE=false

# where runtime data is stored
DATA_DIR=data
EOF
```

important:
- start with `echo` backend first (no external backend dependency)
- `echo` is only a dummy smoke-test backend: it just replies with `echo: <your message>`
- we switch to `pi` later after basic flow works

---

## 6) first local run (smoke test)

```bash
cd /root/code/visor
set -a
source .env
set +a
mkdir -p bin
go build -o bin/visor .
./bin/visor
```

keep it running in this terminal.

open a second terminal and check health:

```bash
curl -s http://127.0.0.1:8080/health
```

expected:

```json
{"status":"ok"}
```

you should also see the visor ascii startup banner in terminal. that's expected.

if this works, stop visor with `ctrl+c`.

---

## 7) expose visor to public https (for telegram webhook)

telegram requires public `https://` URL.

fastest test method: cloudflared quick tunnel.

install cloudflared:

```bash
curl -fsSL https://pkg.cloudflare.com/cloudflare-main.gpg | sudo gpg --dearmor -o /usr/share/keyrings/cloudflare-main.gpg
echo 'deb [signed-by=/usr/share/keyrings/cloudflare-main.gpg] https://pkg.cloudflare.com/cloudflared noble main' | sudo tee /etc/apt/sources.list.d/cloudflared.list
sudo apt update
sudo apt install -y cloudflared
```

start visor again in terminal 1:

```bash
cd /root/code/visor
set -a
source .env
set +a
mkdir -p bin
go build -o bin/visor .
./bin/visor
```

start tunnel in terminal 2:

```bash
cloudflared tunnel --url http://127.0.0.1:8080
```

cloudflared will print a public url like:
- `https://something.trycloudflare.com`

copy that URL.

---

## 8) set telegram webhook

replace values below and run:

```bash
curl -s "https://api.telegram.org/botYOUR_BOT_TOKEN/setWebhook" \
  -d "url=https://YOUR_PUBLIC_HOST/webhook" \
  -d "secret_token=YOUR_TELEGRAM_WEBHOOK_SECRET"
```

check webhook status:

```bash
curl -s "https://api.telegram.org/botYOUR_BOT_TOKEN/getWebhookInfo" | jq
```

if you see errors, fix them before continuing.

branding note:
- official logo file path in repo: `docs/assets/visor-logo.png`

---

## 9) real test

send a message to your bot in telegram.

with `AGENT_BACKEND=echo`, visor should reply immediately with an `echo: ...` message.

if that works: base setup is correct ✅

m12 note (important):
- with `AGENT_BACKEND=echo` there is *no* real guided setup assistant (echo is smoke-test only)
- the interactive m12 setup flow needs a real backend (`pi`)

---

## 10) switch from echo to pi backend

only after step 9 works.

1) install pi cli (follow your pi.dev install/auth flow)
2) verify:

```bash
pi --version
pi auth status
```

3) update `.env`:

```bash
AGENT_BACKEND=pi
AGENT_BACKENDS=pi,echo
```

4) restart visor and test again.

optional (use the m12 guided setup):
- if you prefer conversational onboarding, start visor with a real backend (`pi`) and follow setup prompts
- m12 can now also:
  - validate OpenAI key (`validate_openai`)
  - send final test message + write setup summary

fallback behavior:
- if pi fails and registry is enabled, visor can fall back to `echo`.

---

## 11) run visor as service (so it survives logout)

create systemd unit:

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

logs:

```bash
journalctl -u visor -f
```

---

## 12) common mistakes (and fixes)

### bot does not answer
- webhook not set correctly
- public URL not reachable
- wrong `USER_PHONE_NUMBER` (chat id)

check:

```bash
curl -s "https://api.telegram.org/botYOUR_BOT_TOKEN/getWebhookInfo" | jq
```

### `config: TELEGRAM_BOT_TOKEN is required`
- `.env` missing or not loaded
- typo in key name

### `config: USER_PHONE_NUMBER is required`
- chat id missing in `.env`

### no response but no crash
- check logs:

```bash
journalctl -u visor -n 200 --no-pager
```

---

## 13) optional next steps

- add OpenAI + ElevenLabs keys for voice features
- configure OTEL/SigNoz from `docs/signoz-setup.md`

---

if you want, add your exact repo URL + preferred runtime mode and i can also generate a one-command installer script.
