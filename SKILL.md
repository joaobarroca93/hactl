# hactl skill reference

Quick reference for using hactl as a Claude skill or AI agent tool.

## Setup

```yaml
# ~/.config/hactl/config.yaml
hass_url: http://192.168.1.10:8123
hass_token: <long-lived-access-token>

filter:
  mode: exposed   # default — only Assist-exposed entities visible
  # mode: all     # required for admin operations (expose/unexpose/rename)
```

Run once after setup (and after changing exposed entities in HA):
```bash
hactl sync
```

## Common commands

```bash
hactl summary --plain                          # one-line home digest (great for LLM context)
hactl state get <entity_id>                    # get single entity state
hactl state list --domain light --plain        # list all lights
hactl state list --area "living room" --plain  # filter by area
hactl service call light.turn_on --entity light.living_room --brightness 80
hactl service call switch.toggle --entity switch.fan
hactl automation trigger <automation_id>
hactl history <entity_id> --last 2h --plain
hactl events watch --type state_changed        # live event stream
```

## Admin operations (requires filter.mode: all)

These commands operate directly on the Home Assistant entity registry. They require `filter.mode: all` in `~/.config/hactl/config.yaml` and will fail fast with a clear error if that is not set.

```bash
hactl expose <entity_id>              # expose entity to Assist
hactl unexpose <entity_id>            # hide entity from Assist
hactl rename <entity_id> <name>       # set friendly name
```

### Important notes

- **Prerequisite:** Set `filter.mode: all` in `~/.config/hactl/config.yaml` before running these commands.
- **No sync required before running:** These commands operate directly on the HA registry and do not consult the local cache.
- **Always sync after:** Run `hactl sync` after expose/unexpose/rename to refresh the local entity cache.
- **Entity IDs are read-only:** `hactl rename` sets the friendly display name only. Entity IDs (e.g. `light.living_room`) cannot be changed via hactl — use the Home Assistant UI (Settings → Devices & Services → Entities) for that.

### Examples

```bash
# Expose a new device to Assist, then sync the cache
hactl expose light.new_bedroom_lamp
hactl sync

# Hide a noisy sensor from Assist
hactl unexpose sensor.wifi_signal_strength
hactl sync

# Give a friendly name to an auto-generated entity
hactl rename light.shelly_abc123_channel_1 "Desk Lamp"
hactl sync
```

## Flags

| Flag        | Description                                    |
|-------------|------------------------------------------------|
| `--plain`   | Compact human-readable prose (best for LLMs)   |
| `--quiet`   | Suppress stdout; errors to stderr; exit code only |
| `--config`  | Path to config file                            |

## Error handling

All errors go to stderr with exit code 1. stdout is always clean JSON (or plain text with `--plain`), safe to pipe into `jq` or inject into prompts.
