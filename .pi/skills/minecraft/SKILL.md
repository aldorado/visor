---
name: minecraft
description: Use when the user mentions Friday, Minecraft, the bot, or wants to interact with the Minecraft world. Also use when checking on Friday's status, sending him commands, or discussing anything game-related.
user-invocable: false
---

# Minecraft Control

You can talk to Friday (your Minecraft bot) via the MindServer Socket.IO API on `46.225.101.185:8080`.

## How to Connect

Use HTTP long-polling (Socket.IO transport). Use curl or any HTTP tool available.

```bash
# 1. Handshake
HANDSHAKE=$(curl -s 'http://46.225.101.185:8080/socket.io/?EIO=4&transport=polling')
SID=$(echo "$HANDSHAKE" | sed 's/^0//' | npx tsx -e "
const stdin = require('fs').readFileSync('/dev/stdin', 'utf8');
console.log(JSON.parse(stdin).sid);
")

# 2. Namespace connect
curl -s -X POST "http://46.225.101.185:8080/socket.io/?EIO=4&transport=polling&sid=$SID" \
  -H 'Content-Type: text/plain' -d '40'

# 3. Poll for namespace ack + initial state
sleep 0.3
curl -s "http://46.225.101.185:8080/socket.io/?EIO=4&transport=polling&sid=$SID"
```

For more complex interactions, use an inline TypeScript script:

```bash
npx tsx -e "
const BASE = 'http://46.225.101.185:8080';

// Handshake
let r = await fetch(BASE + '/socket.io/?EIO=4&transport=polling');
const text = await r.text();
const sid = JSON.parse(text.slice(1)).sid;

// Namespace connect
await fetch(BASE + '/socket.io/?EIO=4&transport=polling&sid=' + sid, {
  method: 'POST',
  headers: { 'Content-Type': 'text/plain' },
  body: '40',
});

// Poll for initial state
await new Promise(r => setTimeout(r, 300));
r = await fetch(BASE + '/socket.io/?EIO=4&transport=polling&sid=' + sid);
console.log(await r.text());

// Send a command
const msg = JSON.stringify(['send-message', 'Friday', { from: 'Ubik', message: '!stats' }]);
await fetch(BASE + '/socket.io/?EIO=4&transport=polling&sid=' + sid, {
  method: 'POST',
  headers: { 'Content-Type': 'text/plain' },
  body: '42' + msg,
});

// Poll for response
await new Promise(r => setTimeout(r, 2000));
r = await fetch(BASE + '/socket.io/?EIO=4&transport=polling&sid=' + sid);
console.log(await r.text());
"
```

## Sending Commands to Friday

When sending commands, use `"from": "Ubik"` in the message payload:

```json
["send-message", "Friday", {"from": "Ubik", "message": "your message here"}]
```

## Available Commands

Send these as the `message` field:

### Queries (read-only)
- `!stats` -- position, health, hunger, biome, weather, time, modes
- `!inventory` -- current items
- `!nearbyBlocks` -- blocks around Friday
- `!entities` -- nearby mobs and players
- `!craftable` -- available crafting recipes
- `!modes` -- behavior mode settings

### Movement
- `!goToPlayer <name> <dist>` -- move to player
- `!followPlayer <name> <dist>` -- follow player continuously
- `!goToCoordinates <x> <y> <z> <closeness>` -- go to coords
- `!searchForBlock <type> <range>` -- find a block type
- `!searchForEntity <type> <range>` -- find an entity
- `!moveAway <distance>` -- back off
- `!stay` -- stop moving
- `!goToSurface` -- go up to surface

### Items & Crafting
- `!equip <item>` -- equip item
- `!consume <item>` -- eat/drink
- `!givePlayer <name> <item> <count>` -- give items to player
- `!collectBlocks <type>` -- harvest blocks
- `!craftRecipe <recipe>` -- craft something
- `!smeltItem <item>` -- smelt in furnace

### Building & Combat
- `!placeHere <type>` -- place a block
- `!digDown` -- dig below
- `!attack` -- attack nearest mob
- `!attackPlayer <name>` -- attack specific player

### Memory & Goals
- `!rememberHere <name>` -- save current location
- `!goToRememberedPlace <name>` -- go to saved location
- `!goal <description>` -- set a task for Friday
- `!endGoal` -- clear current goal

### Control
- `!stop` -- force stop all actions
- `!stfu` -- silence Friday
- `!restart` -- restart the agent

### Chat
You can also just send normal text and Friday will respond conversationally using the local LLM.

## Parsing Responses

Socket.IO polling responses come in this format:
- `42["event-name", arg1, arg2, ...]` -- event data
- Multiple events can be concatenated in one response
- `bot-output` events contain Friday's responses: `42["bot-output","Friday","message text"]`
- `agents-status` events show connection state: `42["agents-status",[{"name":"Friday","in_game":true,...}]]`

## Tips

- The polling session expires after ~25s of inactivity. For multi-step interactions, keep polling.
- Send a ping `2` to keep the session alive if needed.
- Friday's LLM (Andy-4 micro 1.5B) is small -- keep commands simple and direct.
- If Friday is unresponsive, check if the mindcraft container is running on the server.
- The server IP is `46.225.101.185`, Minecraft port `25565`, MindServer port `8080`.
