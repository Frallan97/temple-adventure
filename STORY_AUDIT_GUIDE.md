# Story Audit Guide

Use this guide to audit story JSON files for the Temple Adventure engine. Work through each section in order. A story must pass all checks before it can be deployed.

---

## How to Run the Story Compiler

The `storyc` CLI tool validates a story JSON through a 4-stage pipeline **without needing a running server**. This is the recommended first step before any manual audit.

```bash
cd backend
go run ./cmd/storyc/ ../stories/my_story.json
```

You can compile multiple stories at once:

```bash
go run ./cmd/storyc/ ../stories/*.json
```

### What `storyc` checks

| Stage | What it does | Catches |
|-------|-------------|---------|
| 1. Structural validation | References, required fields, puzzle types | Missing items, bad room refs, invalid puzzle types |
| 2. Expansion | Converts spec → full world definition | Broken puzzle templates, missing targets |
| 3. Deep validation | Variable flow, effect refs, win condition | Undefined variables, missing win condition |
| 4. Gameplay analysis | Graph-based reachability and logic checks | Unreachable rooms, self-referential locks, key behind its own door, circular puzzle deps, dead ends, orphan dialogue nodes |

### Reading the output

- **✓** — Check passed
- **✗** — Error (must fix before deploying)
- **⚠** — Warning (should fix, but not blocking)
- **PASS** / **FAIL** — Overall result. Exit code is 0 on pass, 1 on fail.

Example output for a broken story:

```
=== Compiling: broken_story.json ===
  Story: Broken (broken)

  Stage 1: Structural validation
    ✓ structural validation passed

  Stage 2: Expansion
    ✓ expanded (2 rooms, 3 items, 0 NPCs, 1 puzzles)

  Stage 3: Deep validation
    ✓ deep validation passed

  Stage 4: Gameplay analysis
    ✗ puzzle "lock1": key_lock unlock_room "room1" is the same as the puzzle room — creates an infinite loop
    ✗ room "island" is not reachable from start_room "room1"
    ⚠ room "island" has no connections — potential dead end

  FAIL — 2 error(s), 1 warning(s)
```

**Fix all errors reported by `storyc` before proceeding to the manual audit below.** The compiler catches §1 (Structural Integrity), §2 (Reachability), and §3 (Puzzle Logic) automatically. The remaining sections (Dialogue Quality, Narrative Coherence, Player Experience, Edge Cases) require human review.

### Alternative: API validation

If you have a running server, you can also validate by posting the story JSON:

```bash
curl -s -X POST http://localhost:8080/api/v1/stories/from-spec \
  -H 'Content-Type: application/json' \
  -d @story.json | jq .
```

- **201 Created**: Story is structurally valid and was saved.
- **422 Unprocessable Entity**: Validation failed. The response contains `validation_errors` — fix every listed error.
- **400 Bad Request**: Malformed JSON.

Note: The API runs stages 1-3 but does **not** run the stage 4 gameplay analysis. Use `storyc` for the full check.

---

## Audit Checklist

### 1. Structural Integrity

These are the same checks the engine runs automatically. Verify them manually if you don't have a running server.

- [ ] `title` is present and non-empty
- [ ] `slug` is present, lowercase, uses only `a-z`, `0-9`, and `-`
- [ ] `start_room` exists in the `rooms` map
- [ ] Every room has a `name` and `description`
- [ ] Every room connection points to an existing room ID
- [ ] Every item ID referenced in a room's `items` array exists in the `items` map
- [ ] Every item has a `name`, `description`, and `portable` field
- [ ] At least one puzzle exists with `"type": "win_condition"`
- [ ] Every puzzle has a unique `id`
- [ ] Every puzzle `type` is one of: `key_lock`, `examine_learn`, `fetch_quest`, `timed_challenge`, `win_condition`, `combination_lock`, `item_combine`, `counter_puzzle`
- [ ] Every puzzle `room` exists in the `rooms` map
- [ ] Every NPC has a `name` and a `room` that exists
- [ ] Every dialogue `next_node` references an existing `node_id` within the same NPC (or is `"__exit__"` or `""`)
- [ ] Every dialogue node has a `node_id`

#### Puzzle-Specific Field Checks

**key_lock**: `key_item`, `lock_target`, `unlock_direction`, `unlock_room` all present and reference existing items/rooms.

**examine_learn**: `source_item`, `target_item` exist. If `unlock_room` set, it exists.

**fetch_quest**: `fetch_item` exists. If `fetch_room` set, it exists.

**timed_challenge**: `trigger_item` exists. `turn_limit` > 0. `failure_effects` types are `"move_player"` or `"lock_connection"` only.

**win_condition**: `win_item` exists.

**combination_lock**: `combination_target` exists. `combination_steps` > 0. If `combination_texts` provided, length matches `combination_steps`.

**item_combine**: `combine_item_a`, `combine_item_b`, `combine_result` all exist. Result item is portable. Result item is NOT placed in any room.

**counter_puzzle**: `counter_items` is non-empty. All items exist and are portable. `counter_target` > 0 and <= number of counter items.

---

### 2. Reachability

The player must be able to reach every room, item, and NPC through normal gameplay.

- [ ] **Every room is reachable** from `start_room` by following connections (accounting for locked doors that can be unlocked)
- [ ] **Every key item is reachable** before the lock it opens. Example: if a key is in room B and the lock is in room C, the player must be able to reach room B without going through room C first.
- [ ] **The win item is reachable** after solving all prerequisite puzzles
- [ ] **No dead ends**: every room the player can enter has a way back out (or the game ends there)
- [ ] **Locked directions are correct**: a `key_lock` puzzle removes the connection from the room initially — verify the `unlock_direction` and `unlock_room` make sense geographically
- [ ] **Items needed for puzzles are in accessible rooms**: trace the path for every puzzle and confirm the player can collect all required items

#### How to Trace Reachability

1. Start at `start_room`. List all rooms reachable via connections.
2. For each `key_lock` puzzle, check: can the player reach the `key_item` without the locked door? If yes, the locked door is solvable.
3. Repeat until all unlockable doors are accounted for.
4. Every room in the map should be reachable through this process.

---

### 3. Puzzle Logic

- [ ] **No skip paths**: the player cannot reach the win condition without solving the required puzzles. Common mistake: the win item is in a room that's directly accessible without unlocking anything.
- [ ] **key_lock: unlock_room is a different room than where the door is**. A `key_lock` that unlocks a direction pointing back to the same room creates an infinite loop. The `unlock_room` should be a room on the OTHER side of the door.
- [ ] **examine_learn: source and target make narrative sense**. Examining the source should logically give knowledge needed for the target.
- [ ] **timed_challenge: turn_limit is fair**. Count the minimum number of commands needed to solve the challenge. The turn limit should be larger than this minimum (by at least 2-3 turns of margin).
- [ ] **item_combine: result item is not already in a room**. The result is generated by combining — it should not exist anywhere on the map.
- [ ] **counter_puzzle: all counter items are reachable and portable**.
- [ ] **Puzzle dependencies are solvable in order**. If puzzle B requires solving puzzle A first, verify A is solvable without B.

---

### 4. Dialogue Quality

- [ ] **Every important NPC uses dialogue trees** (the `dialogue` array), not simple `greeting`/`topics`. Players cannot discover topics without hints, making simple mode frustrating for key NPCs.
- [ ] **The greeting node makes sense as a first impression**. It should establish who the NPC is and hint at what they can help with.
- [ ] **Every dialogue path leads somewhere or exits cleanly**. No node should leave the player confused about what to do. Terminal nodes (no choices) should feel like a natural end.
- [ ] **`__exit__` nodes have meaningful context**. The player should feel the conversation concluded, not that it broke.
- [ ] **Conditional choices (`need_item`) don't create unwinnable states**. If a choice requires an item, the player must be able to get that item and return to the NPC. The NPC should still have at least one visible choice even without the item.
- [ ] **No orphan nodes**: every node with a `node_id` should be reachable from the greeting or from a `topic` that the player can trigger via `ask about`.
- [ ] **Dialogue guides the player**. Key NPCs should hint at: where to go, what to look for, or what to do next. The player should not have to guess.
- [ ] **NPC descriptions match their personality**. The `description` (shown on examine) and dialogue text should feel consistent.

---

### 5. Narrative Coherence

- [ ] **Room descriptions match their contents**. If a room description mentions a "heavy oak door", there should be a door item. If it mentions a chest, there should be a chest item.
- [ ] **Item descriptions make sense in context**. A "rusty key" in a fancy salon is suspicious unless explained. Items should feel like they belong.
- [ ] **Puzzle flow tells a story**. The sequence of puzzles should feel like a narrative progression, not arbitrary obstacles. Example: learn about the cellar from barkeep → find key in alley → unlock cellar → find treasure.
- [ ] **NPC knowledge is consistent**. If an NPC mentions something, it should be true in the game world. If the barkeep says "the key is in the alley", the key should be in the alley.
- [ ] **Win text provides closure**. The `win_text` should wrap up the story, not just say "you win".
- [ ] **Examine text rewards curiosity**. Items with `examine_text` should reveal useful or interesting information, not just repeat the description.
- [ ] **Room descriptions don't mention locked areas as accessible**. If a direction is locked, the room description shouldn't say "a path leads north" — it should mention the obstacle.

---

### 6. Player Experience

- [ ] **The start room gives the player direction**. There should be something to examine, an NPC to talk to, or an obvious first action. Don't drop the player in an empty room.
- [ ] **There are no unwinnable states**. At every point in the game, it should be possible to reach the win condition. Watch for: consumable items used on the wrong thing, one-way doors that lock the player out of required items.
- [ ] **The critical path is clear enough**. A player who explores and talks to NPCs should be able to figure out what to do without external help.
- [ ] **The `hint` system works** (if rooms have hints). Hints should be useful at every stage, not just at the start.
- [ ] **The story is completable in a reasonable number of turns**. For a short story (2-4 rooms), 15-25 turns is typical. For longer stories, up to 50.
- [ ] **Custom verbs are hinted at**. If a puzzle requires `open chest` or `turn dial`, an NPC or item description should mention that verb. Players will try `use` by default.

---

### 7. Edge Cases

- [ ] **Trying puzzle items before finding the key/clue gives a helpful fail message**. The `lock_fail_text`, `target_fail_text`, or `combine_fail_text` should tell the player what's missing.
- [ ] **Taking items out of order doesn't break anything**. Can the player take the win item's prerequisites in any order?
- [ ] **Talking to NPCs multiple times works**. Dialogue trees should handle re-entry (the engine resumes at the last node). Terminal nodes should make sense on repeat visits.
- [ ] **Items have sensible aliases**. Players will try common names. A "rusty key" should have aliases like `["key", "iron key"]`. A "mug of ale" should have `["mug", "ale", "drink"]`.
- [ ] **NPC aliases cover common references**. A "Barkeep" should have aliases like `["bartender", "innkeeper"]`.
- [ ] **Non-portable items can't be taken**. Verify that items meant to stay in rooms (doors, furniture, mechanisms) have `"portable": false`.

---

## Audit Report Template

After auditing, produce a report in this format:

```
# Story Audit: [Story Title]

## Summary
- Rooms: N
- Items: N
- NPCs: N
- Puzzles: N
- Estimated turns to complete: N

## Critical Issues (must fix)
- [Description of issue and how to fix it]

## Warnings (should fix)
- [Description of concern]

## Suggestions (nice to have)
- [Description of improvement]

## Playthrough Notes
[Walk through the intended solution path step by step, noting any friction points]

## Verdict: PASS / FAIL
```

---

## Common Bugs to Watch For

| Bug | How to Detect | Example |
|-----|--------------|---------|
| Self-referential lock | `key_lock` where `unlock_room` = `room` | Cellar door unlocks "down" to the same room |
| Win item unguarded | Win item in a directly accessible room with no puzzle gate | Heirloom sitting in the open with no locked door |
| Orphan room | Room exists but no connection leads to it | Secret room with no entrance |
| Missing reverse connection | Room A connects to B, but B can't reach A | One-way trip with no way back |
| Unreachable key | Key is behind the door it unlocks | Key in the cellar, cellar door locked |
| Dead NPC dialogue | NPC has `topics` but no way for the player to discover topic names | Barkeep knows about "heirloom" but nothing hints at that word |
| Consumable item used wrong | Player uses a one-time item on the wrong target | Key consumed on wrong door, now stuck |
| Empty room | Room with no items, NPCs, or purpose | Hallway with nothing to do |
| Missing fail text | Player tries locked door and gets no feedback | `use door` gives no response |
| Puzzle ordering impossible | Puzzle B requires puzzle A's reward, but A requires B's | Circular dependency |
