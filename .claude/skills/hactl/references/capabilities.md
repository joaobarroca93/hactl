# hactl capabilities reference

Complete reference for entity types, available services, and the correct hactl command for each operation.

---

## How to interact with an entity

| Entity type | Read state | Change state |
|---|---|---|
| Hardware (light, switch, climate, …) | `hactl state get` | `hactl service call` |
| Virtual / helper (input_boolean, input_text, …) | `hactl state get` | `hactl state set` |
| Read-only (sensor, binary_sensor, weather, person, …) | `hactl state get` | — not writable |
| Todo lists | `hactl todo list` | `hactl todo add/done/remove` |
| Entity registry (expose, name) | — | `hactl expose` / `hactl unexpose` / `hactl rename` *(requires `filter.mode: all`)* |

`hactl state set` is blocked for hardware domains — it updates HA's internal record but sends no command to the device. Always use `hactl service call` for anything physical.

---

## Entity domains

### light

Dimmable and colour lights.

| Service | Key flags | Notes |
|---|---|---|
| `light.turn_on` | `--entity` | Turns light on |
| `light.turn_off` | `--entity` | Turns light off |
| `light.toggle` | `--entity` | Toggles on/off |

Convenience flags available on `hactl service call`:

| Flag | Type | Description |
|---|---|---|
| `--brightness` | 0–100 | Brightness percentage |
| `--rgb` | R,G,B | Colour as three 0–255 values |
| `--color-temp` | mireds | Colour temperature |

```bash
hactl service call light.turn_on  --entity light.living_room
hactl service call light.turn_on  --entity light.living_room --brightness 60
hactl service call light.turn_on  --entity light.living_room --rgb 255,140,0
hactl service call light.turn_on  --entity light.living_room --color-temp 370
hactl service call light.turn_off --entity light.kitchen
hactl service call light.toggle   --entity light.bedroom
```

---

### switch

Binary switches (plugs, relays, wall switches).

| Service | Notes |
|---|---|
| `switch.turn_on` | Turns switch on |
| `switch.turn_off` | Turns switch off |
| `switch.toggle` | Toggles on/off |

```bash
hactl service call switch.turn_on  --entity switch.office_plug
hactl service call switch.turn_off --entity switch.office_plug
hactl service call switch.toggle   --entity switch.fan
```

---

### climate

Thermostats and HVAC units.

| Service | Key flags | Notes |
|---|---|---|
| `climate.set_temperature` | `--temperature` | Set target temperature |
| `climate.set_hvac_mode` | `--hvac-mode` | Set mode: `heat` `cool` `heat_cool` `auto` `dry` `fan_only` `off` |
| `climate.set_fan_mode` | `--data fan_mode=auto` | Set fan speed |
| `climate.set_humidity` | `--data humidity=50` | Set target humidity |
| `climate.turn_on` | — | Turn on |
| `climate.turn_off` | — | Turn off |

Convenience flags:

| Flag | Description |
|---|---|
| `--temperature` | Target temperature in °C |
| `--hvac-mode` | HVAC mode string |

```bash
hactl service call climate.set_temperature --entity climate.bedroom --temperature 21.5
hactl service call climate.set_hvac_mode   --entity climate.bedroom --hvac-mode heat
hactl service call climate.set_hvac_mode   --entity climate.bedroom --hvac-mode off
hactl service call climate.set_fan_mode    --entity climate.bedroom --data fan_mode=auto
hactl service call climate.turn_off        --entity climate.bedroom
```

Useful state attributes: `current_temperature`, `temperature` (setpoint), `hvac_mode`, `hvac_modes`, `fan_mode`.

---

### cover

Blinds, shutters, curtains, garage doors.

| Service | Notes |
|---|---|
| `cover.open_cover` | Open fully |
| `cover.close_cover` | Close fully |
| `cover.stop_cover` | Stop mid-movement |
| `cover.toggle` | Toggle open/closed |
| `cover.set_cover_position` | Set position 0–100 |
| `cover.set_cover_tilt_position` | Set tilt 0–100 (if supported) |

```bash
hactl service call cover.open_cover          --entity cover.living_room_blinds
hactl service call cover.close_cover         --entity cover.garage_door
hactl service call cover.set_cover_position  --entity cover.living_room_blinds --data position=50
hactl service call cover.stop_cover          --entity cover.living_room_blinds
```

---

### fan

Ceiling fans and stand fans.

| Service | Notes |
|---|---|
| `fan.turn_on` | Turn on |
| `fan.turn_off` | Turn off |
| `fan.toggle` | Toggle on/off |
| `fan.set_percentage` | Speed as 0–100 |
| `fan.set_preset_mode` | Named preset (e.g. `auto`, `sleep`) |
| `fan.oscillate` | Enable/disable oscillation |
| `fan.set_direction` | `forward` or `reverse` |

```bash
hactl service call fan.turn_on        --entity fan.bedroom
hactl service call fan.set_percentage --entity fan.bedroom --data percentage=60
hactl service call fan.set_preset_mode --entity fan.bedroom --data preset_mode=sleep
hactl service call fan.oscillate      --entity fan.bedroom --data oscillating=true
```

---

### media_player

TVs, speakers, streaming devices.

| Service | Notes |
|---|---|
| `media_player.turn_on` | Power on |
| `media_player.turn_off` | Power off |
| `media_player.media_play` | Play |
| `media_player.media_pause` | Pause |
| `media_player.media_stop` | Stop |
| `media_player.media_next_track` | Next track |
| `media_player.media_previous_track` | Previous track |
| `media_player.volume_set` | Volume 0.0–1.0 |
| `media_player.volume_up` | Volume up |
| `media_player.volume_down` | Volume down |
| `media_player.volume_mute` | Toggle mute |
| `media_player.select_source` | Switch input/source |

```bash
hactl service call media_player.turn_on    --entity media_player.living_room_tv
hactl service call media_player.media_pause --entity media_player.living_room_tv
hactl service call media_player.volume_set  --entity media_player.living_room_tv --data volume_level=0.4
hactl service call media_player.select_source --entity media_player.living_room_tv --data source=HDMI1
```

---

### vacuum

Robot vacuums.

| Service | Notes |
|---|---|
| `vacuum.start` | Start cleaning |
| `vacuum.pause` | Pause |
| `vacuum.stop` | Stop |
| `vacuum.return_to_base` | Return to dock |
| `vacuum.locate` | Play locate sound |
| `vacuum.clean_spot` | Clean current spot |

```bash
hactl service call vacuum.start          --entity vacuum.roborock
hactl service call vacuum.return_to_base --entity vacuum.roborock
hactl service call vacuum.locate         --entity vacuum.roborock
```

---

### lock

Door locks and smart padlocks.

| Service | Notes |
|---|---|
| `lock.lock` | Lock |
| `lock.unlock` | Unlock |
| `lock.open` | Open latch (if supported) |

```bash
hactl service call lock.lock   --entity lock.front_door
hactl service call lock.unlock --entity lock.front_door
```

---

### alarm_control_panel

Security alarm systems.

| Service | Notes |
|---|---|
| `alarm_control_panel.alarm_disarm` | Disarm (use `--data code=1234` if required) |
| `alarm_control_panel.alarm_arm_away` | Arm away |
| `alarm_control_panel.alarm_arm_home` | Arm home/stay |
| `alarm_control_panel.alarm_arm_night` | Arm night |
| `alarm_control_panel.alarm_arm_vacation` | Arm vacation |
| `alarm_control_panel.alarm_trigger` | Trigger alarm |

```bash
hactl service call alarm_control_panel.alarm_arm_away --entity alarm_control_panel.home
hactl service call alarm_control_panel.alarm_disarm   --entity alarm_control_panel.home --data code=1234
```

---

### siren

Audible/visual alarms and sirens.

| Service | Notes |
|---|---|
| `siren.turn_on` | Activate |
| `siren.turn_off` | Deactivate |
| `siren.toggle` | Toggle |

```bash
hactl service call siren.turn_on  --entity siren.alarm
hactl service call siren.turn_off --entity siren.alarm
```

---

### automation

Automations defined in HA. Managed via `hactl automation` subcommands or `hactl service call`.

| hactl command | Equivalent service |
|---|---|
| `hactl automation list` | — |
| `hactl automation trigger <id>` | `automation.trigger` |
| `hactl automation enable <id>` | `automation.turn_on` |
| `hactl automation disable <id>` | `automation.turn_off` |

```bash
hactl automation list
hactl automation trigger my_morning_routine
hactl automation enable  automation.night_mode
hactl automation disable automation.night_mode
```

The `automation.` prefix is optional — hactl adds it automatically.

---

### script

Scripts defined in HA.

```bash
hactl service call script.turn_on --entity script.welcome_home
hactl service call script.my_script --data timeout=30 --data mode=fast
```

---

### scene

Scenes activate a preset combination of entity states.

```bash
hactl service call scene.turn_on --entity scene.movie_night
hactl service call scene.turn_on --entity scene.morning
```

---

### button

Momentary push buttons (one-shot trigger, no on/off state).

```bash
hactl service call button.press --entity button.restart_device
```

---

### input_boolean

Virtual on/off toggle. Use `hactl state set` (not service call).

```bash
hactl state set input_boolean.guest_mode on
hactl state set input_boolean.guest_mode off
hactl state get input_boolean.guest_mode
```

---

### input_text

Virtual text field. Use `hactl state set`.

```bash
hactl state set input_text.notes "away until Friday"
hactl state get input_text.notes
```

---

### input_number

Virtual numeric slider/input.

```bash
hactl service call input_number.set_value --entity input_number.target_humidity --data value=55
hactl service call input_number.increment  --entity input_number.target_humidity
hactl service call input_number.decrement  --entity input_number.target_humidity
```

---

### input_select

Virtual dropdown selector.

```bash
hactl service call input_select.select_option   --entity input_select.scene_mode --data option=Evening
hactl service call input_select.select_next     --entity input_select.scene_mode
hactl service call input_select.select_previous --entity input_select.scene_mode
```

---

### input_datetime

Virtual date/time field.

```bash
hactl service call input_datetime.set_datetime --entity input_datetime.alarm_time --data time=07:30:00
hactl service call input_datetime.set_datetime --entity input_datetime.alarm_time --data datetime="2026-03-01 07:30:00"
```

---

### sensor / binary_sensor

Read-only. No services. Use `hactl state get` or `hactl state list`.

```bash
hactl state get sensor.outdoor_temperature
hactl state get binary_sensor.front_door_contact
hactl state list --domain sensor
hactl state list --domain binary_sensor
```

Common sensor attributes: `unit_of_measurement`, `device_class`, `state_class`.

Binary sensor states: `on` (detected/open/active) / `off` (clear/closed/inactive).

---

### todo

Todo lists. Items are fetched via WebSocket; mutations use service calls.

| hactl command | Notes |
|---|---|
| `hactl todo list [entity_id]` | List items. Omit entity_id to list all exposed todo lists |
| `hactl todo add <entity_id> <item>` | Add an item |
| `hactl todo done <entity_id> <item>` | Mark item as completed |
| `hactl todo remove <entity_id> <item>` | Remove an item |

The `todo.` prefix is optional — hactl adds it automatically.

```bash
hactl todo list
hactl todo list shopping_list
hactl todo list todo.shopping_list --plain

hactl todo add shopping_list "Milk"
hactl todo done shopping_list "Milk"
hactl todo remove shopping_list "Milk"
```

Item status values: `needs_action` (pending) / `completed`.

---

### person

Person tracker entities. Read-only — persons are updated by HA integrations, not by hactl.

```bash
hactl person list
hactl person list --plain
# → Alice: home
```

State values: `home`, `not_home`, or the name of a zone.

---

### weather

Weather entities. Read-only. Current conditions plus forecast (if supplied by the integration).

```bash
hactl weather                            # auto-selects first exposed weather entity
hactl weather weather.forecast_home
hactl weather --plain
# → sunny, 21.5°C, humidity 60%, wind 12.0 km/h; forecast: Wed rainy 22/14, Thu sunny 20
```

Useful attributes: `temperature`, `humidity`, `wind_speed`, `temperature_unit`, `wind_speed_unit`, `forecast`.

> **Note:** the `forecast` attribute is populated by many integrations (Met.no, OpenWeatherMap, etc.). If it is absent, the weather command still shows current conditions.

---

### notify

Send notifications to devices or notification services. `--entity` is **not required** — the service name itself is the target.

| Service | Notes |
|---|---|
| `notify.notify` | Broadcast to all configured notification services |
| `notify.mobile_app_<device>` | Target a specific mobile device |
| `notify.persistent_notification` | Create a persistent notification in the HA UI |

```bash
# Discover available notify services
hactl service list --domain notify --plain

# Send to all devices
hactl service call notify.notify --data title="Alert" --data message="Hello"

# Send to a specific phone
hactl service call notify.mobile_app_my_phone --data title="Alert" --data message="Hello"
```

---

### auth

Authentication management. Does not require an existing token to be configured.

| Command | Notes |
|---|---|
| `hactl auth login` | Interactive prompt for HA URL and token; validates and writes config |
| `hactl auth whoami` | Show HA version and URL for the current token |
| `hactl auth check` | Exit 0 if token is valid, exit 1 if not configured or unauthorized |

`auth check` is silent on success (exit 0) and writes one line to stderr on failure (exit 1) — intended for pre-flight checks in scripts and agents:

```bash
hactl auth check || exit 1
```

`auth whoami` respects `--plain` and `--quiet`:

```bash
hactl auth whoami --plain
# → Home Assistant 2024.12.0 · http://192.168.1.10:8123

hactl auth whoami
# → { "version": "2024.12.0", "hass_url": "http://192.168.1.10:8123" }
```

---

### expose / unexpose / rename (admin)

Commands that write directly to the Home Assistant entity registry. **Require `filter.mode: all`** in `~/.config/hactl/config.yaml` — they fail immediately with a clear error if `exposed` mode is active.

| Command | Notes |
|---|---|
| `hactl expose <entity_id>` | Mark entity as exposed to HA Assist |
| `hactl unexpose <entity_id>` | Hide entity from HA Assist |
| `hactl rename <entity_id> <name>` | Set the friendly display name |

Always run `hactl sync` after any of these to refresh the local entity cache.

`hactl rename` sets the display name only — entity IDs cannot be changed via hactl.

```bash
# Requires filter.mode: all in config
hactl expose light.new_bedroom_lamp
hactl sync

hactl unexpose sensor.wifi_signal_strength
hactl sync

hactl rename light.shelly_abc123_channel_1 "Desk Lamp"
hactl sync
```

---


### homeassistant (system services)

System-wide services. Most work in both filter modes; only `restart` and `stop` require `filter.mode: all`.

| Service | Notes |
|---|---|
| `homeassistant.restart` | Restart HA — **restricted to `filter.mode: all`** |
| `homeassistant.stop` | Stop HA — **restricted to `filter.mode: all`** |
| `homeassistant.check_config` | Validate config files — works in exposed mode |
| `homeassistant.reload_all` | Reload all YAML config — works in exposed mode |
| `homeassistant.turn_on` | Turn on any entity by ID |
| `homeassistant.turn_off` | Turn off any entity by ID |
| `homeassistant.toggle` | Toggle any entity by ID |

`homeassistant.turn_on/off/toggle` are cross-domain — they work on any entity type and are not subject to domain-mismatch checks.

```bash
# Requires filter.mode: all in config
hactl service call homeassistant.restart

# Works in both filter modes
hactl service call homeassistant.check_config
hactl service call homeassistant.reload_all

# Cross-domain turn on/off (works in both modes)
hactl service call homeassistant.turn_off --entity light.living_room
hactl service call homeassistant.toggle   --entity switch.fan
```

---

## Reading state

### Get a single entity

```bash
hactl state get light.living_room
hactl state get climate.bedroom --plain
hactl state get sensor.outdoor_temperature | jq '.state'
```

### List entities

```bash
hactl state list                         # all exposed entities
hactl state list --domain light          # filter by domain
hactl state list --domain sensor
hactl state list --area garagem          # filter by area_id (from hactl area list)
hactl state list --domain switch --plain
```

### Summary digest

```bash
hactl summary                            # JSON digest across all domains
hactl summary --plain                    # one-line prose, ideal for LLM context
hactl summary --area "garagem"
```

### History

```bash
hactl history sensor.outdoor_temperature --last 24h
hactl history light.living_room --last 1h --plain
# → "on at 08:32, off at 09:15, on at 14:20 (still on)"
```

### List available services

```bash
hactl service list                   # all domains, JSON
hactl service list --domain notify   # filter by domain
hactl service list --domain notify --plain
# → notify.mobile_app_my_phone
# → notify.notify
# → notify.persistent_notification
```

### Areas

```bash
hactl area list
hactl area list --plain
# → Garage (id=garagem)
# → Living Room (id=sala)
```

Use the `id=` value with `--area` flags.

---

## Entity filter

| `filter.mode` | Behaviour |
|---|---|
| `exposed` (default) | Only entities exposed to HA Assist are accessible |
| `all` | All entities accessible; restricted services unblocked |

Hidden and non-existent entities return the same error — callers cannot distinguish between them:

```
error: entity not found: light.hidden
```

Run `hactl sync` to update the local cache after changing which entities are exposed in HA Assist. The cache also stores entity→area mappings used by `--area` filtering.

---

## Service call flags reference

| Flag | Type | Applicable services |
|---|---|---|
| `--entity` | entity_id | Any service targeting a specific entity |
| `--brightness` | 0–100 | `light.turn_on` |
| `--rgb` | R,G,B | `light.turn_on` |
| `--color-temp` | mireds | `light.turn_on` |
| `--temperature` | float °C | `climate.set_temperature` |
| `--hvac-mode` | string | `climate.set_hvac_mode` |
| `--data key=value` | any | Any service — for parameters without a dedicated flag |

`--data` is repeatable:

```bash
hactl service call notify.mobile_app_phone \
  --data title="Alert" \
  --data message="Motion detected"
```

---

## Output formats

| Flag | Stdout | Best for |
|---|---|---|
| *(default)* | Indented JSON | `jq` pipelines, structured data |
| `--plain` | Compact prose | Injecting into LLM prompts |
| `--quiet` | Nothing (exit code only) | Shell scripting |

```bash
# jq pipeline
hactl state list --domain light | jq '.[] | select(.state == "on") | .entity_id'

# LLM context injection
hactl summary --plain

# Scripting
hactl service call switch.turn_off --entity switch.printer --quiet && echo "off"
```
