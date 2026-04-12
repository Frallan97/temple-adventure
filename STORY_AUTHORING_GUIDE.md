# Story Authoring Guide

This guide covers everything you need to create stories for the Temple Adventure engine. Stories are authored as JSON files using the **StorySpec** format, which gets expanded into a fully-wired game world.

## Quick Start

A minimal story needs: rooms, items, at least one `win_condition` puzzle, and a start room.

```json
{
  "title": "My Story",
  "slug": "my-story",
  "description": "A short description.",
  "author": "Your Name",
  "start_room": "start",
  "rooms": { ... },
  "items": { ... },
  "puzzles": [ ... ],
  "npcs": { ... }
}
```

Post to the API: `POST /api/v1/stories/from-spec` with the JSON body.

---

## StorySpec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `title` | string | Yes | Story title (must be unique) |
| `slug` | string | Yes | URL-friendly identifier |
| `description` | string | No | Story blurb |
| `author` | string | No | Author name (defaults to "Anonymous") |
| `start_room` | string | Yes | Room ID where the player starts |
| `rooms` | object | Yes | Map of room ID to RoomSpec |
| `items` | object | Yes | Map of item ID to ItemSpec |
| `puzzles` | array | Yes | List of PuzzleSpecs (at least one `win_condition` required) |
| `npcs` | object | No | Map of NPC ID to NpcSpec |

---

## Rooms

```json
"tavern": {
  "name": "The Dusty Tavern",
  "description": "A dimly lit tavern with creaking floorboards.",
  "connections": {
    "north": "alley",
    "east": "cellar"
  },
  "items": ["mug", "notice_board"],
  "description_after_puzzle": "The tavern feels different now. The cellar door stands open."
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Display name shown to the player |
| `description` | string | Yes | Room description when the player looks around |
| `connections` | object | Yes | Map of direction to room ID |
| `items` | array | Yes | List of item IDs present in the room |
| `description_after_puzzle` | string | No | Replaces description when a puzzle in this room is solved |

**Directions**: `north`, `south`, `east`, `west`, `up`, `down`. Reverse connections are created automatically (e.g., if room A connects north to room B, room B gets a south connection to A).

---

## Items

```json
"old_key": {
  "name": "rusty key",
  "description": "An old iron key, coated in rust but still solid.",
  "aliases": ["key", "iron key"],
  "portable": true,
  "examine_text": "Despite the rust, the key looks like it could still work."
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Display name |
| `description` | string | Yes | Shown when examining the item |
| `aliases` | array | No | Alternative names the player can use |
| `portable` | bool | Yes | Can the player pick it up? |
| `examine_text` | string | No | Custom response when examining (adds an "examine" interaction) |

**Portable items** can be taken with `take <item>` and dropped with `drop <item>`.
**Non-portable items** stay in the room but can be interacted with via custom puzzle verbs (e.g., `use door`, `turn dial`).

---

## Puzzles

Puzzles use a template system. Set the `type` field and fill in the fields relevant to that type. Every story needs at least one `win_condition` puzzle.

### Common Fields (all puzzle types)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique puzzle identifier |
| `type` | string | Yes | One of the 8 types below |
| `name` | string | Yes | Puzzle name |
| `description` | string | Yes | Puzzle description |
| `room` | string | Yes | Room where the puzzle takes place |
| `completion_text` | string | No | Text shown when the puzzle is solved |

---

### `key_lock`

Use a key item on a locked object to unlock a new direction.

```json
{
  "id": "cellar_lock",
  "type": "key_lock",
  "room": "stairs",
  "name": "Unlock the Cellar",
  "description": "The cellar door is locked.",
  "key_item": "old_key",
  "lock_target": "cellar_door",
  "unlock_direction": "down",
  "unlock_room": "cellar",
  "completion_text": "The door swings open.",
  "lock_fail_text": "The door is locked. You need a key."
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `key_item` | Yes | Item ID used as the key (consumed on use) |
| `lock_target` | Yes | Item ID of the locked object |
| `unlock_direction` | Yes | Direction that gets unlocked |
| `unlock_room` | Yes | Room that becomes accessible |
| `lock_verb` | No | Verb to use (default: `"use"`) |
| `lock_fail_text` | No | Message when trying without the key |

**What happens**: The connection in `unlock_direction` is removed from the room initially (locked). When the player uses the key item on the lock target, the key is consumed and the connection is restored.

---

### `examine_learn`

Examine one item to learn a clue, then use that knowledge on another item.

```json
{
  "id": "panel_puzzle",
  "type": "examine_learn",
  "room": "control_room",
  "name": "Crack the Code",
  "description": "Figure out the panel sequence.",
  "source_item": "diagram",
  "source_learn_text": "The diagram shows: press red, then blue.",
  "target_item": "control_panel",
  "target_verb": "activate",
  "target_success_text": "The panel hums to life!",
  "target_fail_text": "You don't know the sequence yet."
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `source_item` | Yes | Item to examine for the clue |
| `source_learn_text` | Yes | Response when examining the source |
| `target_item` | Yes | Item to interact with using the knowledge |
| `target_verb` | No | Verb for the target interaction (default: `"use"`) |
| `target_success_text` | No | Response on success |
| `target_fail_text` | No | Response when trying without examining first |
| `unlock_direction` | No | Optionally unlock a direction on solve |
| `unlock_room` | No | Room to unlock |

---

### `fetch_quest`

Bring an item to a specific room and use it there.

```json
{
  "id": "deliver_gem",
  "type": "fetch_quest",
  "room": "altar_room",
  "name": "Place the Gem",
  "description": "The altar has an empty socket.",
  "fetch_item": "red_gem",
  "fetch_room": "altar_room",
  "fetch_verb": "place",
  "fetch_success_text": "The gem clicks into the socket. A passage opens!",
  "fetch_consume_item": true,
  "unlock_direction": "north",
  "unlock_room": "hidden_chamber"
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `fetch_item` | Yes | Item the player must bring |
| `fetch_room` | No | Destination room (defaults to `room`) |
| `fetch_verb` | No | Verb to use (default: `"use"`) |
| `fetch_success_text` | No | Response on success |
| `fetch_consume_item` | No | Remove the item after use? (default: false) |
| `unlock_direction` | No | Optionally unlock a direction |
| `unlock_room` | No | Room to unlock |

---

### `timed_challenge`

A puzzle with a turn limit. The player activates a trigger and must complete an objective before time runs out.

```json
{
  "id": "flooding",
  "type": "timed_challenge",
  "room": "engine_room",
  "name": "Stop the Flood",
  "description": "The room is flooding!",
  "trigger_item": "valve",
  "trigger_verb": "turn",
  "trigger_text": "Water starts pouring in! You have 5 turns!",
  "turn_limit": 5,
  "failure_text": "The water sweeps you back to the entrance.",
  "failure_effects": [
    {"type": "move_player", "room": "entrance"}
  ],
  "fetch_item": "wrench",
  "fetch_verb": "use",
  "fetch_success_text": "You tighten the pipe. The flooding stops!"
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `trigger_item` | Yes | Item that starts the timer |
| `trigger_verb` | No | Verb to start (default: `"turn"`) |
| `trigger_text` | No | Response when activating |
| `turn_limit` | Yes | Number of turns before failure |
| `failure_text` | No | Text shown on timeout |
| `failure_effects` | No | Effects applied on failure (see below) |
| `fetch_item` | No | Optional fetch sub-puzzle |
| `fetch_room` | No | Where to use the fetch item |
| `fetch_verb` | No | Verb for fetch (default: `"use"`) |
| `fetch_success_text` | No | Response when fetch succeeds |

**Failure effects** are simplified:
- `{"type": "move_player", "room": "entrance"}` — teleport the player
- `{"type": "lock_connection", "room": "engine_room", "direction": "north"}` — block a path

---

### `win_condition`

Taking or using an item ends the game. **Every story must have at least one.**

```json
{
  "id": "win_treasure",
  "type": "win_condition",
  "room": "vault",
  "name": "Claim the Treasure",
  "win_item": "golden_idol",
  "win_verb": "take",
  "win_text": "You grasp the golden idol. Victory!"
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `win_item` | Yes | Item that triggers the win |
| `win_verb` | No | Verb to use (default: `"take"`) |
| `win_text` | No | Victory message |

---

### `combination_lock`

Interact with an item multiple times in sequence to solve it.

```json
{
  "id": "safe_combo",
  "type": "combination_lock",
  "room": "office",
  "name": "Crack the Safe",
  "description": "A combination safe.",
  "combination_target": "safe_dial",
  "combination_verb": "turn",
  "combination_steps": 3,
  "combination_texts": [
    "Click! First tumbler falls into place.",
    "Click! Second tumbler aligns.",
    "The safe door swings open!"
  ],
  "unlock_direction": "east",
  "unlock_room": "vault"
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `combination_target` | Yes | Item to interact with |
| `combination_steps` | Yes | Number of interactions needed |
| `combination_verb` | No | Verb to use (default: `"turn"`) |
| `combination_texts` | No | Response for each step (array length must match steps) |
| `unlock_direction` | No | Optionally unlock a direction |
| `unlock_room` | No | Room to unlock |

---

### `item_combine`

Combine two inventory items into a new item.

```json
{
  "id": "craft_torch",
  "type": "item_combine",
  "room": "workshop",
  "name": "Make a Torch",
  "description": "Combine cloth and a stick.",
  "combine_item_a": "cloth",
  "combine_item_b": "stick",
  "combine_result": "torch",
  "combine_verb": "combine",
  "combine_consume_a": true,
  "combine_consume_b": true,
  "combine_text": "You wrap the cloth around the stick. A makeshift torch!",
  "combine_fail_text": "You need both cloth and a stick."
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `combine_item_a` | Yes | First item (the one the player interacts with) |
| `combine_item_b` | Yes | Second item (must be in inventory) |
| `combine_result` | Yes | Resulting item (must be defined, portable, and not in any room) |
| `combine_verb` | No | Verb to use (default: `"use"`) |
| `combine_consume_a` | No | Remove item A? (default: false) |
| `combine_consume_b` | No | Remove item B? (default: false) |
| `combine_text` | No | Success message |
| `combine_fail_text` | No | Message when missing an ingredient |

---

### `counter_puzzle`

Use multiple items to accumulate a count toward a target.

```json
{
  "id": "ritual",
  "type": "counter_puzzle",
  "room": "altar",
  "name": "Complete the Ritual",
  "description": "Place all three crystals on the altar.",
  "counter_items": ["fire_crystal", "water_crystal", "earth_crystal"],
  "counter_verb": "place",
  "counter_target": 3,
  "counter_item_texts": {
    "fire_crystal": "Flames dance across the altar.",
    "water_crystal": "Water ripples across the surface.",
    "earth_crystal": "The ground trembles."
  },
  "counter_consume_items": true,
  "unlock_direction": "north",
  "unlock_room": "sanctum"
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `counter_items` | Yes | List of item IDs that count toward the goal |
| `counter_target` | Yes | How many items needed (must be <= item count) |
| `counter_verb` | No | Verb to use (default: `"use"`) |
| `counter_default_text` | No | Default response per item (default: `"Done."`) |
| `counter_item_texts` | No | Per-item response text (map of item ID to string) |
| `counter_consume_items` | No | Remove items after use? (default: false) |
| `unlock_direction` | No | Optionally unlock a direction |
| `unlock_room` | No | Room to unlock |

All counter items must be portable.

---

## NPCs

NPCs have two dialogue modes. Use **simple mode** for background characters and **dialogue tree mode** for important characters.

### Simple Mode (Greeting + Topics)

```json
"shopkeeper": {
  "name": "Shopkeeper",
  "description": "A cheerful woman behind the counter.",
  "aliases": ["woman", "shop owner"],
  "room": "shop",
  "greeting": "Welcome to my shop! How can I help?",
  "topics": {
    "prices": "Everything is reasonably priced!",
    "sword": "The sword? That's 50 gold."
  }
}
```

- `talk shopkeeper` returns the greeting
- `ask shopkeeper about prices` returns the topic response
- Players must guess topic names (less discoverable)

### Dialogue Tree Mode

For branching conversations with player choices. When `dialogue` is present, it overrides `greeting` and `topics`.

```json
"barkeep": {
  "name": "Barkeep",
  "description": "A burly man with kind eyes.",
  "aliases": ["bartender"],
  "room": "tavern",
  "dialogue": [
    {
      "node_id": "greeting",
      "text": "Welcome, stranger! What can I do for you?",
      "choices": [
        {"text": "I saw the notice about a lost heirloom", "next_node": "heirloom"},
        {"text": "What's the story with the cellar?", "next_node": "cellar"},
        {"text": "Just looking around", "next_node": "looking"}
      ]
    },
    {
      "node_id": "heirloom",
      "text": "My grandmother's locket went missing weeks ago.",
      "choices": [
        {"text": "How do I get into the cellar?", "next_node": "cellar_key"},
        {"text": "I'll find it for you", "next_node": "__exit__"}
      ]
    },
    {
      "node_id": "cellar_key",
      "text": "I dropped the key in the back alley."
    },
    {
      "node_id": "cellar",
      "text": "It's been locked up for months. Strange noises lately."
    },
    {
      "node_id": "looking",
      "text": "Help yourself to the ale. Check the notice board for work."
    }
  ]
}
```

### Dialogue Node Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `node_id` | string | Yes | Unique ID within this NPC |
| `text` | string | Yes | What the NPC says |
| `topic` | string | No | If set, reachable via `ask <npc> about <topic>` |
| `choices` | array | No | Player choices (if absent, conversation ends) |

### Dialogue Choice Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `text` | string | Yes | What the player says (shown as numbered option) |
| `next_node` | string | Yes | Node ID to go to, or `"__exit__"` to end |
| `need_item` | string | No | Only show this choice if player has the item |
| `set_var` | string | No | Set a variable on choose (format: `"key=value"`) |
| `give_item` | string | No | Add an item to player inventory on choose |

### How Dialogue Works

1. **`talk <npc>`** starts at the first node with no `topic` (the greeting). If already in a conversation, resumes at the current node.
2. **`ask <npc> about <topic>`** jumps to the node with that `topic` value.
3. The NPC's text is shown, followed by numbered choices.
4. **`say <N>`** or just typing `<N>` selects a choice.
5. Choices with `need_item` are hidden if the player doesn't have the item.
6. `next_node: "__exit__"` or an empty string ends the conversation.
7. A node with no `choices` ends the conversation automatically.

### Dialogue State

The engine tracks:
- `dlg.<npcID>.node` — which node the player is currently at (cleared on exit)
- `dlg.<npcID>.visited.<nodeID>` — set to true when a node is shown

You can use these in conditions elsewhere. For example, to check if the player has talked to an NPC:
```json
{"type": "var_equals", "key": "dlg.barkeep.visited.cellar_key", "value": true}
```

---

## NPC Fields Reference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Display name |
| `description` | string | Yes | Shown when player examines the NPC |
| `aliases` | array | No | Alternative names |
| `room` | string | Yes | Starting room (must exist) |
| `greeting` | string | No | Simple mode: response to `talk` |
| `topics` | object | No | Simple mode: map of topic to response |
| `dialogue` | array | No | Tree mode: list of DialogueNodeSpecs (overrides greeting/topics) |

---

## Conditions and Effects

These are the building blocks used internally by puzzle templates and available in raw dialogue/NPC definitions.

### Conditions

All conditions support `"negate": true` to invert the result.

| Type | Key | Value | Description |
|------|-----|-------|-------------|
| `has_item` | item ID | — | Player has item in inventory |
| `var_equals` | variable name | expected value | Variable equals value (bool/int/string) |
| `var_gte` | variable name | number | Integer variable >= value |
| `var_lte` | variable name | number | Integer variable <= value |
| `in_room` | room ID | — | Player is in this room |
| `item_in_room` | item ID | — | Item is in the current room |
| `puzzle_complete` | puzzle ID | — | Puzzle has been solved |
| `npc_in_room` | NPC ID | room ID (optional) | NPC is in room (defaults to player's room) |

### Effects

| Type | Key | Value | Description |
|------|-----|-------|-------------|
| `set_var` | variable name | new value | Set a variable (bool/int/string) |
| `increment_var` | variable name | amount | Add to an integer variable |
| `add_item` | item ID | — | Give item to player |
| `remove_item` | item ID | — | Remove item from player inventory |
| `add_room_item` | item ID | room ID | Place item in a room |
| `remove_room_item` | item ID | room ID (optional) | Remove item from room |
| `move_player` | — | room ID | Teleport player to room |
| `move_npc` | NPC ID | room ID | Move NPC to room |
| `unlock_connection` | `room.direction` | target room | Open a locked path |
| `lock_connection` | `room.direction` | — | Block a path |
| `set_status` | — | status string | Set game status (`"completed"` = win) |

---

## Player Commands

| Command | Aliases | Description |
|---------|---------|-------------|
| `look` | `l`, `examine`, `inspect` | Describe room or examine item/NPC |
| `move <dir>` | `go`, `walk`, `n/s/e/w`, `north/south/east/west`, `up/down` | Move in a direction |
| `take <item>` | `get`, `grab` | Pick up an item |
| `drop <item>` | — | Drop an item from inventory |
| `use <item>` | — | Use an item (triggers interactions) |
| `talk <npc>` | `speak`, `chat` | Start/resume conversation |
| `ask <npc> about <topic>` | — | Ask about a specific topic |
| `say <N>` | `respond`, `choose`, or bare number | Select a dialogue choice |
| `inventory` | `i` | List inventory |
| `hint` | `h`, `clue` | Get a context-sensitive hint |
| `help` | `?` | Show command list |

Any unrecognized verb is tried as a custom item interaction (e.g., `push button`, `open chest`, `pull lever`).

---

## Validation Rules

The engine validates stories before accepting them.

### Structure Validation (before expansion)
- `title` and `slug` are required
- At least one room exists
- `start_room` exists in rooms
- All item references in rooms point to existing items
- All room connections point to existing rooms
- Puzzle IDs are unique
- Type-specific required fields are present
- NPC rooms exist
- Dialogue `next_node` references point to existing `node_id`s within the same NPC

### Deep Validation (after expansion)
- All variables used in conditions are set by some effect
- All items referenced in effects exist
- All `unlock_connection` targets reference valid rooms
- All puzzles have at least one step
- At least one effect sets status to `"completed"` (win condition exists)

---

## Design Tips

1. **Gate access with locked doors**, not hidden items. Use `key_lock` to separate areas so the player has a clear progression.

2. **Use dialogue trees** for important NPCs. Simple `greeting`/`topics` mode is fine for flavor NPCs, but players can't discover topics without hints.

3. **Every win_condition item should be behind at least one puzzle**. If the player can walk straight to the win item and pick it up, there's no challenge.

4. **Keep rooms small and focused**. 3-5 rooms is plenty for a short story. Each room should have a purpose.

5. **Use examine_text** on items to drop hints. Players who examine things should be rewarded with useful information.

6. **Test your story by playing through it**. Verify: Can the player reach every room? Can they get every required item? Is the win condition achievable?

7. **Use `description_after_puzzle`** to show the player that something changed. When a puzzle is solved, the room should feel different.

8. **NPC dialogue should guide the player**. Use dialogue choices to naturally hint at what to do next ("Where's the key?" leads to "Check the alley").

---

## Complete Example

See `stories/salon.json` for a 2-room story with dialogue trees and a branching NPC conversation that leads to the win condition.

See `stories/tavern_test.json` for a 4-room story with a key_lock puzzle, three dialogue-tree NPCs, and a multi-step adventure flow.
