# hactl test runner

Agent-executable capability test plan. Execute each code block in sequence
in the **same bash session** — variables defined earlier are available in later blocks.

**Agent instructions:**
1. Run the **Helpers and discovery** block first.
2. Run each numbered section block in order.
3. Run the **Security audit** block last to validate captured logs.
4. Set `HACTL_AUTO=1` before starting to skip all destructive/interactive tests.

Every block prints `[PASS]` / `[FAIL]` / `[SKIP]` inline.
Run the **Summary** block (end of file) after all sections for a final count.

---

## Helpers and discovery

```bash
# ---------------------------------------------------------------------------
# Log directory
# ---------------------------------------------------------------------------
export HACTL_TEST_LOG="$HOME/hactl-test-logs/$(date +%Y%m%d_%H%M%S)"
export HACTL_AUTO="${HACTL_AUTO:-0}"
mkdir -p "$HACTL_TEST_LOG"
: > "$HACTL_TEST_LOG/failures.log"
echo "Session log: $HACTL_TEST_LOG"
echo "Auto mode:   $HACTL_AUTO  (set HACTL_AUTO=1 to skip destructive tests)"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
_pass() { echo "[PASS] $*"; }
_fail() { echo "[FAIL] $*"; echo "[FAIL] $*" >> "$HACTL_TEST_LOG/failures.log"; }

# run <label> <hactl_subcommand_and_args...>
run() {
  local label="$1"; shift
  echo ""
  echo "==> $label: hactl $*"
  hactl "$@" \
    > "$HACTL_TEST_LOG/${label}.out" \
    2>"$HACTL_TEST_LOG/${label}.err"
  echo "$?" > "$HACTL_TEST_LOG/${label}.exit"
  cat "$HACTL_TEST_LOG/${label}.out"
  [[ -s "$HACTL_TEST_LOG/${label}.err" ]] && { echo "[stderr]"; cat "$HACTL_TEST_LOG/${label}.err"; }
}

# assert_exit <expected_code> <label>
assert_exit() {
  local expected="$1" label="$2"
  local actual; actual=$(cat "$HACTL_TEST_LOG/${label}.exit" 2>/dev/null)
  [[ "$actual" == "$expected" ]] \
    && _pass "$label: exit=$expected" \
    || _fail "$label: exit=$actual (expected $expected)"
}

# assert_json <label> '<jq_bool_expression>'
assert_json() {
  local label="$1" expr="$2"
  local r; r=$(jq -r "if ($expr) then \"PASS\" else \"FAIL\" end" \
    "$HACTL_TEST_LOG/${label}.out" 2>/dev/null)
  [[ "$r" == "PASS" ]] \
    && _pass "$label: $expr" \
    || _fail "$label: $expr → $r"
}

# assert_stderr <label> '<grep_pattern>'
assert_stderr() {
  local label="$1" pattern="$2"
  grep -q "$pattern" "$HACTL_TEST_LOG/${label}.err" \
    && _pass "$label: stderr contains '$pattern'" \
    || _fail "$label: stderr missing '$pattern'"
}

# assert_nonempty_stdout <label>
assert_nonempty_stdout() {
  [[ -s "$HACTL_TEST_LOG/${1}.out" ]] \
    && _pass "$1: stdout non-empty" \
    || _fail "$1: stdout empty"
}

# skip_if_unset <VAR_NAME> <section_label>
# Returns 1 (→ skip) if var is empty, 0 (→ continue) otherwise.
skip_if_unset() {
  local var="$1" section="$2"
  [[ -z "${!var}" ]] && { echo "[SKIP] $var not found — skipping $section"; return 1; }
  return 0
}

# confirm <message>
# In HACTL_AUTO=1 mode returns 1 (caller should skip). Otherwise waits for Enter.
confirm() {
  if [[ "$HACTL_AUTO" == "1" ]]; then
    echo "[SKIP] $1 (HACTL_AUTO=1)"
    return 1
  fi
  echo ""
  echo "[HUMAN REQUIRED] $1"
  echo "Press Enter to proceed, or Ctrl-C to abort..."
  read -r
  return 0
}

# ---------------------------------------------------------------------------
# Discovery: pick the first exposed entity of each domain
# ---------------------------------------------------------------------------
_first() {
  hactl state list --domain "$1" 2>/dev/null | jq -r '.[0].entity_id // empty'
}

export LIGHT_ID=$(_first light)
export SWITCH_ID=$(_first switch)
export CLIMATE_ID=$(_first climate)
export COVER_ID=$(_first cover)
export FAN_ID=$(_first fan)
export MEDIA_PLAYER_ID=$(_first media_player)
export VACUUM_ID=$(_first vacuum)
export LOCK_ID=$(_first lock)
export ALARM_ID=$(_first alarm_control_panel)
export SIREN_ID=$(_first siren)
export SCENE_ID=$(_first scene)
export SCRIPT_ID=$(_first script)
export BUTTON_ID=$(_first button)
export SENSOR_ID=$(_first sensor)
export BINARY_SENSOR_ID=$(_first binary_sensor)
export INPUT_BOOLEAN_ID=$(_first input_boolean)
export INPUT_TEXT_ID=$(_first input_text)
export INPUT_NUMBER_ID=$(_first input_number)
export INPUT_SELECT_ID=$(_first input_select)
export INPUT_DATETIME_ID=$(_first input_datetime)
export AUTOMATION_ID=$(_first automation)
export TODO_ID=$(_first todo)
export PERSON_ID=$(_first person)
export WEATHER_ID=$(_first weather)
export AREA_ID=$(hactl area list 2>/dev/null | jq -r '.[0].area_id // empty')

echo ""
echo "=== Discovered entities ==="
for v in LIGHT_ID SWITCH_ID CLIMATE_ID COVER_ID FAN_ID MEDIA_PLAYER_ID \
          VACUUM_ID LOCK_ID ALARM_ID SIREN_ID SCENE_ID SCRIPT_ID BUTTON_ID \
          SENSOR_ID BINARY_SENSOR_ID INPUT_BOOLEAN_ID INPUT_TEXT_ID \
          INPUT_NUMBER_ID INPUT_SELECT_ID INPUT_DATETIME_ID AUTOMATION_ID \
          TODO_ID PERSON_ID WEATHER_ID AREA_ID; do
  printf "  %-25s %s\n" "$v" "${!v:-(not found — relevant sections will be skipped)}"
done
```

---

## 0. Prerequisites

```bash
echo "=== 0. Prerequisites ==="

# 0.1 — Sync exposed entities
run "sync" sync
assert_exit 0 "sync"
assert_stderr "sync" "Synced"      # message goes to stdout; check it
[[ -f ~/.config/hactl/exposed-entities.json ]] \
  && _pass "sync: exposed-entities.json exists" \
  || _fail "sync: exposed-entities.json missing"
[[ -f ~/.config/hactl/entity-areas.json ]] \
  && _pass "sync: entity-areas.json exists" \
  || _fail "sync: entity-areas.json missing"

# 0.2 — No-token guard
SAVED_TOKEN="$HASS_TOKEN"
unset HASS_TOKEN
run "no_token_guard" state list
assert_exit 1 "no_token_guard"
assert_stderr "no_token_guard" "HASS_TOKEN is required"
export HASS_TOKEN="$SAVED_TOKEN"
_pass "no_token_guard: token restored"

# 0.3 — Token not in any log output so far
if grep -qr "$HASS_TOKEN" "$HACTL_TEST_LOG" 2>/dev/null; then
  _fail "token_leak_prereq: HASS_TOKEN found in log files"
else
  _pass "token_leak_prereq: HASS_TOKEN not in logs"
fi
```

---

## 1. State — read

```bash
echo "=== 1. State read ==="

# 1.1 — List all exposed entities
run "state_list" state list
assert_exit 0 "state_list"
assert_json "state_list" 'type == "array"'
assert_json "state_list" 'length > 0'

# 1.2 — List by domain
skip_if_unset LIGHT_ID "1.2 list by domain" && {
  run "state_list_domain" state list --domain light
  assert_exit 0 "state_list_domain"
  assert_json "state_list_domain" '[.[].entity_id | startswith("light.")] | all'
}

# 1.3 — List by area
skip_if_unset AREA_ID "1.3 list by area" && {
  run "state_list_area" state list --area "$AREA_ID"
  assert_exit 0 "state_list_area"
  assert_json "state_list_area" 'type == "array"'
}

# 1.4 — Get a single entity (JSON)
skip_if_unset SENSOR_ID "1.4 state get json" && {
  run "state_get" state get "$SENSOR_ID"
  assert_exit 0 "state_get"
  assert_json "state_get" 'has("entity_id") and has("state") and has("attributes") and has("last_changed")'
}

# 1.5 — Get a single entity (plain)
skip_if_unset LIGHT_ID "1.5 state get plain" && {
  run "state_get_plain" state get "$LIGHT_ID" --plain
  assert_exit 0 "state_get_plain"
  assert_nonempty_stdout "state_get_plain"
  # Plain output must contain the entity_id and not be JSON
  grep -q "$LIGHT_ID" "$HACTL_TEST_LOG/state_get_plain.out" \
    && _pass "state_get_plain: entity_id in output" \
    || _fail "state_get_plain: entity_id missing"
  grep -qv '^\[' "$HACTL_TEST_LOG/state_get_plain.out" \
    && _pass "state_get_plain: output is not JSON" \
    || _fail "state_get_plain: output looks like JSON"
}

# 1.6 — jq pipeline
run "state_list_jq" state list
assert_exit 0 "state_list_jq"
jq -r '.[] | .entity_id' "$HACTL_TEST_LOG/state_list_jq.out" > /dev/null \
  && _pass "state_list_jq: jq pipeline works" \
  || _fail "state_list_jq: jq failed"

# 1.7 — Non-existent entity returns correct error
run "state_get_nonexistent" state get "light.this_entity_does_not_exist"
assert_exit 1 "state_get_nonexistent"
assert_stderr "state_get_nonexistent" "entity not found: light.this_entity_does_not_exist"
```

---

## 2. State — set (virtual helpers only)

```bash
echo "=== 2. State set ==="

# 2.1 — input_boolean round-trip
skip_if_unset INPUT_BOOLEAN_ID "2.1 input_boolean" && {
  run "state_set_bool_on" state set "$INPUT_BOOLEAN_ID" on
  assert_exit 0 "state_set_bool_on"
  assert_json "state_set_bool_on" '.state == "on"'

  run "state_set_bool_off" state set "$INPUT_BOOLEAN_ID" off
  assert_exit 0 "state_set_bool_off"
  assert_json "state_set_bool_off" '.state == "off"'
}

# 2.2 — input_text
skip_if_unset INPUT_TEXT_ID "2.2 input_text" && {
  run "state_set_text" state set "$INPUT_TEXT_ID" "hactl test value"
  assert_exit 0 "state_set_text"
  assert_json "state_set_text" '.state == "hactl test value"'
}

# 2.3 — Plain output on set
skip_if_unset INPUT_BOOLEAN_ID "2.3 plain set" && {
  run "state_set_plain" state set "$INPUT_BOOLEAN_ID" on --plain
  assert_exit 0 "state_set_plain"
  grep -q "set to on" "$HACTL_TEST_LOG/state_set_plain.out" \
    && _pass "state_set_plain: 'set to on' in output" \
    || _fail "state_set_plain: expected 'set to on'"
}

# 2.4 — Quiet mode
skip_if_unset INPUT_BOOLEAN_ID "2.4 quiet set" && {
  run "state_set_quiet" state set "$INPUT_BOOLEAN_ID" off --quiet
  assert_exit 0 "state_set_quiet"
  [[ ! -s "$HACTL_TEST_LOG/state_set_quiet.out" ]] \
    && _pass "state_set_quiet: stdout empty" \
    || _fail "state_set_quiet: unexpected stdout"
}

# 2.5 — Blocked on hardware domains
for domain in light switch climate cover fan media_player vacuum lock \
              alarm_control_panel siren button scene script; do
  # Skip if no entity exposed for this domain
  DOMAIN_ENTITY=$(hactl state list --domain "$domain" 2>/dev/null | jq -r '.[0].entity_id // empty')
  [[ -z "$DOMAIN_ENTITY" ]] && continue
  label="state_set_blocked_${domain}"
  run "$label" state set "$DOMAIN_ENTITY" on
  assert_exit 1 "$label"
  assert_stderr "$label" "entities are controlled via services"
done
```

---

## 3. Service calls

```bash
echo "=== 3. Service calls ==="

# 3.1 — light
skip_if_unset LIGHT_ID "3.1 light" && {
  run "svc_light_on"  service call light.turn_on  --entity "$LIGHT_ID"
  assert_exit 0 "svc_light_on"

  run "svc_light_off" service call light.turn_off --entity "$LIGHT_ID"
  assert_exit 0 "svc_light_off"

  run "svc_light_brightness" service call light.turn_on --entity "$LIGHT_ID" --brightness 60
  assert_exit 0 "svc_light_brightness"

  run "svc_light_rgb" service call light.turn_on --entity "$LIGHT_ID" --rgb 255,140,0
  assert_exit 0 "svc_light_rgb"

  run "svc_light_toggle" service call light.toggle --entity "$LIGHT_ID"
  assert_exit 0 "svc_light_toggle"

  # Confirm physical state via state get
  run "svc_light_verify" state get "$LIGHT_ID"
  assert_exit 0 "svc_light_verify"
  assert_json "svc_light_verify" 'has("state")'
}

# 3.2 — switch
skip_if_unset SWITCH_ID "3.2 switch" && {
  run "svc_switch_on"     service call switch.turn_on  --entity "$SWITCH_ID"
  assert_exit 0 "svc_switch_on"
  run "svc_switch_off"    service call switch.turn_off --entity "$SWITCH_ID"
  assert_exit 0 "svc_switch_off"
  run "svc_switch_toggle" service call switch.toggle   --entity "$SWITCH_ID"
  assert_exit 0 "svc_switch_toggle"
}

# 3.3 — climate
skip_if_unset CLIMATE_ID "3.3 climate" && {
  run "svc_climate_temp" service call climate.set_temperature --entity "$CLIMATE_ID" --temperature 21.5
  assert_exit 0 "svc_climate_temp"
  run "svc_climate_mode" service call climate.set_hvac_mode   --entity "$CLIMATE_ID" --hvac-mode heat
  assert_exit 0 "svc_climate_mode"
  run "svc_climate_off"  service call climate.set_hvac_mode   --entity "$CLIMATE_ID" --hvac-mode off
  assert_exit 0 "svc_climate_off"
}

# 3.4 — cover
skip_if_unset COVER_ID "3.4 cover" && {
  run "svc_cover_open"  service call cover.open_cover  --entity "$COVER_ID"
  assert_exit 0 "svc_cover_open"
  run "svc_cover_close" service call cover.close_cover --entity "$COVER_ID"
  assert_exit 0 "svc_cover_close"
  run "svc_cover_pos"   service call cover.set_cover_position --entity "$COVER_ID" --data position=50
  assert_exit 0 "svc_cover_pos"
  run "svc_cover_stop"  service call cover.stop_cover  --entity "$COVER_ID"
  assert_exit 0 "svc_cover_stop"
}

# 3.5 — fan
skip_if_unset FAN_ID "3.5 fan" && {
  run "svc_fan_on"  service call fan.turn_on  --entity "$FAN_ID"
  assert_exit 0 "svc_fan_on"
  run "svc_fan_off" service call fan.turn_off --entity "$FAN_ID"
  assert_exit 0 "svc_fan_off"
}

# 3.6 — media_player
skip_if_unset MEDIA_PLAYER_ID "3.6 media_player" && {
  run "svc_media_pause" service call media_player.media_pause --entity "$MEDIA_PLAYER_ID"
  assert_exit 0 "svc_media_pause"
  run "svc_media_vol"   service call media_player.volume_set  --entity "$MEDIA_PLAYER_ID" --data volume_level=0.3
  assert_exit 0 "svc_media_vol"
}

# 3.7 — vacuum
skip_if_unset VACUUM_ID "3.7 vacuum" && {
  confirm "About to send vacuum.return_to_base to $VACUUM_ID" && {
    run "svc_vacuum_dock" service call vacuum.return_to_base --entity "$VACUUM_ID"
    assert_exit 0 "svc_vacuum_dock"
  }
}

# 3.8 — lock  ⚠ physical device
skip_if_unset LOCK_ID "3.8 lock" && {
  confirm "About to lock/unlock $LOCK_ID — ensure you can re-lock it immediately" && {
    run "svc_lock_lock"   service call lock.lock   --entity "$LOCK_ID"
    assert_exit 0 "svc_lock_lock"
    run "svc_lock_unlock" service call lock.unlock --entity "$LOCK_ID"
    assert_exit 0 "svc_lock_unlock"
    run "svc_lock_relock" service call lock.lock   --entity "$LOCK_ID"
    assert_exit 0 "svc_lock_relock"
  }
}

# 3.9 — alarm_control_panel  ⚠ real alarm
skip_if_unset ALARM_ID "3.9 alarm" && {
  confirm "About to arm/disarm alarm $ALARM_ID — use a test/simulator alarm only" && {
    run "svc_alarm_arm_home" service call alarm_control_panel.alarm_arm_home  --entity "$ALARM_ID"
    assert_exit 0 "svc_alarm_arm_home"
    run "svc_alarm_disarm"   service call alarm_control_panel.alarm_disarm    --entity "$ALARM_ID" --data code=1234
    assert_exit 0 "svc_alarm_disarm"
  }
}

# 3.10 — siren  ⚠ audible
skip_if_unset SIREN_ID "3.10 siren" && {
  confirm "About to activate siren $SIREN_ID" && {
    run "svc_siren_on"  service call siren.turn_on  --entity "$SIREN_ID"
    assert_exit 0 "svc_siren_on"
    run "svc_siren_off" service call siren.turn_off --entity "$SIREN_ID"
    assert_exit 0 "svc_siren_off"
  }
}

# 3.11 — scene
skip_if_unset SCENE_ID "3.11 scene" && {
  run "svc_scene_on" service call scene.turn_on --entity "$SCENE_ID"
  assert_exit 0 "svc_scene_on"
}

# 3.12 — script
skip_if_unset SCRIPT_ID "3.12 script" && {
  run "svc_script_run" service call script.turn_on --entity "$SCRIPT_ID"
  assert_exit 0 "svc_script_run"
}

# 3.13 — button
skip_if_unset BUTTON_ID "3.13 button" && {
  confirm "About to press $BUTTON_ID — verify it triggers a safe action" && {
    run "svc_button_press" service call button.press --entity "$BUTTON_ID"
    assert_exit 0 "svc_button_press"
  }
}

# 3.14 — input_number
skip_if_unset INPUT_NUMBER_ID "3.14 input_number" && {
  run "svc_input_num" service call input_number.set_value --entity "$INPUT_NUMBER_ID" --data value=50
  assert_exit 0 "svc_input_num"
  run "svc_input_num_verify" state get "$INPUT_NUMBER_ID"
  assert_exit 0 "svc_input_num_verify"
  assert_json "svc_input_num_verify" '.state == "50" or (.state | tonumber) == 50'
}

# 3.15 — input_select
skip_if_unset INPUT_SELECT_ID "3.15 input_select" && {
  OPTION=$(hactl state get "$INPUT_SELECT_ID" 2>/dev/null | jq -r '.attributes.options[0] // empty')
  [[ -n "$OPTION" ]] && {
    run "svc_input_select" service call input_select.select_option \
      --entity "$INPUT_SELECT_ID" --data "option=$OPTION"
    assert_exit 0 "svc_input_select"
  }
}

# 3.16 — input_datetime
skip_if_unset INPUT_DATETIME_ID "3.16 input_datetime" && {
  run "svc_input_dt" service call input_datetime.set_datetime \
    --entity "$INPUT_DATETIME_ID" --data time=07:30:00
  assert_exit 0 "svc_input_dt"
}

# 3.17 — Domain mismatch guard
skip_if_unset LIGHT_ID "3.17 domain mismatch" && {
  run "svc_domain_mismatch" service call switch.turn_on --entity "$LIGHT_ID"
  assert_exit 1 "svc_domain_mismatch"
  assert_stderr "svc_domain_mismatch" "domain mismatch"
}

# 3.18 — Missing --entity hint
run "svc_missing_entity" service call light.turn_on
assert_exit 1 "svc_missing_entity"
assert_stderr "svc_missing_entity" "entity"

# 3.19 — homeassistant cross-domain (exempt from domain-mismatch check)
skip_if_unset LIGHT_ID "3.19 homeassistant cross-domain" && {
  run "svc_ha_turn_off" service call homeassistant.turn_off --entity "$LIGHT_ID"
  assert_exit 0 "svc_ha_turn_off"
  run "svc_ha_turn_on"  service call homeassistant.turn_on  --entity "$LIGHT_ID"
  assert_exit 0 "svc_ha_turn_on"
}
```

---

## 4. Restricted services

```bash
echo "=== 4. Restricted services ==="

# 4.1 — homeassistant.restart blocked in exposed mode
run "restart_blocked" service call homeassistant.restart
assert_exit 1 "restart_blocked"
assert_stderr "restart_blocked" "not permitted in exposed mode"

# 4.2 — homeassistant.check_config allowed in all mode
# NOTE: requires filter.mode: all in config — skip in auto mode.
confirm "Set filter.mode: all in config, then test homeassistant.check_config?" && {
  run "check_config_all_mode" service call homeassistant.check_config
  assert_exit 0 "check_config_all_mode"
  echo ">>> Restore filter.mode: exposed in config before continuing <<<"
}
```

---

## 5. Automations

```bash
echo "=== 5. Automations ==="

# 5.1 — List
run "automation_list" automation list
assert_exit 0 "automation_list"
assert_json "automation_list" 'type == "array"'

run "automation_list_plain" automation list --plain
assert_exit 0 "automation_list_plain"

# 5.2 — Trigger
skip_if_unset AUTOMATION_ID "5.2 automation trigger" && {
  run "automation_trigger" automation trigger "$AUTOMATION_ID"
  assert_exit 0 "automation_trigger"

  # Without prefix
  BARE_ID="${AUTOMATION_ID#automation.}"
  run "automation_trigger_bare" automation trigger "$BARE_ID"
  assert_exit 0 "automation_trigger_bare"
}

# 5.3 — Disable and re-enable
skip_if_unset AUTOMATION_ID "5.3 automation enable/disable" && {
  run "automation_disable" automation disable "$AUTOMATION_ID"
  assert_exit 0 "automation_disable"

  run "automation_list_after_disable" automation list
  assert_exit 0 "automation_list_after_disable"
  assert_json "automation_list_after_disable" \
    "[.[] | select(.entity_id == \"$AUTOMATION_ID\") | .state == \"off\"] | any"

  run "automation_enable" automation enable "$AUTOMATION_ID"
  assert_exit 0 "automation_enable"

  run "automation_list_after_enable" automation list
  assert_json "automation_list_after_enable" \
    "[.[] | select(.entity_id == \"$AUTOMATION_ID\") | .state == \"on\"] | any"
}
```

---

## 6. Todo lists

```bash
echo "=== 6. Todo lists ==="

# 6.1 — List all todo lists
run "todo_list_all" todo list
assert_exit 0 "todo_list_all"
assert_json "todo_list_all" 'type == "array"'

# 6.2 — List a specific list
skip_if_unset TODO_ID "6.2 todo list specific" && {
  run "todo_list_specific" todo list "$TODO_ID"
  assert_exit 0 "todo_list_specific"
  assert_json "todo_list_specific" 'type == "array"'

  run "todo_list_plain" todo list "$TODO_ID" --plain
  assert_exit 0 "todo_list_plain"

  # Without prefix
  BARE_TODO="${TODO_ID#todo.}"
  run "todo_list_bare" todo list "$BARE_TODO"
  assert_exit 0 "todo_list_bare"
}

# 6.3 — Add, complete, remove round-trip
skip_if_unset TODO_ID "6.3 todo add/done/remove" && {
  TEST_ITEM="hactl-test-item-$(date +%s)"

  run "todo_add" todo add "$TODO_ID" "$TEST_ITEM"
  assert_exit 0 "todo_add"

  run "todo_list_after_add" todo list "$TODO_ID"
  assert_exit 0 "todo_list_after_add"
  assert_json "todo_list_after_add" \
    "[.[] | select(.summary == \"$TEST_ITEM\") | .status == \"needs_action\"] | any"

  run "todo_done" todo done "$TODO_ID" "$TEST_ITEM"
  assert_exit 0 "todo_done"

  run "todo_list_after_done" todo list "$TODO_ID"
  assert_json "todo_list_after_done" \
    "[.[] | select(.summary == \"$TEST_ITEM\") | .status == \"completed\"] | any"

  run "todo_remove" todo remove "$TODO_ID" "$TEST_ITEM"
  assert_exit 0 "todo_remove"

  run "todo_list_after_remove" todo list "$TODO_ID"
  assert_json "todo_list_after_remove" \
    "[.[] | select(.summary == \"$TEST_ITEM\")] | length == 0"
}

# 6.4 — Quiet and plain output
skip_if_unset TODO_ID "6.4 todo quiet/plain" && {
  QUIET_ITEM="hactl-quiet-$(date +%s)"

  run "todo_add_quiet" todo add "$TODO_ID" "$QUIET_ITEM" --quiet
  assert_exit 0 "todo_add_quiet"
  [[ ! -s "$HACTL_TEST_LOG/todo_add_quiet.out" ]] \
    && _pass "todo_add_quiet: stdout empty" \
    || _fail "todo_add_quiet: unexpected stdout"

  run "todo_add_plain" todo add "$TODO_ID" "plain-test-$(date +%s)" --plain
  assert_exit 0 "todo_add_plain"
  grep -q "added" "$HACTL_TEST_LOG/todo_add_plain.out" \
    && _pass "todo_add_plain: 'added' in output" \
    || _fail "todo_add_plain: expected 'added' in output"

  # Clean up quiet item
  run "todo_cleanup" todo remove "$TODO_ID" "$QUIET_ITEM" --quiet
}
```

---

## 7. History

```bash
echo "=== 7. History ==="

# 7.1 — JSON output
skip_if_unset SENSOR_ID "7.1 history json" && {
  run "history_json" history "$SENSOR_ID" --last 1h
  assert_exit 0 "history_json"
  assert_json "history_json" 'type == "array"'
}

# 7.2 — Plain output
skip_if_unset LIGHT_ID "7.2 history plain" && {
  run "history_plain" history "$LIGHT_ID" --last 1h --plain
  assert_exit 0 "history_plain"
  assert_nonempty_stdout "history_plain"
}

# 7.3 — Invalid duration format
skip_if_unset SENSOR_ID "7.3 history invalid duration" && {
  run "history_bad_duration" history "$SENSOR_ID" --last 2days
  assert_exit 1 "history_bad_duration"
  assert_stderr "history_bad_duration" "invalid"
}

# 7.4 — Non-existent entity
run "history_nonexistent" history "light.this_entity_does_not_exist" --last 1h
assert_exit 1 "history_nonexistent"
assert_stderr "history_nonexistent" "entity not found"
```

---

## 8. Summary

```bash
echo "=== 8. Summary ==="

# 8.1 — Full JSON summary
run "summary_json" summary
assert_exit 0 "summary_json"
assert_json "summary_json" 'has("generated_at") and has("domains")'
assert_json "summary_json" '.domains | type == "array"'

# 8.2 — Plain summary
run "summary_plain" summary --plain
assert_exit 0 "summary_plain"
assert_nonempty_stdout "summary_plain"
# Plain output must not be JSON
grep -qE '^\[|^\{' "$HACTL_TEST_LOG/summary_plain.out" \
  && _fail "summary_plain: output looks like JSON" \
  || _pass "summary_plain: not JSON"

# 8.3 — Area-filtered summary
skip_if_unset AREA_ID "8.3 summary area filter" && {
  run "summary_area" summary --area "$AREA_ID"
  assert_exit 0 "summary_area"
  assert_json "summary_area" 'has("domains")'
}

# 8.4 — Quiet mode
run "summary_quiet" summary --quiet
assert_exit 0 "summary_quiet"
[[ ! -s "$HACTL_TEST_LOG/summary_quiet.out" ]] \
  && _pass "summary_quiet: stdout empty" \
  || _fail "summary_quiet: unexpected stdout"
```

---

## 9. Areas

```bash
echo "=== 9. Areas ==="

# 9.1 — JSON listing
run "area_list" area list
assert_exit 0 "area_list"
assert_json "area_list" 'type == "array"'
assert_json "area_list" '[.[]] | map(has("area_id") and has("name")) | all'

# 9.2 — Plain listing
run "area_list_plain" area list --plain
assert_exit 0 "area_list_plain"
assert_nonempty_stdout "area_list_plain"
grep -q "id=" "$HACTL_TEST_LOG/area_list_plain.out" \
  && _pass "area_list_plain: (id=...) format present" \
  || _fail "area_list_plain: (id=...) format missing"
```

---

## 10. Persons

```bash
echo "=== 10. Persons ==="

# 10.1 — JSON listing
run "person_list" person list
assert_exit 0 "person_list"
assert_json "person_list" 'type == "array"'
skip_if_unset PERSON_ID "10.1 person fields" && {
  assert_json "person_list" '[.[]] | map(has("entity_id") and has("state")) | all'
}

# 10.2 — Plain listing
run "person_list_plain" person list --plain
assert_exit 0 "person_list_plain"
skip_if_unset PERSON_ID "10.2 person plain" && {
  assert_nonempty_stdout "person_list_plain"
  # Each line should be "Name: state"
  grep -qE '.+: (home|not_home|.+)' "$HACTL_TEST_LOG/person_list_plain.out" \
    && _pass "person_list_plain: 'Name: state' format" \
    || _fail "person_list_plain: unexpected format"
}
```

---

## 11. Weather

```bash
echo "=== 11. Weather ==="

# 11.1 — Auto-select first weather entity
run "weather_auto" weather
assert_exit 0 "weather_auto"
assert_json "weather_auto" 'has("entity_id") and has("condition")'

# 11.2 — Explicit entity
skip_if_unset WEATHER_ID "11.2 weather explicit" && {
  run "weather_explicit" weather "$WEATHER_ID"
  assert_exit 0 "weather_explicit"
  assert_json "weather_explicit" '.entity_id == "'"$WEATHER_ID"'"'

  # Without prefix
  BARE_WEATHER="${WEATHER_ID#weather.}"
  run "weather_bare" weather "$BARE_WEATHER"
  assert_exit 0 "weather_bare"
}

# 11.3 — Plain output
run "weather_plain" weather --plain
assert_exit 0 "weather_plain"
assert_nonempty_stdout "weather_plain"
grep -qE '^[a-z]' "$HACTL_TEST_LOG/weather_plain.out" \
  && _pass "weather_plain: starts with condition word" \
  || _fail "weather_plain: unexpected format"
```

---

## 12. Events stream

> **Manual only.** The events stream is interactive and cannot be asserted
> automatically. Run the commands below in a separate terminal, trigger a state
> change (e.g. toggle a light), verify events appear, then Ctrl-C.

```bash
echo "=== 12. Events (manual verification required) ==="
echo "[MANUAL] hactl events watch"
echo "[MANUAL] hactl events watch --type state_changed"
echo "[MANUAL] hactl events watch --domain light"
echo "[MANUAL] hactl events watch --type state_changed --plain"
echo "Run each in a separate terminal. Toggle a device to trigger events."
echo "Verify compact 'entity: old -> new' format appears with --plain."
_pass "events: marked as manual — no automated assertion"
```

---

## 13. Output format consistency

```bash
echo "=== 13. Output format consistency ==="

# JSON must be valid for state list
jq . "$HACTL_TEST_LOG/state_list.out" > /dev/null 2>&1 \
  && _pass "format: state_list is valid JSON" \
  || _fail "format: state_list is not valid JSON"

# --plain must not contain JSON brackets
grep -qE '^\[|^\{' "$HACTL_TEST_LOG/summary_plain.out" \
  && _fail "format: summary_plain contains JSON" \
  || _pass "format: summary_plain is not JSON"

# --quiet must produce no stdout
[[ ! -s "$HACTL_TEST_LOG/summary_quiet.out" ]] \
  && _pass "format: summary_quiet stdout empty" \
  || _fail "format: summary_quiet stdout non-empty"

# Non-zero exit on error
[[ "$(cat "$HACTL_TEST_LOG/state_get_nonexistent.exit")" != "0" ]] \
  && _pass "format: error produces non-zero exit" \
  || _fail "format: error produced exit 0"

# Stderr is clean on success
[[ ! -s "$HACTL_TEST_LOG/state_list.err" ]] \
  && _pass "format: state_list stderr empty on success" \
  || _fail "format: state_list wrote to stderr on success"
```

---

## 14. Entity filter

```bash
echo "=== 14. Entity filter ==="

# 14.1 — Hidden entity: same error as non-existent
# Find a real entity that is NOT in the exposed cache.
HIDDEN_ENTITY=$(comm -13 \
  <(jq -r '.[]' ~/.config/hactl/exposed-entities.json | sort) \
  <(hactl state list --domain sensor 2>/dev/null | \
    jq -r '.[].entity_id' | sort) \
  | head -1)

if [[ -n "$HIDDEN_ENTITY" ]]; then
  run "state_get_hidden" state get "$HIDDEN_ENTITY"
  assert_exit 1 "state_get_hidden"
  assert_stderr "state_get_hidden" "entity not found"

  # Error message must be identical format to non-existent entity
  HIDDEN_MSG=$(cat "$HACTL_TEST_LOG/state_get_hidden.err")
  NONEXIST_MSG=$(cat "$HACTL_TEST_LOG/state_get_nonexistent.err")
  HIDDEN_PATTERN=$(echo "$HIDDEN_MSG"   | sed 's/: .*/: /')
  NONEXIST_PATTERN=$(echo "$NONEXIST_MSG" | sed 's/: .*/: /')
  [[ "$HIDDEN_PATTERN" == "$NONEXIST_PATTERN" ]] \
    && _pass "filter: hidden and non-existent produce identical error format" \
    || _fail "filter: error messages differ — hidden='$HIDDEN_MSG' nonexist='$NONEXIST_MSG'"
else
  echo "[SKIP] 14.1 — no unexposed sensor entity available on this setup"
fi

# 14.2 — All mode: hidden entity accessible
# (requires manual config change — gate behind confirm)
confirm "Switch config to filter.mode: all and test access to hidden entity $HIDDEN_ENTITY?" && {
  if [[ -n "$HIDDEN_ENTITY" ]]; then
    run "state_get_hidden_all_mode" state get "$HIDDEN_ENTITY"
    assert_exit 0 "state_get_hidden_all_mode"
    assert_json "state_get_hidden_all_mode" 'has("state")'
    echo ">>> Restore filter.mode: exposed before continuing <<<"
  else
    echo "[SKIP] No hidden entity available"
  fi
}
```

---

## 15. Sync

```bash
echo "=== 15. Sync ==="

# 15.1 — Re-sync and verify cache files are updated
run "sync_rerun" sync
assert_exit 0 "sync_rerun"

CACHE_MTIME=$(stat -f "%m" ~/.config/hactl/exposed-entities.json 2>/dev/null \
  || stat -c "%Y" ~/.config/hactl/exposed-entities.json 2>/dev/null)
NOW=$(date +%s)
AGE=$(( NOW - CACHE_MTIME ))
[[ "$AGE" -lt 30 ]] \
  && _pass "sync_rerun: exposed-entities.json updated within last 30s" \
  || _fail "sync_rerun: exposed-entities.json not recently updated (age=${AGE}s)"

# 15.2 — Entity count matches state list
CACHE_COUNT=$(jq 'length' ~/.config/hactl/exposed-entities.json)
LIST_COUNT=$(jq 'length' "$HACTL_TEST_LOG/state_list.out")
[[ "$CACHE_COUNT" -eq "$LIST_COUNT" ]] \
  && _pass "sync_rerun: cache count ($CACHE_COUNT) matches state list ($LIST_COUNT)" \
  || _fail "sync_rerun: cache count ($CACHE_COUNT) ≠ state list ($LIST_COUNT)"
```

---

## 16. Admin commands

> Requires `filter.mode: all` in `~/.config/hactl/config.yaml`.
> Use a non-critical entity (e.g. a sensor or helper).

```bash
echo "=== 16. Admin commands ==="

# 16.1 — Guard: all three commands blocked in exposed mode
run "expose_guard"   expose   "${SENSOR_ID:-sensor.test}"
assert_exit 1 "expose_guard"
assert_stderr "expose_guard" "filter.mode: all"

run "unexpose_guard" unexpose "${SENSOR_ID:-sensor.test}"
assert_exit 1 "unexpose_guard"
assert_stderr "unexpose_guard" "filter.mode: all"

run "rename_guard"   rename   "${SENSOR_ID:-sensor.test}" "Test"
assert_exit 1 "rename_guard"
assert_stderr "rename_guard" "filter.mode: all"

# 16.2–16.10 — Require filter.mode: all
confirm "Switch config to filter.mode: all to test expose/unexpose/rename?" && {
  skip_if_unset SENSOR_ID "16.3–16.9" || {
    echo "[SKIP] 16.3–16.9: SENSOR_ID not set"
  }
  skip_if_unset SENSOR_ID "16.3–16.9" && {
    ORIG_NAME=$(hactl state get "$SENSOR_ID" 2>/dev/null | jq -r '.attributes.friendly_name // empty')

    # 16.3 — unexpose
    run "admin_unexpose" unexpose "$SENSOR_ID"
    assert_exit 0 "admin_unexpose"
    run "admin_sync_after_unexpose" sync
    assert_exit 0 "admin_sync_after_unexpose"
    run "admin_get_after_unexpose" state get "$SENSOR_ID"
    assert_exit 1 "admin_get_after_unexpose"
    assert_stderr "admin_get_after_unexpose" "entity not found"

    # 16.4 — re-expose
    run "admin_expose" expose "$SENSOR_ID"
    assert_exit 0 "admin_expose"
    run "admin_sync_after_expose" sync
    assert_exit 0 "admin_sync_after_expose"
    run "admin_get_after_expose" state get "$SENSOR_ID"
    assert_exit 0 "admin_get_after_expose"
    assert_json "admin_get_after_expose" 'has("state")'

    # 16.5 — rename
    run "admin_rename" rename "$SENSOR_ID" "hactl Test Sensor"
    assert_exit 0 "admin_rename"
    run "admin_sync_after_rename" sync
    assert_exit 0 "admin_sync_after_rename"
    run "admin_get_after_rename" state get "$SENSOR_ID"
    assert_exit 0 "admin_get_after_rename"
    assert_json "admin_get_after_rename" '.attributes.friendly_name == "hactl Test Sensor"'

    # 16.6 — entity ID unchanged after rename
    assert_json "admin_get_after_rename" ".entity_id == \"$SENSOR_ID\""

    # 16.7 — restore original name
    [[ -n "$ORIG_NAME" ]] && {
      run "admin_rename_restore" rename "$SENSOR_ID" "$ORIG_NAME"
      assert_exit 0 "admin_rename_restore"
      run "admin_sync_restore" sync
      assert_exit 0 "admin_sync_restore"
    }

    # 16.8 — quiet output
    run "admin_expose_quiet" expose "$SENSOR_ID" --quiet
    assert_exit 0 "admin_expose_quiet"
    [[ ! -s "$HACTL_TEST_LOG/admin_expose_quiet.out" ]] \
      && _pass "admin_expose_quiet: stdout empty" \
      || _fail "admin_expose_quiet: unexpected stdout"

    # 16.9 — invalid entity
    run "admin_expose_invalid" expose "light.entity_that_does_not_exist_xyz"
    assert_exit 1 "admin_expose_invalid"
  }

  echo ">>> Restore filter.mode: exposed in config and run sync before continuing <<<"
  confirm "Confirm you have restored filter.mode: exposed and run hactl sync"
}
```

---

## 17. Edge cases

```bash
echo "=== 17. Edge cases ==="

# --data without =
skip_if_unset LIGHT_ID "17.1 --data no equals" && {
  run "edge_data_no_equals" service call light.turn_on --entity "$LIGHT_ID" --data brightness
  assert_exit 1 "edge_data_no_equals"
  assert_stderr "edge_data_no_equals" "key=value"
}

# --area with no matching entities
run "edge_area_nonexistent" state list --area "this_area_does_not_exist_xyz"
assert_exit 0 "edge_area_nonexistent"
assert_json "edge_area_nonexistent" '. == []'

# todo list with no todo entities (relies on section 6 having run)
[[ -z "$TODO_ID" ]] && {
  run "edge_todo_none" todo list
  assert_exit 0 "edge_todo_none"
  grep -q "no todo lists found" "$HACTL_TEST_LOG/edge_todo_none.out" \
    && _pass "edge_todo_none: 'no todo lists found'" \
    || _fail "edge_todo_none: expected 'no todo lists found'"
}

# automation trigger with non-existent ID
run "edge_automation_nonexistent" automation trigger "nonexistent_automation_xyz"
assert_exit 1 "edge_automation_nonexistent"
assert_stderr "edge_automation_nonexistent" "entity not found"

# expose in exposed mode (already covered in 16.1 — verify label)
assert_exit 1 "expose_guard"
assert_stderr "expose_guard" "filter.mode: all"
```

---

## 18. Security audit

Validates captured logs for security-relevant properties.
Run this block last, after all other sections have completed.

```bash
echo "=== 18. Security audit ==="

# 18.1 — Token must not appear in any log file
if grep -rl "$HASS_TOKEN" "$HACTL_TEST_LOG" 2>/dev/null | grep -qv '^$'; then
  _fail "sec: HASS_TOKEN found in log files — review before sharing"
  grep -rl "$HASS_TOKEN" "$HACTL_TEST_LOG"
else
  _pass "sec: HASS_TOKEN not in any log file"
fi

# 18.2 — Observed entity IDs must all be in the exposed cache
EXPOSED=$(jq -r '.[]' ~/.config/hactl/exposed-entities.json 2>/dev/null | sort)
OBSERVED=$(jq -r '.[].entity_id' "$HACTL_TEST_LOG/state_list.out" 2>/dev/null | sort)
EXTRA=$(comm -13 <(echo "$EXPOSED") <(echo "$OBSERVED"))
if [[ -n "$EXTRA" ]]; then
  _fail "sec: entities in state list NOT in exposed cache (filter bypass):"
  echo "$EXTRA"
else
  _pass "sec: all observed entities are in the exposed cache"
fi

# 18.3 — Hidden and non-existent entities must produce identical error format
if [[ -f "$HACTL_TEST_LOG/state_get_hidden.err" ]]; then
  HIDDEN_FMT=$(sed 's/: [^ ]*/: <id>/' "$HACTL_TEST_LOG/state_get_hidden.err")
  NONEXIST_FMT=$(sed 's/: [^ ]*/: <id>/' "$HACTL_TEST_LOG/state_get_nonexistent.err")
  [[ "$HIDDEN_FMT" == "$NONEXIST_FMT" ]] \
    && _pass "sec: hidden and non-existent produce identical error format" \
    || _fail "sec: error messages differ — probing vector possible"
else
  echo "[SKIP] sec 18.3: state_get_hidden not run (no unexposed entity found)"
fi

# 18.4 — Admin commands blocked in exposed mode
for label in expose_guard unexpose_guard rename_guard; do
  if [[ -f "$HACTL_TEST_LOG/${label}.exit" ]]; then
    assert_exit 1 "$label"
    assert_stderr "$label" "filter.mode: all"
  else
    _fail "sec: $label was never run"
  fi
done

# 18.5 — Restricted service blocked in exposed mode
if [[ -f "$HACTL_TEST_LOG/restart_blocked.exit" ]]; then
  assert_exit 1 "restart_blocked"
  assert_stderr "restart_blocked" "not permitted in exposed mode"
else
  _fail "sec: restart_blocked was never run"
fi

# 18.6 — state set blocked for all hardware domains
for domain in light switch climate cover fan media_player vacuum lock \
              alarm_control_panel siren button scene script; do
  label="state_set_blocked_${domain}"
  [[ ! -f "$HACTL_TEST_LOG/${label}.exit" ]] && continue  # no entity, was skipped
  assert_exit 1 "$label"
  assert_stderr "$label" "entities are controlled via services"
done

# 18.7 — filter.mode must be restored to exposed
CURRENT_MODE=$(grep -E '^\s*mode:' ~/.config/hactl/config.yaml 2>/dev/null | awk '{print $2}')
[[ "$CURRENT_MODE" == "exposed" || -z "$CURRENT_MODE" ]] \
  && _pass "sec: filter.mode is exposed (or defaulting to exposed)" \
  || _fail "sec: filter.mode=$CURRENT_MODE — restore to exposed immediately"
```

---

## Summary

Run this block last to get a final pass/fail count for the session.

```bash
echo ""
echo "========================================"
echo "  hactl test session summary"
echo "  Log: $HACTL_TEST_LOG"
echo "========================================"
PASS=$(grep -c '^\[PASS\]' "$HACTL_TEST_LOG/failures.log" 2>/dev/null || true)
FAIL=$(grep -c '^\[FAIL\]' "$HACTL_TEST_LOG/failures.log" 2>/dev/null || true)
# Count PASS from session output (failures.log only captures failures)
PASS=$(grep -rh '^\[PASS\]' "$HACTL_TEST_LOG/" 2>/dev/null | wc -l | tr -d ' ')
FAIL=$(grep -c '.' "$HACTL_TEST_LOG/failures.log" 2>/dev/null || echo 0)
echo "  PASS: $PASS"
echo "  FAIL: $FAIL"
if [[ "$FAIL" -gt 0 ]]; then
  echo ""
  echo "Failures:"
  cat "$HACTL_TEST_LOG/failures.log"
fi
echo "========================================"
[[ "$FAIL" -eq 0 ]] && echo "All checks passed." || echo "Review failures above."
```
