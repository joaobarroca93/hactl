---
name: hactl
description: >
  Control and query a Home Assistant instance using the hactl CLI. Use this
  skill whenever the user wants to interact with their smart home — turning
  lights on or off, adjusting the thermostat, checking locks, managing blinds,
  reading sensors, triggering automations, sending notifications, managing todo
  lists, or getting a home status summary. Trigger this skill whenever hactl,
  Home Assistant, HA, or any smart-home device or entity is mentioned, even if
  the user just asks "are the lights on?" or "is the front door locked?".
compatibility:
  tools: [Bash]
  requires: hactl CLI installed and configured (HASS_URL + HASS_TOKEN)
---

# hactl skill

hactl is a fast, single-binary CLI for Home Assistant, designed for AI agents,
scripts, and developers. It enforces an entity allowlist so agents can only see
and act on entities the user has explicitly permitted.

## Before you act — orient first

When the user asks about home state or wants to do something, start by
orienting yourself:

```bash
hactl summary --plain        # quick digest of all domains
hactl area list --plain      # available areas (use area ids with --area)
```

If the task is scoped to a specific domain or area:

```bash
hactl state list --domain light --plain
hactl state list --area "living room" --plain
```

This avoids acting on stale assumptions and surfaces the actual entity IDs
you'll need.

## The core rule: what kind of entity?

Before issuing a command, ask: is this entity hardware, a virtual helper, or
read-only?

| Entity type | Read | Change |
|---|---|---|
| Hardware (light, switch, climate, cover, fan, lock, media_player, vacuum, …) | `hactl state get` | `hactl service call <domain>.<action> --entity <id>` |
| Virtual helper (input_boolean, input_text, input_number, input_select, input_datetime, …) | `hactl state get` | `hactl state set <id> <value>` |
| Read-only (sensor, binary_sensor, weather, person) | `hactl state get` | — not writable |
| Todo lists | `hactl todo list` | `hactl todo add/done/remove` |
| Automations | `hactl automation list` | `hactl automation trigger/enable/disable` |

**Never use `hactl state set` for hardware entities** — it updates HA's
internal record but sends no command to the device. Use `hactl service call`.

## Output format

Always use `--plain` when injecting output into your reasoning or the
conversation — it's compact prose optimised for LLM context:

```bash
hactl summary --plain
# → 3 lights on (living room 80%, bedroom 40%), heating 21°C, front door locked

hactl state get climate.bedroom --plain
# → heat mode, target 21°C, actual 19.5°C

hactl history sensor.outdoor_temperature --last 6h --plain
# → 12.1 at 14:00, 11.4 at 16:00, 10.2 at 18:00 (still 10.2)
```

Use JSON (default) when you need to pipe into `jq` or extract specific fields:

```bash
hactl state list --domain light | jq '.[] | select(.state == "on") | .entity_id'
```

## Common patterns

### Lights

```bash
hactl service call light.turn_on  --entity light.living_room
hactl service call light.turn_on  --entity light.living_room --brightness 60
hactl service call light.turn_on  --entity light.living_room --rgb 255,140,0
hactl service call light.turn_on  --entity light.living_room --color-temp 370
hactl service call light.turn_off --entity light.kitchen
hactl service call light.toggle   --entity light.bedroom
```

### Climate / thermostat

```bash
hactl service call climate.set_temperature --entity climate.bedroom --temperature 21.5
hactl service call climate.set_hvac_mode   --entity climate.bedroom --hvac-mode heat
hactl service call climate.set_hvac_mode   --entity climate.bedroom --hvac-mode off
```

### Switches

```bash
hactl service call switch.turn_on  --entity switch.office_plug
hactl service call switch.turn_off --entity switch.office_plug
hactl service call switch.toggle   --entity switch.fan
```

### Locks

```bash
hactl service call lock.lock   --entity lock.front_door
hactl service call lock.unlock --entity lock.front_door
```

### Covers (blinds, garage doors)

```bash
hactl service call cover.open_cover         --entity cover.living_room_blinds
hactl service call cover.close_cover        --entity cover.garage_door
hactl service call cover.set_cover_position --entity cover.blinds --data position=50
```

### Notifications

`notify` services don't take `--entity` — the service name is the target.
Discover available targets first:

```bash
hactl service list --domain notify --plain
# → notify.mobile_app_my_phone
# → notify.persistent_notification

hactl service call notify.mobile_app_my_phone --data title="Alert" --data message="Hello"
hactl service call notify.persistent_notification --data title="Info" --data message="Done"
```

### Todo lists

```bash
hactl todo list                           # all exposed todo lists
hactl todo list shopping_list --plain
hactl todo add shopping_list "Milk"
hactl todo done shopping_list "Milk"
hactl todo remove shopping_list "Milk"
```

### Automations

```bash
hactl automation list --plain
hactl automation trigger my_morning_routine
hactl automation enable  automation.night_mode
hactl automation disable automation.night_mode
```

### Scenes and scripts

```bash
hactl service call scene.turn_on  --entity scene.movie_night
hactl service call script.turn_on --entity script.welcome_home
hactl service call script.my_script --data timeout=30 --data mode=fast
```

### Extra parameters

For parameters without a dedicated flag, use `--data key=value` (repeatable):

```bash
hactl service call fan.set_percentage --entity fan.bedroom --data percentage=60
hactl service call notify.mobile_app_phone \
  --data title="Alert" \
  --data message="Motion detected"
```

## After a service call

hactl polls the entity state after a call and returns the settled value, so
you can trust the output. If you want to confirm silently in a script:

```bash
hactl service call light.turn_off --entity light.bedroom --quiet && echo "done"
```

## Errors to watch for

```
error: entity not found: light.xyz          # entity is hidden or doesn't exist
error: switch entities are controlled via services
  use: hactl service call switch.turn_on/off --entity switch.fan
error: domain mismatch: service light.turn_on cannot target a switch entity
  did you mean: hactl service call switch.turn_on --entity switch.fan
error: service homeassistant.restart is not permitted in exposed mode
error: unauthorized: check your HASS_TOKEN
error: connection refused: dial tcp ...     # HA unreachable
```

The entity filter intentionally makes hidden entities look identical to
non-existent ones — this is by design.

## History and events

```bash
hactl history light.living_room --last 1h --plain
hactl history sensor.temperature --last 24h

hactl events watch --type state_changed --domain light   # live stream
```

## Setup reference

```bash
# First-time setup
hactl sync    # fetch all exposed entities into the local cache

# Re-sync after exposing/hiding entities in HA Assist
hactl sync
```

Config lives at `~/.config/hactl/config.yaml`. Set `HASS_URL` and
`HASS_TOKEN` env vars or put them in the config file.

## Detailed domain reference

For full per-domain service listings (cover, fan, media_player, vacuum, alarm,
input_*, etc.) read `references/capabilities.md` in this skill directory.
