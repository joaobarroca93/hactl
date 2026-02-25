# hactl capability testing plan

This document provides a structured set of manual tests to verify every capability
described in `CAPABILITIES.md` against a real Home Assistant setup.

**Before you start**, replace the placeholder entity IDs below with real ones from
your HA setup. Every section begins with a "Discovery" step that shows you how to
find suitable entity IDs.

---

## Progress tracker

Check off each test as you complete it. Use `[x]` for pass, `[~]` for skip, `[!]` for fail.

**Prerequisites**
- [x] 0.1 Environment setup
- [x] 0.2 Sync exposed entities
- [x] 0.3 Audit log setup
- [ ] 0.4 No-token guard

**1. State — read**
- [x] 1.1 List all entities (JSON)
- [x] 1.2 List by domain
- [x] 1.3 List by area
- [x] 1.4 Get single entity (JSON)
- [x] 1.5 Get single entity (plain)
- [ ] 1.6 jq pipeline
- [x] 1.7 Hidden / non-existent entity

**2. State — set**
- [~] 2.1 input_boolean — no exposed entities
- [~] 2.2 input_text — no exposed entities
- [~] 2.3 Plain output on set — no exposed entities
- [~] 2.4 Quiet mode on set — no exposed entities
- [x] 2.5 Blocked on hardware domains

**3. Service calls**
- [x] 3.1 light
- [x] 3.2 switch
- [~] 3.3 climate — no exposed entities
- [ ] 3.4 cover
- [ ] 3.5 fan
- [ ] 3.6 media_player
- [x] 3.7 vacuum
- [ ] 3.8 lock
- [ ] 3.9 alarm_control_panel
- [ ] 3.10 siren
- [ ] 3.11 scene
- [ ] 3.12 script
- [ ] 3.13 button
- [ ] 3.14 input_number
- [ ] 3.15 input_select
- [ ] 3.16 input_datetime
- [x] 3.17 Notify service (no --entity required)
- [ ] 3.18 Domain mismatch guard
- [ ] 3.19 Entity-required domain without --entity
- [ ] 3.20 homeassistant cross-domain

**3b. Service list**
- [ ] 3b.1 List all services (JSON)
- [ ] 3b.2 Filter by domain
- [ ] 3b.3 Plain output
- [ ] 3b.4 Unknown domain returns empty list

**4. Restricted services**
- [ ] 4.1 Blocked in exposed mode
- [ ] 4.2 Allowed in all mode

**5. Automations**
- [ ] 5.1 List
- [ ] 5.2 Trigger
- [ ] 5.3 Disable and enable

**6. Todo lists**
- [ ] 6.1 Discovery
- [ ] 6.2 List a specific todo list
- [ ] 6.3 Add, complete, and remove an item
- [ ] 6.4 Plain and quiet output

**7. History**
- [ ] 7.1 JSON output
- [ ] 7.2 Plain output
- [ ] 7.3 No history in window
- [ ] 7.4 Invalid duration

**8. Summary**
- [ ] 8.1 Full JSON summary
- [ ] 8.2 Plain summary
- [ ] 8.3 Area-filtered summary
- [ ] 8.4 Alert detection

**9. Areas**
- [ ] 9.1 JSON listing
- [ ] 9.2 Plain listing

**10. Persons**
- [ ] 10.1 JSON listing
- [ ] 10.2 Plain listing

**11. Weather**
- [ ] 11.1 Auto-select first weather entity
- [ ] 11.2 Explicit entity
- [ ] 11.3 Plain output
- [ ] 11.4 No weather entity

**12. Events stream** *(manual — run in separate terminal)*
- [ ] 12.1 Stream all events
- [ ] 12.2 Filter by event type
- [ ] 12.3 Filter by domain
- [ ] 12.4 Combined filters
- [ ] 12.5 Plain output

**13. Output format consistency**
- [ ] 13 Format checks

**14. Entity filter modes**
- [ ] 14.1 Exposed mode: hidden entity blocked
- [ ] 14.2 All mode: hidden entity accessible
- [ ] 14.3 Restore exposed mode

**15. Sync**
- [ ] 15.1 Cache update after exposing
- [ ] 15.2 Cache update after hiding
- [ ] 15.3 Area mapping update

**16. Admin commands**
- [ ] 16.1 Guard: blocked in exposed mode
- [ ] 16.2 Setup: switch to all mode
- [ ] 16.3 unexpose
- [ ] 16.4 expose
- [ ] 16.5 rename: set friendly name
- [ ] 16.6 rename: entity ID unchanged
- [ ] 16.7 rename: restore original name
- [ ] 16.8 Quiet and plain output
- [ ] 16.9 Invalid entity ID
- [ ] 16.10 Restore exposed mode

**17. Edge cases**
- [ ] 17 Edge cases

**18. Security audit**
- [ ] 18.1 Token not in logs
- [ ] 18.2 Hidden entities not in exposed output
- [ ] 18.3 Identical error for hidden vs non-existent
- [ ] 18.4 Admin commands blocked in exposed mode
- [ ] 18.5 Restricted services blocked
- [ ] 18.6 State set blocked for hardware domains
- [ ] 18.7 All mode does not persist
- [ ] 18.8 Log summary

---

## Prerequisites

### 0.1 — Environment setup

```bash
# Set the required env vars (or put them in ~/.config/hactl/config.yaml)
export HASS_URL=http://<your-ha-host>:8123
export HASS_TOKEN=<your-long-lived-access-token>
```

Expected: no error output; all subsequent commands succeed.

### 0.2 — Sync exposed entities

```bash
hactl sync
```

Expected:

- Prints `Synced N exposed entities to ~/.config/hactl/exposed-entities.json`
- Prints `Synced N entity→area mappings to ~/.config/hactl/entity-areas.json`
- Both cache files exist on disk.

### 0.3 — Audit log setup

All test output should be saved to a dated log directory so results can be
reviewed later for unexpected data exposure or security regressions.

```bash
# Create a timestamped directory for this test session
export HACTL_TEST_LOG=$(pwd)/hactl-test-logs/$(date +%Y%m%d_%H%M%S)
mkdir -p "$HACTL_TEST_LOG"
: > "$HACTL_TEST_LOG/session.log"   # cumulative log — everything appends here
echo "Session started: $(date)" >> "$HACTL_TEST_LOG/session.log"

# Helper: run a hactl command, print it, log both stdout and stderr with a label.
# - Per-label files (.out / .err / .exit) hold the latest result for that label.
# - session.log is append-only and captures every command in order.
# Usage: run <label> <hactl args...>
run() {
  local label="$1"; shift
  local outfile="$HACTL_TEST_LOG/${label}.out"
  local errfile="$HACTL_TEST_LOG/${label}.err"
  local session="$HACTL_TEST_LOG/session.log"
  printf '\n==> %s: hactl %s\n' "$label" "$*" | tee -a "$session"
  hactl "$@" > "$outfile" 2>"$errfile"
  local rc=$?
  cat "$outfile"              # show stdout on terminal
  cat "$outfile" >> "$session"
  if [[ -s "$errfile" ]]; then
    echo "[stderr]" | tee -a "$session"
    cat "$errfile" | tee -a "$session"
  fi
  echo "[exit: $rc]" | tee -a "$session"
  echo "$rc" > "$HACTL_TEST_LOG/${label}.exit"
  return $rc
}
```

Use `run <label> <args>` throughout your session instead of bare `hactl` calls.
Labels become filenames, so use short snake_case names (e.g. `state_list`,
`expose_guard`).

> **Token safety:** `HASS_TOKEN` must never appear in output files. The helper
> above does not log env vars, but verify manually before sharing any logs:
>
> ```bash
> grep -r "$HASS_TOKEN" "$HACTL_TEST_LOG" && echo "TOKEN FOUND IN LOGS — review before sharing"
> ```

### 0.4 — No-token guard

```bash
unset HASS_TOKEN
hactl state list
export HASS_TOKEN=<your-token>   # restore for the rest of the tests
```

Expected: `error: HASS_TOKEN is required. Set it via the HASS_TOKEN environment variable or hass_token in config.yaml`

---

## 1. State — read

### 1.1 — List all exposed entities (JSON)

```bash
hactl state list
```

Expected:

- JSON array printed to stdout.
- Every object has `entity_id`, `state`, and `attributes` fields.
- Only entities you have exposed to HA Assist appear.

### 1.2 — List by domain

```bash
hactl state list --domain light
hactl state list --domain sensor
hactl state list --domain binary_sensor
hactl state list --domain switch
```

Expected: each command returns only entities whose `entity_id` starts with the
given domain prefix.

### 1.3 — List by area

```bash
# First, find area ids
hactl area list --plain
# Pick an area_id (e.g. "sala") then:
hactl state list --area sala
```

Expected: only entities assigned to that area appear.

### 1.4 — Get a single entity (JSON)

```bash
hactl state get light.<your_light>
hactl state get sensor.<your_sensor>
hactl state get binary_sensor.<your_binary_sensor>
```

Expected: a single JSON object with `entity_id`, `state`, `attributes`, and
`last_changed`.

### 1.5 — Get a single entity (plain)

```bash
hactl state get light.<your_light> --plain
hactl state get climate.<your_climate> --plain
```

Expected: compact single line, e.g. `light.living_room: on (brightness 60%)`.

### 1.6 — jq pipeline

```bash
hactl state list --domain light | jq '.[] | select(.state == "on") | .entity_id'
```

Expected: entity IDs of all lights currently on, one per line.

### 1.7 — Hidden / non-existent entity

```bash
hactl state get light.this_entity_does_not_exist
```

Expected: `error: entity not found: light.this_entity_does_not_exist` (exit code non-zero).

---

## 2. State — set (virtual helpers only)

### 2.1 — input_boolean

```bash
# Discovery
hactl state list --domain input_boolean

# Test
hactl state set input_boolean.<your_boolean> on
hactl state get input_boolean.<your_boolean>    # verify state is "on"

hactl state set input_boolean.<your_boolean> off
hactl state get input_boolean.<your_boolean>    # verify state is "off"
```

Expected: state changes reflected immediately.

### 2.2 — input_text

```bash
# Discovery
hactl state list --domain input_text

# Test
hactl state set input_text.<your_text> "testing hactl"
hactl state get input_text.<your_text>    # verify value
```

Expected: state is the string you set.

### 2.3 — Plain output on set

```bash
hactl state set input_boolean.<your_boolean> on --plain
```

Expected: `input_boolean.<id> set to on`

### 2.4 — Quiet mode on set

```bash
hactl state set input_boolean.<your_boolean> off --quiet
echo "exit: $?"
```

Expected: no stdout, exit code 0.

### 2.5 — Blocked: state set on hardware domains

For each of these, the command must fail with a helpful error:

```bash
hactl state set light.<your_light> on
hactl state set switch.<your_switch> on
hactl state set climate.<your_climate> heat
hactl state set cover.<your_cover> open
hactl state set fan.<your_fan> on
hactl state set media_player.<your_media_player> on
hatml state set vacuum.<your_vacuum> cleaning
hactl state set lock.<your_lock> unlocked
hactl state set alarm_control_panel.<your_alarm> armed_away
hactl state set siren.<your_siren> on
hactl state set button.<your_button> on
hactl state set scene.<your_scene> on
hactl state set script.<your_script> on
```

Expected for each: `error: <domain> entities are controlled via services` with a
hint listing the correct service call.

---

## 3. Service calls

### 3.1 — light

**Discovery:**

```bash
hactl state list --domain light --plain
```

Replace `light.<id>` with a real entity from your setup.

```bash
# Turn on
hactl service call light.turn_on --entity light.<id>
hactl state get light.<id> --plain   # verify: on

# Turn off
hactl service call light.turn_off --entity light.<id>
hactl state get light.<id> --plain   # verify: off

# Toggle
hactl service call light.toggle --entity light.<id>
```

Convenience flags (use a dimmable colour bulb):

```bash
# Brightness (0–100 → converted to 0–255 internally)
hactl service call light.turn_on --entity light.<id> --brightness 60

# RGB colour
hactl service call light.turn_on --entity light.<id> --rgb 255,140,0

# Colour temperature in mireds
hactl service call light.turn_on --entity light.<id> --color-temp 370
```

Expected: lights respond physically; `state get` after the call reflects the new state.

### 3.2 — switch

**Discovery:** `hactl state list --domain switch --plain`

```bash
hactl service call switch.turn_on  --entity switch.<id>
hactl state get switch.<id> --plain   # verify: on

hactl service call switch.turn_off --entity switch.<id>
hactl state get switch.<id> --plain   # verify: off

hactl service call switch.toggle   --entity switch.<id>
```

### 3.3 — climate

**Discovery:** `hactl state list --domain climate --plain`

```bash
# Set temperature
hactl service call climate.set_temperature --entity climate.<id> --temperature 21.5

# Set HVAC mode
hactl service call climate.set_hvac_mode --entity climate.<id> --hvac-mode heat
hactl service call climate.set_hvac_mode --entity climate.<id> --hvac-mode off

# Set fan mode via --data
hactl service call climate.set_fan_mode --entity climate.<id> --data fan_mode=auto

# Turn on / off
hactl service call climate.turn_on  --entity climate.<id>
hactl service call climate.turn_off --entity climate.<id>

# Verify attributes
hactl state get climate.<id> | jq '{mode: .attributes.hvac_mode, setpoint: .attributes.temperature, current: .attributes.current_temperature}'
```

### 3.4 — cover

**Discovery:** `hactl state list --domain cover --plain`

```bash
hactl service call cover.open_cover  --entity cover.<id>
hactl service call cover.close_cover --entity cover.<id>
hactl service call cover.set_cover_position --entity cover.<id> --data position=50
hactl service call cover.stop_cover  --entity cover.<id>
hactl service call cover.toggle      --entity cover.<id>

# Tilt (if supported by your cover)
hactl service call cover.set_cover_tilt_position --entity cover.<id> --data tilt_position=45
```

### 3.5 — fan

**Discovery:** `hactl state list --domain fan --plain`

```bash
hactl service call fan.turn_on  --entity fan.<id>
hactl service call fan.turn_off --entity fan.<id>
hactl service call fan.toggle   --entity fan.<id>

hactl service call fan.set_percentage  --entity fan.<id> --data percentage=60
hactl service call fan.set_preset_mode --entity fan.<id> --data preset_mode=sleep
hactl service call fan.oscillate       --entity fan.<id> --data oscillating=true
hactl service call fan.set_direction   --entity fan.<id> --data direction=forward
```

### 3.6 — media_player

**Discovery:** `hactl state list --domain media_player --plain`

```bash
hactl service call media_player.turn_on  --entity media_player.<id>
hactl service call media_player.media_play   --entity media_player.<id>
hactl service call media_player.media_pause  --entity media_player.<id>
hactl service call media_player.media_stop   --entity media_player.<id>
hactl service call media_player.media_next_track     --entity media_player.<id>
hactl service call media_player.media_previous_track --entity media_player.<id>

hactl service call media_player.volume_set  --entity media_player.<id> --data volume_level=0.4
hactl service call media_player.volume_up   --entity media_player.<id>
hactl service call media_player.volume_down --entity media_player.<id>
hactl service call media_player.volume_mute --entity media_player.<id> --data is_volume_muted=true

# Source selection — replace HDMI1 with a source your device supports
hactl service call media_player.select_source --entity media_player.<id> --data source=HDMI1

hactl service call media_player.turn_off --entity media_player.<id>
```

### 3.7 — vacuum

**Discovery:** `hactl state list --domain vacuum --plain`

```bash
hactl service call vacuum.start          --entity vacuum.roborock_qrevo_curvx
hactl service call vacuum.pause          --entity vacuum.roborock_qrevo_curvx
hactl service call vacuum.stop           --entity vacuum.roborock_qrevo_curvx
hactl service call vacuum.return_to_base --entity vacuum.roborock_qrevo_curvx
hactl service call vacuum.locate         --entity vacuum.roborock_qrevo_curvx
hactl service call vacuum.clean_spot     --entity vacuum.roborock_qrevo_curvx
```

### 3.8 — lock

**Discovery:** `hactl state list --domain lock --plain`

> **Caution:** test with a lock you can physically verify and re-lock immediately.

```bash
hactl service call lock.lock   --entity lock.<id>
hactl state get lock.<id> --plain   # verify: locked

hactl service call lock.unlock --entity lock.<id>
hactl state get lock.<id> --plain   # verify: unlocked

hactl service call lock.lock   --entity lock.<id>   # re-lock when done
```

### 3.9 — alarm_control_panel

**Discovery:** `hactl state list --domain alarm_control_panel --plain`

> **Caution:** use a test/simulator alarm or ensure you can disarm immediately.

```bash
hactl service call alarm_control_panel.alarm_arm_home    --entity alarm_control_panel.<id>
hactl service call alarm_control_panel.alarm_disarm      --entity alarm_control_panel.<id> --data code=1234
hactl service call alarm_control_panel.alarm_arm_away    --entity alarm_control_panel.<id>
hactl service call alarm_control_panel.alarm_disarm      --entity alarm_control_panel.<id> --data code=1234
hactl service call alarm_control_panel.alarm_arm_night   --entity alarm_control_panel.<id>
hactl service call alarm_control_panel.alarm_disarm      --entity alarm_control_panel.<id> --data code=1234
```

### 3.10 — siren

**Discovery:** `hactl state list --domain siren --plain`

```bash
hactl service call siren.turn_on  --entity siren.<id>
hactl service call siren.turn_off --entity siren.<id>
hactl service call siren.toggle   --entity siren.<id>
```

### 3.11 — scene

**Discovery:** `hactl state list --domain scene --plain`

```bash
hactl service call scene.turn_on --entity scene.<id>
```

Expected: devices in the scene change to their preset states.

### 3.12 — script

**Discovery:** `hactl state list --domain script --plain`

```bash
hactl service call script.turn_on --entity script.<id>

# With extra data
hactl service call script.<id> --data mode=fast
```

### 3.13 — button

**Discovery:** `hactl state list --domain button --plain`

```bash
hactl service call button.press --entity button.<id>
```

Expected: button action fires in HA (check logbook if unsure).

### 3.14 — input_number

**Discovery:** `hactl state list --domain input_number --plain`

```bash
hactl service call input_number.set_value --entity input_number.<id> --data value=55
hactl state get input_number.<id> --plain   # verify: 55

hactl service call input_number.increment --entity input_number.<id>
hactl service call input_number.decrement --entity input_number.<id>
```

### 3.15 — input_select

**Discovery:** `hactl state list --domain input_select --plain`

```bash
# List available options first
hactl state get input_select.<id> | jq '.attributes.options'

hactl service call input_select.select_option   --entity input_select.<id> --data option=<valid_option>
hactl service call input_select.select_next     --entity input_select.<id>
hactl service call input_select.select_previous --entity input_select.<id>
```

### 3.16 — input_datetime

**Discovery:** `hactl state list --domain input_datetime --plain`

```bash
# Time only
hactl service call input_datetime.set_datetime --entity input_datetime.<id> --data time=07:30:00

# Full datetime
hactl service call input_datetime.set_datetime --entity input_datetime.<id> --data "datetime=2026-03-01 07:30:00"
```

### 3.17 — Notify service (no --entity required)

First, discover the available notify service names:

```bash
hactl service list --domain notify --plain
```

Then send a notification — use the exact slug from the output above:

```bash
# Broadcast to all
hactl service call notify.notify --data title="hactl test" --data message="Testing"

# Target a specific device (replace with slug from service list)
hactl service call notify.mobile_<device> \
  --data title="hactl test" \
  --data message="Testing repeatable data flags"
```

Expected: notification arrives on the device(s); exit 0.

### 3.18 — Domain mismatch guard

```bash
# light entity targeted with a switch service
hactl service call switch.turn_on --entity light.<your_light>
```

Expected: `error: domain mismatch: service switch.turn_on cannot target a light entity` with a `did you mean:` hint.

### 3.19 — Entity-required domain without --entity

```bash
hactl service call light.turn_on
hactl service call switch.toggle
hactl service call climate.set_temperature
```

Expected: `error: service <domain>.<svc> requires --entity` with usage hint. Exit 1.

### 3.20 — homeassistant cross-domain services (allowed in both modes)

```bash
hactl service call homeassistant.turn_off --entity light.<your_light>
hactl state get light.<your_light> --plain   # verify: off

hactl service call homeassistant.turn_on  --entity light.<your_light>
hactl service call homeassistant.toggle   --entity switch.<your_switch>
```

Expected: no domain-mismatch error; devices respond.

---

## 3b. Service list

### 3b.1 — List all services (JSON)

```bash
hactl service list
```

Expected: JSON array where each element has `domain` (string) and `services` (array of strings). Exit 0.

### 3b.2 — Filter by domain

```bash
hactl service list --domain notify
```

Expected: JSON array with a single element whose `domain` is `"notify"` and `services` contains at least `"notify"`. Exit 0.

### 3b.3 — Plain output

```bash
hactl service list --domain notify --plain
```

Expected: one `notify.<service_name>` per line, no JSON. Exit 0.

### 3b.4 — Unknown domain returns empty list

```bash
hactl service list --domain this_domain_does_not_exist
```

Expected: empty JSON array `[]`, exit 0 (not an error — domain simply has no services).

---

## 4. Restricted services (homeassistant.restart / stop)

### 4.1 — Blocked in exposed mode (default)

```bash
# Ensure config has no filter.mode override, or set filter.mode: exposed
hactl service call homeassistant.restart
```

Expected: `error: service homeassistant.restart is not permitted in exposed mode` with hint to set `filter.mode: all`.

### 4.2 — Allowed in all mode

Add `filter.mode: all` to `~/.config/hactl/config.yaml`, then:

```bash
hactl service call homeassistant.check_config
```

Expected: success (JSON or plain response from HA). Do **not** test `restart`/`stop`
unless you are certain you want HA to restart.

Restore `filter.mode: exposed` (or remove the key) after this test.

---

## 5. Automations

### 5.1 — List

```bash
hactl automation list
hactl automation list --plain
```

Expected: JSON array (or plain text) with `entity_id`, `state` (on/off),
`friendly_name`, and `last_triggered`.

### 5.2 — Trigger

```bash
# Discovery: pick an automation ID from the list above
hactl automation trigger automation.<id>

# Without prefix (prefix added automatically)
hactl automation trigger <id_without_prefix>
```

Expected: automation runs; plain output `triggered automation.<id>`.

### 5.3 — Disable and enable

```bash
hactl automation disable automation.<id>
hactl automation list --plain   # verify: <name> [off]

hactl automation enable automation.<id>
hactl automation list --plain   # verify: <name> [on]
```

---

## 6. Todo lists

### 6.1 — Discovery

```bash
hactl todo list
```

Expected: all exposed todo lists with their items (JSON or plain).

### 6.2 — List a specific todo list

```bash
# With prefix
hactl todo list todo.<list_id>

# Without prefix (added automatically)
hactl todo list <list_id>

# Plain output
hactl todo list <list_id> --plain
```

Expected: plain output uses `[ ]` and `[x]` status markers.

### 6.3 — Add, complete, and remove an item

```bash
hactl todo add <list_id> "hactl test item"
hactl todo list <list_id> --plain   # verify item appears with [ ]

hactl todo done <list_id> "hactl test item"
hactl todo list <list_id> --plain   # verify item shows [x]

hactl todo remove <list_id> "hactl test item"
hactl todo list <list_id> --plain   # verify item is gone
```

### 6.4 — Plain and quiet output

```bash
hactl todo add <list_id> "quiet test" --quiet
echo "exit: $?"   # should be 0, no stdout

hactl todo add <list_id> "plain test" --plain
# Expected: added "plain test" to todo.<list_id>

hactl todo remove <list_id> "quiet test" --quiet
hactl todo remove <list_id> "plain test" --quiet
```

---

## 7. History

### 7.1 — JSON output

```bash
hactl history sensor.<your_sensor> --last 1h
hactl history light.<your_light> --last 24h
```

Expected: JSON array of history entries with `entity_id`, `state`, and
`last_changed` fields.

### 7.2 — Plain output

```bash
hactl history light.<your_light> --last 1h --plain
```

Expected: compact prose, e.g. `on at 08:32, off at 09:15, on at 14:20 (still on)`.

### 7.3 — No history in window

```bash
hactl history sensor.<rarely_changing_sensor> --last 1m --plain
```

Expected: `no history for <entity> in the last 1m` (or an empty JSON array in JSON mode).

### 7.4 — Invalid duration

```bash
hactl history sensor.<your_sensor> --last 2days
```

Expected: `error: invalid --last value "2days": use values like 1h, 30m, 24h`

---

## 8. Summary

### 8.1 — Full JSON summary

```bash
hactl summary
```

Expected: JSON object with `generated_at`, `domains` array, and optionally `alerts`.
Each domain entry has `domain`, `total`, `active`, `inactive`, and `entities`.

### 8.2 — Plain summary

```bash
hactl summary --plain
```

Expected: single-line prose, e.g.
`lights: 2 on (Living Room 60%, Kitchen), 3 off (...), Thermostat heat setpoint 21.5°C (actual 20.3°C)`

### 8.3 — Area-filtered summary

```bash
hactl area list --plain    # find an area id
hactl summary --area <area_id>
hactl summary --area <area_id> --plain
```

Expected: only entities assigned to that area appear in the digest.

### 8.4 — Alert detection

Verify alerts are surfaced correctly:

- **Light on during daytime** (07:00–21:00): turn a light on and run `hactl summary`.
JSON `alerts` array and `--plain` output should mention it.
- **Unlocked lock**: if you have a lock entity, unlock it and run `hactl summary`.
Expect a `lock open:` alert.
- **Unusual climate setpoint**: set temperature below 16 or above 26 °C, then run
`hactl summary`. Expect an unusual-temperature alert.

---

## 9. Areas

### 9.1 — JSON listing

```bash
hactl area list
```

Expected: JSON array with `area_id` and `name` fields.

### 9.2 — Plain listing

```bash
hactl area list --plain
```

Expected: one line per area: `Living Room (id=sala)`

---

## 10. Persons

### 10.1 — JSON listing

```bash
hactl person list
```

Expected: JSON array with `entity_id`, `friendly_name`, `state`.
State is `home`, `not_home`, or a zone name.

### 10.2 — Plain listing

```bash
hactl person list --plain
```

Expected: `Joao Barroca: home` (one line per person).

---

## 11. Weather

### 11.1 — Auto-select first weather entity (JSON)

```bash
hactl weather
```

Expected: JSON with `entity_id`, `condition`, `temperature`, `humidity`,
`wind_speed`, and optionally `forecast`.

### 11.2 — Explicit entity

```bash
hactl weather weather.<your_weather_entity>

# Without prefix (added automatically)
hactl weather <your_weather_entity_name>
```

### 11.3 — Plain output

```bash
hactl weather --plain
```

Expected: single line, e.g.
`sunny, 21.5°C, humidity 60%, wind 12.0 km/h; forecast: Mon rainy 22/14, Tue sunny 20`

### 11.4 — No weather entity

If you temporarily remove the weather entity from exposed entities (run `hactl sync`
after removing it from Assist), then:

```bash
hactl weather
```

Expected: `error: no weather entity found`

---

## 12. Events stream

### 12.1 — Stream all events

```bash
# Run in one terminal, trigger some HA actions in another
hactl events watch
```

Expected: JSON lines appear for each HA event. Press Ctrl-C to stop.

### 12.2 — Filter by event type

```bash
notifyhactl events watch --type state_changed
```

Expected: only `state_changed` events appear. Trigger a state change (e.g. toggle a switch) to verify.

### 12.3 — Filter by domain

```bash
hactl events watch --domain light
```

Expected: only events for `light.*` entities appear. Toggle a light to verify.

### 12.4 — Combined filters

```bash
hactl events watch --type state_changed --domain motion
```

Expected: only `state_changed` events for `motion.*` entities.

### 12.5 — Plain output

```bash
hactl events watch --plain --type state_changed
```

Expected: compact lines like `light.living_room: off -> on`

---

## 13. Output format consistency

Run these checks across at least one representative command per group:


| Check                            | Command                                                                   | Expected            |
| -------------------------------- | ------------------------------------------------------------------------- | ------------------- |
| Default JSON is valid            | `hactl state list                                                         | jq .`               |
| `--plain` is single-line prose   | `hactl summary --plain`                                                   | no JSON brackets    |
| `--quiet` produces no stdout     | `hactl state get sensor.<id> --quiet && echo ok`                          | only `ok` printed   |
| `--quiet` exit code 0 on success | `hactl service call switch.turn_on --entity switch.<id> --quiet; echo $?` | `0`                 |
| Non-zero exit on error           | `hactl state get light.nonexistent; echo $?`                              | non-zero (e.g. `1`) |


---

## 14. Entity filter modes

### 14.1 — Exposed mode (default): hidden entity blocked

Identify an entity that is **not** exposed to HA Assist, then:

```bash
hactl state get <hidden_entity>
```

Expected: `error: entity not found: <hidden_entity>` — indistinguishable from
a non-existent entity.

### 14.2 — All mode: hidden entity accessible

```bash
# Add filter.mode: all to ~/.config/hactl/config.yaml
hactl state get <hidden_entity>
```

Expected: the entity's full state JSON is returned.

### 14.3 — Restore exposed mode after the above test

---

## 15. hactl sync (detailed)

### 15.1 — Cache update after exposing a new entity

1. In HA Assist settings, expose a new entity.
2. Run `hactl sync`.
3. Confirm the entity now appears in `hactl state list`.

### 15.2 — Cache update after hiding an entity

1. In HA Assist settings, hide an entity that was previously visible.
2. Run `hactl sync`.
3. Confirm the entity now returns `error: entity not found` from `hactl state get`.

### 15.3 — Area mapping update

1. Reassign an entity to a different area in HA.
2. Run `hactl sync`.
3. Confirm `hactl state list --area <new_area>` includes the entity,
  and `--area <old_area>` no longer does.

---

## 16. Admin commands — expose / unexpose / rename

These commands require `filter.mode: all` in `~/.config/hactl/config.yaml`.
Use a non-critical entity for testing (e.g. a sensor or helper) to avoid
accidentally hiding something important.

### 16.1 — Guard: blocked in exposed mode

```bash
# Ensure filter.mode is "exposed" (or not set — it defaults to exposed)
hactl expose sensor.<any_sensor>
hactl unexpose sensor.<any_sensor>
hactl rename sensor.<any_sensor> "Test Name"
```

Expected for each:

```
error: this command requires filter.mode: all in ~/.config/hactl/config.yaml
       these are admin operations — set filter.mode: all to proceed
```

### 16.2 — Setup: switch to all mode

Add `filter.mode: all` to `~/.config/hactl/config.yaml` before the tests below.
Restore `filter.mode: exposed` when done with this section.

### 16.3 — unexpose: hide an entity from Assist

Pick an entity that is currently exposed:

```bash
# Confirm it is visible before the test
hactl state get sensor.<your_sensor>   # should succeed

hactl unexpose sensor.<your_sensor>
# Expected: Entity 'sensor.<your_sensor>' is now hidden from Assist.
#           Run 'hactl sync' to update the local cache.

hactl sync
hactl state get sensor.<your_sensor>   # should now return: error: entity not found
```

### 16.4 — expose: re-expose the entity

```bash
hactl expose sensor.<your_sensor>
# Expected: Entity 'sensor.<your_sensor>' is now exposed to Assist.
#           Run 'hactl sync' to update the local cache.

hactl sync
hactl state get sensor.<your_sensor>   # should succeed again
```

### 16.5 — rename: set a friendly name

```bash
# Record the current friendly_name
hactl state get sensor.<your_sensor> | jq '.attributes.friendly_name'

hactl rename sensor.<your_sensor> "hactl Test Sensor"
# Expected: Entity 'sensor.<your_sensor>' renamed to 'hactl Test Sensor'.
#           Run 'hactl sync' to update the local cache.

hactl sync
hactl state get sensor.<your_sensor> | jq '.attributes.friendly_name'
# Expected: "hactl Test Sensor"
```

### 16.6 — rename: entity ID is unchanged

```bash
hactl state get sensor.<your_sensor> | jq '.entity_id'
```

Expected: the original entity ID — rename only changes the display name.

### 16.7 — rename: restore original name

```bash
hactl rename sensor.<your_sensor> "<original_friendly_name>"
hactl sync
```

### 16.8 — Quiet and plain output

```bash
hactl expose sensor.<your_sensor> --quiet
echo "exit: $?"   # should be 0, no stdout

hactl unexpose sensor.<your_sensor> --plain 2>/dev/null || true
# (plain is not a special format for these commands — output is the same prose line)
```

### 16.9 — Invalid entity ID

```bash
hactl expose light.entity_that_does_not_exist
```

Expected: HA returns an error (entity not in registry); hactl prints `error: <HA error message>`.

### 16.10 — Restore exposed mode

Set `filter.mode: exposed` (or remove the key) in `~/.config/hactl/config.yaml`
and run `hactl sync` to confirm everything is back to normal.

---

## 17. Edge cases


| Scenario                                     | Command                                                                  | Expected                                                                          |
| -------------------------------------------- | ------------------------------------------------------------------------ | --------------------------------------------------------------------------------- |
| `--data` without `=`                         | `hactl service call light.turn_on --entity light.<id> --data brightness` | `error: --data must be in key=value format`                                       |
| `--rgb` wrong component count                | `hactl service call light.turn_on --entity light.<id> --rgb 255,128`     | silently ignored (3 parts not present); or verify JSON payload has no `rgb_color` |
| Service without entity where entity required | `hactl service call light.turn_on`                                       | error with hint                                                                   |
| `hactl state list --area nonexistent`        | no entities match                                                        | returns empty JSON array `[]` or empty plain output                               |
| `hactl history` with invalid entity          | `hactl history light.nonexistent --last 1h`                              | `error: entity not found: light.nonexistent`                                      |
| `hactl todo list` with no exposed todo lists | depends on setup                                                         | `no todo lists found` message                                                     |
| `hactl automation trigger nonexistent`       | `hactl automation trigger nonexistent_automation`                        | `error: entity not found: automation.nonexistent_automation`                      |
| `hactl expose` in exposed mode               | `hactl expose light.<id>`                                                | `error: this command requires filter.mode: all …`                                 |
| `hactl expose` unknown entity (all mode)     | `hactl expose light.does_not_exist`                                      | error from HA entity registry                                                     |
| `hactl rename` without sync                  | rename then check `hactl summary --plain`                                | friendly name not updated until `hactl sync` is run                               |


---

## 18. Security audit review

Run this section after completing all other tests. The goal is to detect
unexpected data exposure, filter bypasses, or information leakage in the
captured logs.

### 18.1 — Token not present in any log file

```bash
grep -r "$HASS_TOKEN" "$HACTL_TEST_LOG"
```

Expected: **no matches**. If the token appears anywhere in stdout/stderr, it is
a critical leak — open an issue.

### 18.2 — Hidden entities do not appear in exposed mode output

```bash
# Collect all entity_ids that appeared in state list output
jq -r '.[].entity_id' "$HACTL_TEST_LOG/state_list.out" | sort > /tmp/observed_entities.txt

# Compare against the local exposed cache
jq -r '.[]' ~/.config/hactl/exposed-entities.json | sort > /tmp/expected_entities.txt

diff /tmp/expected_entities.txt /tmp/observed_entities.txt
```

Expected: **no extra lines** in `/tmp/observed_entities.txt`. Any entity that
appears in the state list but not in the exposed cache is a filter bypass.

### 18.3 — Error messages are indistinguishable for hidden vs non-existent entities

```bash
# Logs from tests 1.7 (non-existent) and 14.1 (hidden but real)
cat "$HACTL_TEST_LOG/state_get_nonexistent.err"
cat "$HACTL_TEST_LOG/state_get_hidden.err"
```

Expected: both lines read `error: entity not found: <entity_id>`. If one
reveals more detail (e.g. "entity hidden" or "access denied"), that is a
probing vector — open an issue.

### 18.4 — Admin commands blocked in exposed mode

```bash
cat "$HACTL_TEST_LOG/expose_guard.err"
cat "$HACTL_TEST_LOG/unexpose_guard.err"
cat "$HACTL_TEST_LOG/rename_guard.err"
```

Expected: all three contain `error: this command requires filter.mode: all`.
If any succeeded without `filter.mode: all` set, that is a privilege escalation
— open an issue.

### 18.5 — Restricted services blocked in exposed mode

```bash
cat "$HACTL_TEST_LOG/restart_blocked.err"
```

Expected: `error: service homeassistant.restart is not permitted in exposed mode`.
If it succeeded or returned a different error, open an issue.

### 18.6 — State set blocked for hardware domains

Verify the `.err` files from test 2.5 for each blocked domain:

```bash
for f in "$HACTL_TEST_LOG"/state_set_blocked_*.err; do
  echo "--- $f ---"
  cat "$f"
done
```

Expected: every file contains `error: <domain> entities are controlled via services`.
Any domain that did not error means `serviceControlled` is missing an entry —
open an issue.

### 18.7 — All mode does not persist after tests

```bash
grep "filter" ~/.config/hactl/config.yaml
```

Expected: `mode: exposed` (or the key is absent, which defaults to exposed).
If `mode: all` is still set after the test session, restore it immediately.

### 18.8 — Log summary

```bash
echo "=== Test session: $HACTL_TEST_LOG ==="
echo "Total log files: $(ls "$HACTL_TEST_LOG" | wc -l)"
echo "Non-zero exits:"
grep -l "^[^0]" "$HACTL_TEST_LOG"/*.exit 2>/dev/null | sed 's/\.exit//'
echo "Stderr output present:"
for f in "$HACTL_TEST_LOG"/*.err; do [[ -s "$f" ]] && echo "  $f"; done
```

Review any unexpected non-zero exits or unexpected stderr output against the
"Expected" notes in each test section above.