# hactl

hactl is a fast, single-binary CLI for Home Assistant. Built for scripting, AI agents, and developers who'd rather type than click.

## Install

```bash
go install github.com/joaobarroca93/hactl@latest
```

Or build from source:

```bash
git clone https://github.com/joaobarroca93/hactl
cd hactl
go build -o hactl .
sudo mv hactl /usr/local/bin/
```

## Claude Code skill

hactl ships with a Claude Code skill so you can control your home directly from a Claude conversation. Once installed, Claude knows how to use hactl — which commands to run, when to read state versus call a service, how to orient itself before acting, and how to handle errors.

**Install the skill:**

```bash
npx skills add github.com/joaobarroca93/hactl
```

After that, prompts like "turn off the kitchen lights" or "what's the temperature in the bedroom?" will automatically invoke hactl. Make sure hactl is configured first (`HASS_URL`, `HASS_TOKEN`, and `hactl sync` done).

**If you work on hactl itself** using Claude Code, the skill loads automatically from `.claude/skills/hactl/` whenever you open the project — no extra steps needed.

## Configuration

hactl reads config from environment variables (preferred for secrets) or `~/.config/hactl/config.yaml`.

| Env var      | Config key     | Default                          | Description                  |
|--------------|----------------|----------------------------------|------------------------------|
| `HASS_URL`   | `hass_url`     | `http://homeassistant.local:8123`| Home Assistant base URL      |
| `HASS_TOKEN` | `hass_token`   | *(required)*                     | Long-lived access token      |
| —            | `filter.mode`  | `exposed`                        | Entity filter mode (see below)|

### Config file example

```yaml
# ~/.config/hactl/config.yaml
hass_url: http://192.168.1.10:8123
hass_token: eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9...

filter:
  # "exposed" — only entities exposed to HA Assist (default, recommended)
  # "all"     — no filter (must be set explicitly)
  mode: exposed
```

### Getting a token

In Home Assistant: **Profile → Long-Lived Access Tokens → Create Token**

## Entity filter

hactl enforces an entity allowlist so that AI agents and scripts can only see and act on the entities you have explicitly permitted.

**Default behaviour (`filter.mode: exposed`):** only entities exposed to [HA Assist](https://www.home-assistant.io/voice_control/voice_remote_expose_devices/) are visible. All other entities appear as if they do not exist — `hactl` returns the same `entity not found` error whether an entity is hidden by the filter or genuinely absent from Home Assistant. This prevents callers from probing your entity namespace.

**`filter.mode: all`:** no filter is applied. All entities are accessible. This must be set explicitly in the config file; there is no CLI flag or environment variable to override filter mode at runtime.

### First-time setup

After configuring your token, sync the allowlist once:

```bash
hactl sync
```

This connects to Home Assistant, fetches all entities exposed to HA Assist, and writes the list to `~/.config/hactl/exposed-entities.json`. Re-run `hactl sync` any time you change which entities are exposed in HA Assist.

## Global flags

| Flag       | Description                                    |
|------------|------------------------------------------------|
| `--plain`  | Compact human-readable prose (great for LLMs)  |
| `--quiet`  | Suppress all output except errors (scripting)  |
| `--config` | Path to config file                            |

## Commands

### sync

Fetch all entities exposed to HA Assist and write the local entity cache. Run this once on first setup, and again whenever you expose or hide entities in Home Assistant.

```bash
hactl sync
# Synced 42 exposed entities to ~/.config/hactl/exposed-entities.json
```

### area

```bash
hactl area list
hactl area list --plain
```

### state

```bash
# Get a single entity state
hactl state get light.living_room
hactl state get climate.bedroom --plain

# List all states
hactl state list
hactl state list --domain light
hactl state list --domain sensor
hactl state list --area "living room"

# Set state — only for virtual/helper entities (input_boolean, input_text, etc.)
hactl state set input_boolean.guest_mode on
hactl state set input_text.notes "away until Friday"

# Hardware-backed entities (light, switch, climate, …) must use service call instead
hactl state set light.living_room on  # and will raise an error
```

### service

#### service call

After a successful call, hactl polls the entity state until it reflects the change (up to 3 seconds), so the output always shows the settled state rather than a stale snapshot.

The following errors are caught before the call is made:
- **Missing `--entity`**: entity-domain services (light, switch, climate, etc.) require `--entity`; passthrough domains (`notify`, `homeassistant`, `tts`, …) do not
- **Domain mismatch**: `light.turn_on --entity switch.fan` is rejected — service and entity domains must match (`homeassistant.*` is exempt)
- **Restricted services**: `homeassistant.restart` and `homeassistant.stop` are blocked in `filter.mode: exposed` (the default). Set `filter.mode: all` in your config to allow them.

```bash
hactl service call light.turn_on --entity light.living_room
hactl service call light.turn_on --entity light.living_room --brightness 80
hactl service call light.turn_on --entity light.living_room --rgb 255,128,0
hactl service call light.turn_off --entity light.kitchen

hactl service call climate.set_temperature --entity climate.bedroom --temperature 21.0
hactl service call climate.set_hvac_mode --entity climate.bedroom --hvac-mode heat

hactl service call switch.toggle --entity switch.fan

# Extra key=value pairs
hactl service call script.my_script --data timeout=30 --data mode=fast

# Notify — no --entity needed; service name is the target
hactl service call notify.notify --data title="Hello" --data message="World"
hactl service call notify.mobile_app_my_phone --data title="Hello" --data message="World"

# Restart HA — requires filter.mode: all in config
hactl service call homeassistant.restart
```

#### service list

Lists all services available in Home Assistant. Useful for discovering notify targets, automation actions, and integration-specific services.

```bash
hactl service list                   # all domains, JSON
hactl service list --domain notify   # filter by domain
hactl service list --domain notify --plain
# → notify.mobile_app_my_phone
# → notify.notify
# → notify.persistent_notification
```

### expose / unexpose / rename

Admin commands that write directly to the Home Assistant entity registry. They require `filter.mode: all` in `~/.config/hactl/config.yaml` and will fail immediately if that is not set.

```bash
# Expose an entity to HA Assist, then refresh the local cache
hactl expose light.new_bedroom_lamp
hactl sync

# Hide an entity from HA Assist
hactl unexpose sensor.wifi_signal_strength
hactl sync

# Set the friendly display name of an entity
hactl rename light.shelly_abc123_channel_1 "Desk Lamp"
hactl sync
```

> **Note:** `hactl rename` sets the friendly display name only. Entity IDs (e.g. `light.living_room`) are read-only from hactl — use the Home Assistant UI to change them.


### todo

Manage Home Assistant todo lists. Items are fetched via the HA REST service API.

```bash
# List all exposed todo lists
hactl todo list
hactl todo list --plain

# List a specific list
hactl todo list shopping_list
hactl todo list todo.shopping_list

# Add an item
hactl todo add shopping_list "Milk"
hactl todo add todo.shopping_list "Eggs"

# Mark an item as done
hactl todo done shopping_list "Milk"

# Remove an item
hactl todo remove shopping_list "Milk"
```

### person

Read person location states (home/away/zone name). Read-only.

```bash
hactl person list
hactl person list --plain
# → Alice: home
```

### weather

Show current conditions and forecast from a weather entity.

```bash
hactl weather                              # auto-selects first exposed weather entity
hactl weather weather.forecast_home
hactl weather --plain
# → sunny, 21.5°C, humidity 60%, wind 12.0 km/h; forecast: Wed rainy 22/14, Thu sunny 20
```

### automation

```bash
hactl automation list
hactl automation list --plain

hactl automation trigger my_morning_routine
hactl automation trigger automation.my_morning_routine

hactl automation enable my_morning_routine
hactl automation disable my_morning_routine
```

### history

```bash
# Last hour (default)
hactl history light.living_room

# Custom window
hactl history sensor.temperature --last 24h
hactl history binary_sensor.front_door --last 2h

# Compact prose output
hactl history light.living_room --plain
# → "on at 08:32, off at 09:15, on at 14:20 (still on)"
```

### summary

Aggregates current state across all domains into a single digest. Highlights unusual conditions (lights on during day, doors unlocked, temperatures out of range).

```bash
hactl summary
hactl summary --area "living room"

# One-sentence digest, ideal for injecting into LLM context
hactl summary --plain
# → "3 lights on (living room 80%, bedroom 40%), heating 21°C, front door locked, motion in hallway 4m ago"
```

### events

Stream live events from Home Assistant WebSocket API as JSON lines.

```bash
# All events
hactl events watch

# Filter by type
hactl events watch --type state_changed

# Filter by domain
hactl events watch --domain light
hactl events watch --domain motion

# Compact output
hactl events watch --type state_changed --plain
```

### Shell completion

```bash
# Bash
hactl completion bash > /etc/bash_completion.d/hactl

# Zsh
hactl completion zsh > "${fpath[1]}/_hactl"

# Fish
hactl completion fish > ~/.config/fish/completions/hactl.fish
```

## Output formats

### JSON (default)

All commands output valid JSON to stdout, making them easy to pipe into `jq`:

```bash
hactl state get light.living_room | jq '.state'
hactl state list --domain light | jq '.[] | select(.state == "on") | .entity_id'
hactl summary | jq '.domains[] | select(.domain == "climate")'
```

### Plain text (`--plain`)

Compact human-readable prose optimised for injecting into LLM prompts:

```bash
hactl summary --plain
# 2 lights on (kitchen 100%, hallway 60%), thermostat heat 21°C (actual 19.5°C), front door locked

hactl history sensor.outdoor_temperature --last 24h --plain
# 8.2 at 00:00, 6.1 at 04:30, 12.4 at 10:15 (still 12.4)
```

### Quiet mode (`--quiet`)

Suppresses all stdout; only errors go to stderr. Exit code 0 on success, 1 on failure:

```bash
hactl service call light.turn_off --entity light.bedroom --quiet && echo "done"
```

## Errors

Errors always go to stderr with exit code 1:

```
error: entity not found: light.nonexistent
error: switch entities are controlled via services
  use: hactl service call switch.turn_on/off --entity switch.fan
  services: switch.turn_on / switch.turn_off / switch.toggle
error: domain mismatch: service light.turn_on cannot target a switch entity
  did you mean: hactl service call switch.turn_on --entity switch.fan
error: service homeassistant.restart is not permitted in exposed mode
  to enable it, set filter.mode: all in your config file
error: unexpected status 400: Bad Request
error: unauthorized: check your HASS_TOKEN
error: connection refused: dial tcp 192.168.1.10:8123: connect: connection refused
```

## AI agent usage

hactl is optimised for use by AI agents. Recommended setup:

```bash
# 1. Expose entities to HA Assist in the Home Assistant UI, then:
hactl sync

# 2. Give the agent read-only hactl access — only exposed entities are visible.
```

Recommended patterns:

```bash
# Get a full picture of the home (filtered to exposed entities)
hactl summary --plain

# Check a specific room before acting
hactl state list --area "living room" --plain

# Act and confirm
hactl service call light.turn_on --entity light.living_room --brightness 60 --plain

# Watch for changes after an action
hactl events watch --type state_changed --domain light
```

The entity filter means agents cannot enumerate or interact with entities you have not explicitly exposed — hidden entities and non-existent entities return the same error, preventing probing.
