#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
EXAMPLE_ENV="$ROOT_DIR/.env.example"
TARGET_ENV="$ROOT_DIR/.env"

if [[ ! -f "$EXAMPLE_ENV" ]]; then
  echo "error: missing $EXAMPLE_ENV"
  exit 1
fi

if [[ ! -f "$TARGET_ENV" ]]; then
  cp "$EXAMPLE_ENV" "$TARGET_ENV"
  echo "created .env from .env.example"
fi

WORK_ENV="$(mktemp)"
cp "$TARGET_ENV" "$WORK_ENV"

get_value() {
  local file="$1"
  local key="$2"
  local line
  line="$(grep -E "^${key}=" "$file" | tail -n 1 || true)"
  if [[ -z "$line" ]]; then
    echo ""
    return
  fi
  echo "${line#*=}"
}

upsert_value() {
  local file="$1"
  local key="$2"
  local value="$3"

  if grep -qE "^${key}=" "$file"; then
    awk -v key="$key" -v value="$value" '
      BEGIN { done = 0 }
      {
        if (!done && $0 ~ "^" key "=") {
          print key "=" value
          done = 1
        } else {
          print $0
        }
      }
    ' "$file" >"${file}.tmp"
    mv "${file}.tmp" "$file"
  else
    printf "\n%s=%s\n" "$key" "$value" >>"$file"
  fi
}

mask() {
  local v="$1"
  if [[ -z "$v" ]]; then
    echo "(empty)"
    return
  fi
  if (( ${#v} <= 4 )); then
    echo "****"
    return
  fi
  echo "****${v: -4}"
}

prompt() {
  local key="$1"
  local label="$2"
  local fallback="$3"
  local secret="${4:-false}"
  local input

  if [[ "$secret" == "true" ]]; then
    read -r -s -p "$label [$fallback]: " input
    echo
  else
    read -r -p "$label [$fallback]: " input
  fi

  if [[ -z "$input" ]]; then
    input="$fallback"
  fi

  echo "$input"
}

echo ""
echo "visor setup wizard"
echo "fills .env interactively (key-based update, no blind overwrite)"
echo ""

example_token="$(get_value "$EXAMPLE_ENV" "TELEGRAM_BOT_TOKEN")"
example_chat="$(get_value "$EXAMPLE_ENV" "USER_PHONE_NUMBER")"
example_backend="$(get_value "$EXAMPLE_ENV" "AGENT_BACKEND")"
example_tz="$(get_value "$EXAMPLE_ENV" "TZ")"
example_openai="$(get_value "$EXAMPLE_ENV" "OPENAI_API_KEY")"

current_token="$(get_value "$TARGET_ENV" "TELEGRAM_BOT_TOKEN")"
current_chat="$(get_value "$TARGET_ENV" "USER_PHONE_NUMBER")"
current_backend="$(get_value "$TARGET_ENV" "AGENT_BACKEND")"
current_tz="$(get_value "$TARGET_ENV" "TZ")"
current_openai="$(get_value "$TARGET_ENV" "OPENAI_API_KEY")"
current_obsidian_vault_path="$(get_value "$TARGET_ENV" "OBSIDIAN_VAULT_PATH")"

telegram_bot_token="$(prompt "TELEGRAM_BOT_TOKEN" "TELEGRAM_BOT_TOKEN" "${current_token:-$example_token}" true)"
user_phone_number="$(prompt "USER_PHONE_NUMBER" "USER_PHONE_NUMBER (telegram chat id)" "${current_chat:-$example_chat}")"

backend_default="${current_backend:-$example_backend}"
if [[ -z "$backend_default" ]]; then
  backend_default="echo"
fi

while true; do
  agent_backend="$(prompt "AGENT_BACKEND" "AGENT_BACKEND (pi|echo)" "$backend_default")"
  if [[ "$agent_backend" == "pi" || "$agent_backend" == "echo" ]]; then
    break
  fi
  echo "please choose exactly 'pi' or 'echo'."
done

set_openai="n"
if [[ -n "$current_openai" ]]; then
  read -r -p "OPENAI_API_KEY currently set. keep it? [Y/n]: " keep_key
  if [[ -z "$keep_key" || "$keep_key" =~ ^[Yy]$ ]]; then
    openai_api_key="$current_openai"
    set_openai="keep"
  else
    set_openai="y"
  fi
else
  read -r -p "set OPENAI_API_KEY now? [y/N]: " set_openai
fi

if [[ "$set_openai" =~ ^[Yy]$ ]]; then
  openai_api_key="$(prompt "OPENAI_API_KEY" "OPENAI_API_KEY" "${example_openai}" true)"
elif [[ "$set_openai" == "keep" ]]; then
  :
else
  openai_api_key=""
fi

tz_default="${current_tz:-$example_tz}"
if [[ -z "$tz_default" ]]; then
  tz_default="Europe/Vienna"
fi
tz_value="$(prompt "TZ" "TZ" "$tz_default")"

obsidian_vault_default="${current_obsidian_vault_path:-/root/obsidian/Sibwax}"
obsidian_vault_path="$(prompt "OBSIDIAN_VAULT_PATH" "OBSIDIAN_VAULT_PATH (for obsidian skills)" "$obsidian_vault_default")"

upsert_value "$WORK_ENV" "TELEGRAM_BOT_TOKEN" "$telegram_bot_token"
upsert_value "$WORK_ENV" "USER_PHONE_NUMBER" "$user_phone_number"
upsert_value "$WORK_ENV" "AGENT_BACKEND" "$agent_backend"
upsert_value "$WORK_ENV" "OPENAI_API_KEY" "$openai_api_key"
upsert_value "$WORK_ENV" "TZ" "$tz_value"
upsert_value "$WORK_ENV" "OBSIDIAN_VAULT_PATH" "$obsidian_vault_path"

echo ""
echo "summary (masked):"
echo "- TELEGRAM_BOT_TOKEN: $(mask "$telegram_bot_token")"
echo "- USER_PHONE_NUMBER: $user_phone_number"
echo "- AGENT_BACKEND: $agent_backend"
echo "- OPENAI_API_KEY: $(mask "$openai_api_key")"
echo "- TZ: $tz_value"
echo "- OBSIDIAN_VAULT_PATH: $obsidian_vault_path"
echo ""

read -r -p "write these values to .env? [y/N]: " confirm
if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
  rm -f "$WORK_ENV"
  echo "aborted. .env not changed."
  exit 0
fi

mv "$WORK_ENV" "$TARGET_ENV"
echo "done. updated $TARGET_ENV"

if [[ -n "$obsidian_vault_path" ]]; then
  mkdir -p "$obsidian_vault_path"/{ideas,logs,forge}
  echo "initialized obsidian folders in $obsidian_vault_path/{ideas,logs,forge}"
fi
