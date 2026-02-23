# hactl

hactl is a fast, single-binary CLI for Home Assistant. Built for scripting, AI agents, and developers who'd rather type than click.

## Install

```bash
go install github.com/joaobarroca/hactl@latest
```

Or build from source:

```bash
git clone https://github.com/joaobarroca/hactl
cd hactl
go build -o hactl .
sudo mv hactl /usr/local/bin/
```

## Configuration

hactl reads config from environment variables (preferred) or `~/.config/hactl/config.yaml`.

| Env var      | Config key   | Default                          | Description                  |
|--------------|--------------|----------------------------------|------------------------------|
| `HASS_URL`   | `hass_url`   | `http://homeassistant.local:8123`| Home Assistant base URL      |
| `HASS_TOKEN` | `hass_token` | *(required)*                     | Long-lived access token      |

### Config file example

```yaml
# ~/.config/hactl/config.yaml
hass_url: http://192.168.1.10:8123
hass_token: eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9...
```

### Getting a token

In Home Assistant: **Profile → Long-Lived Access Tokens → Create Token**

## Global flags

| Flag       | Description                                    |
|------------|------------------------------------------------|
| `--plain`  | Compact human-readable prose (great for LLMs)  |
| `--quiet`  | Suppress all output except errors (scripting)  |
| `--config` | Path to config file                            |

## Commands

### state

```bash
# Get a single entity state
hactl state get light.living_room
hactl state get climate.bedroom --plain

# Set a state directly
hactl state set input_boolean.guest_mode on

# List all states
hactl state list
hactl state list --domain light
hactl state list --domain sensor
hactl state list --area "living room"
```

### service

```bash
# Call any service
hactl service call light.turn_on --entity light.living_room
hactl service call light.turn_on --entity light.living_room --brightness 80
hactl service call light.turn_on --entity light.living_room --rgb 255,128,0
hactl service call light.turn_off --entity light.kitchen

hactl service call climate.set_temperature --entity climate.bedroom --temperature 21.0
hactl service call climate.set_hvac_mode --entity climate.bedroom --hvac-mode heat

hactl service call switch.toggle --entity switch.fan

# Extra key=value pairs
hactl service call script.my_script --data timeout=30 --data mode=fast

# Restart HA
hactl service call homeassistant.restart
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
error: unauthorized: check your HASS_TOKEN
error: connection refused: dial tcp 192.168.1.10:8123: connect: connection refused
```

## AI agent usage

hactl is optimised for use by AI agents. Recommended patterns:

```bash
# Get a full picture of the home
hactl summary --plain

# Check a specific room before acting
hactl state list --area "living room" --plain

# Act and confirm
hactl service call light.turn_on --entity light.living_room --brightness 60 --plain

# Watch for changes after an action
hactl events watch --type state_changed --domain light
```
