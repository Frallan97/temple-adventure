package storygen

import (
	"fmt"
	"sort"
	"strings"

	"temple-adventure/engine"
)

// ValidationResult holds errors (must fix) and warnings (should fix).
type ValidationResult struct {
	Errors   []string
	Warnings []string
}

func (vr *ValidationResult) addError(msg string) {
	vr.Errors = append(vr.Errors, msg)
}

func (vr *ValidationResult) addWarning(msg string) {
	vr.Warnings = append(vr.Warnings, msg)
}

func (vr *ValidationResult) Ok() bool {
	return len(vr.Errors) == 0
}

// ValidateSpec checks a StorySpec for structural errors before expansion.
func ValidateSpec(spec *StorySpec) []string {
	var errs []string

	if spec.Title == "" {
		errs = append(errs, "title is required")
	}
	if spec.Slug == "" {
		errs = append(errs, "slug is required")
	}
	if len(spec.Rooms) == 0 {
		errs = append(errs, "at least one room is required")
	}
	if spec.StartRoom == "" {
		errs = append(errs, "start_room is required")
	} else if _, ok := spec.Rooms[spec.StartRoom]; !ok {
		errs = append(errs, fmt.Sprintf("start_room %q not found in rooms", spec.StartRoom))
	}

	// Check item references in rooms
	for roomID, room := range spec.Rooms {
		for _, itemID := range room.Items {
			if _, ok := spec.Items[itemID]; !ok {
				errs = append(errs, fmt.Sprintf("room %q references unknown item %q", roomID, itemID))
			}
		}
		for dir, targetID := range room.Connections {
			if _, ok := spec.Rooms[targetID]; !ok {
				errs = append(errs, fmt.Sprintf("room %q connection %q references unknown room %q", roomID, dir, targetID))
			}
		}
	}

	// Check puzzles
	winCount := 0
	seenEndingIDs := map[string]string{} // endingID → puzzleID
	validTypes := map[string]bool{
		"key_lock": true, "examine_learn": true, "fetch_quest": true,
		"timed_challenge": true, "win_condition": true,
		"combination_lock": true, "item_combine": true, "counter_puzzle": true,
	}
	seenIDs := map[string]bool{}

	for _, ps := range spec.Puzzles {
		if ps.ID == "" {
			errs = append(errs, "puzzle has empty id")
			continue
		}
		if seenIDs[ps.ID] {
			errs = append(errs, fmt.Sprintf("duplicate puzzle id %q", ps.ID))
		}
		seenIDs[ps.ID] = true

		if !validTypes[ps.Type] {
			errs = append(errs, fmt.Sprintf("puzzle %q has invalid type %q", ps.ID, ps.Type))
			continue
		}

		if ps.Room != "" {
			if _, ok := spec.Rooms[ps.Room]; !ok {
				errs = append(errs, fmt.Sprintf("puzzle %q references unknown room %q", ps.ID, ps.Room))
			}
		}

		// Type-specific checks
		switch ps.Type {
		case "key_lock":
			errs = append(errs, checkItemRef(spec, ps.ID, "key_item", ps.KeyItem)...)
			errs = append(errs, checkItemRef(spec, ps.ID, "lock_target", ps.LockTarget)...)
			errs = append(errs, checkRoomRef(spec, ps.ID, "unlock_room", ps.UnlockRoom)...)
			if ps.UnlockDirection == "" {
				errs = append(errs, fmt.Sprintf("puzzle %q: unlock_direction is required", ps.ID))
			}
		case "examine_learn":
			errs = append(errs, checkItemRef(spec, ps.ID, "source_item", ps.SourceItem)...)
			errs = append(errs, checkItemRef(spec, ps.ID, "target_item", ps.TargetItem)...)
			if ps.UnlockRoom != "" {
				errs = append(errs, checkRoomRef(spec, ps.ID, "unlock_room", ps.UnlockRoom)...)
			}
		case "fetch_quest":
			errs = append(errs, checkItemRef(spec, ps.ID, "fetch_item", ps.FetchItem)...)
			if ps.FetchRoom != "" {
				errs = append(errs, checkRoomRef(spec, ps.ID, "fetch_room", ps.FetchRoom)...)
			}
			if ps.UnlockRoom != "" {
				errs = append(errs, checkRoomRef(spec, ps.ID, "unlock_room", ps.UnlockRoom)...)
			}
		case "timed_challenge":
			errs = append(errs, checkItemRef(spec, ps.ID, "trigger_item", ps.TriggerItem)...)
			if ps.TurnLimit <= 0 {
				errs = append(errs, fmt.Sprintf("puzzle %q: turn_limit must be > 0", ps.ID))
			}
			if ps.FetchItem != "" {
				errs = append(errs, checkItemRef(spec, ps.ID, "fetch_item", ps.FetchItem)...)
			}
			if ps.UnlockRoom != "" {
				errs = append(errs, checkRoomRef(spec, ps.ID, "unlock_room", ps.UnlockRoom)...)
			}
			for _, fe := range ps.FailureEffects {
				if fe.Type != "move_player" && fe.Type != "lock_connection" {
					errs = append(errs, fmt.Sprintf("puzzle %q: invalid failure effect type %q", ps.ID, fe.Type))
				}
			}
		case "win_condition":
			errs = append(errs, checkItemRef(spec, ps.ID, "win_item", ps.WinItem)...)
			winCount++

			// Endings validation
			if len(ps.Endings) > 0 && ps.EndingID != "" {
				errs = append(errs, fmt.Sprintf("puzzle %q: cannot use both 'endings' array and 'ending_id' — they are mutually exclusive", ps.ID))
			}
			if len(ps.Endings) > 0 && ps.WinText != "" {
				errs = append(errs, fmt.Sprintf("puzzle %q: 'win_text' is ignored when 'endings' array is present — remove win_text or use it as the fallback ending text", ps.ID))
			}
			hasFallback := false
			for _, ending := range ps.Endings {
				if ending.ID == "" {
					errs = append(errs, fmt.Sprintf("puzzle %q: ending is missing 'id'", ps.ID))
				} else if prev, dup := seenEndingIDs[ending.ID]; dup {
					errs = append(errs, fmt.Sprintf("puzzle %q: ending id %q already used in puzzle %q", ps.ID, ending.ID, prev))
				} else {
					seenEndingIDs[ending.ID] = ps.ID
				}
				if ending.Text == "" {
					errs = append(errs, fmt.Sprintf("puzzle %q: ending %q is missing 'text'", ps.ID, ending.ID))
				}
				if len(ending.Conditions) == 0 {
					hasFallback = true
				}
			}
			if len(ps.Endings) > 0 && !hasFallback {
				errs = append(errs, fmt.Sprintf("puzzle %q: endings array has no fallback ending (an ending with no conditions) — player may see no ending text", ps.ID))
			}
			// Track EndingID from per-puzzle field
			if ps.EndingID != "" {
				if prev, dup := seenEndingIDs[ps.EndingID]; dup {
					errs = append(errs, fmt.Sprintf("puzzle %q: ending_id %q already used in puzzle %q", ps.ID, ps.EndingID, prev))
				} else {
					seenEndingIDs[ps.EndingID] = ps.ID
				}
			}
		case "combination_lock":
			errs = append(errs, checkItemRef(spec, ps.ID, "combination_target", ps.CombinationTarget)...)
			if ps.CombinationSteps <= 0 {
				errs = append(errs, fmt.Sprintf("puzzle %q: combination_steps must be > 0", ps.ID))
			}
			if len(ps.CombinationTexts) > 0 && len(ps.CombinationTexts) != ps.CombinationSteps {
				errs = append(errs, fmt.Sprintf("puzzle %q: combination_texts length (%d) must match combination_steps (%d)", ps.ID, len(ps.CombinationTexts), ps.CombinationSteps))
			}
			if ps.UnlockRoom != "" {
				errs = append(errs, checkRoomRef(spec, ps.ID, "unlock_room", ps.UnlockRoom)...)
			}
		case "item_combine":
			errs = append(errs, checkItemRef(spec, ps.ID, "combine_item_a", ps.CombineItemA)...)
			errs = append(errs, checkItemRef(spec, ps.ID, "combine_item_b", ps.CombineItemB)...)
			errs = append(errs, checkItemRef(spec, ps.ID, "combine_result", ps.CombineResult)...)
			if ps.CombineResult != "" {
				if item, ok := spec.Items[ps.CombineResult]; ok && !item.Portable {
					errs = append(errs, fmt.Sprintf("puzzle %q: combine_result %q should be portable", ps.ID, ps.CombineResult))
				}
				for roomID, room := range spec.Rooms {
					for _, itemID := range room.Items {
						if itemID == ps.CombineResult {
							errs = append(errs, fmt.Sprintf("puzzle %q: combine_result %q should not be placed in room %q", ps.ID, ps.CombineResult, roomID))
						}
					}
				}
			}
		case "counter_puzzle":
			if len(ps.CounterItems) == 0 {
				errs = append(errs, fmt.Sprintf("puzzle %q: counter_items is required", ps.ID))
			}
			for _, itemID := range ps.CounterItems {
				errs = append(errs, checkItemRef(spec, ps.ID, "counter_items", itemID)...)
				if item, ok := spec.Items[itemID]; ok && !item.Portable {
					errs = append(errs, fmt.Sprintf("puzzle %q: counter item %q must be portable", ps.ID, itemID))
				}
			}
			if ps.CounterTarget <= 0 {
				errs = append(errs, fmt.Sprintf("puzzle %q: counter_target must be > 0", ps.ID))
			}
			if ps.CounterTarget > len(ps.CounterItems) {
				errs = append(errs, fmt.Sprintf("puzzle %q: counter_target (%d) exceeds counter_items count (%d)", ps.ID, ps.CounterTarget, len(ps.CounterItems)))
			}
			if ps.UnlockRoom != "" {
				errs = append(errs, checkRoomRef(spec, ps.ID, "unlock_room", ps.UnlockRoom)...)
			}
		}
	}

	if winCount == 0 {
		errs = append(errs, "at least one win_condition puzzle is required")
	}

	// Check NPCs
	for npcID, npc := range spec.Npcs {
		if npc.Name == "" {
			errs = append(errs, fmt.Sprintf("npc %q: name is required", npcID))
		}
		if npc.Room == "" {
			errs = append(errs, fmt.Sprintf("npc %q: room is required", npcID))
		} else if _, ok := spec.Rooms[npc.Room]; !ok {
			errs = append(errs, fmt.Sprintf("npc %q: references unknown room %q", npcID, npc.Room))
		}

		// Validate dialogue tree node references
		if len(npc.Dialogue) > 0 {
			nodeIDs := make(map[string]bool)
			for _, dn := range npc.Dialogue {
				if dn.NodeID == "" {
					errs = append(errs, fmt.Sprintf("npc %q: dialogue node missing node_id", npcID))
				}
				nodeIDs[dn.NodeID] = true
			}
			for _, dn := range npc.Dialogue {
				for _, cs := range dn.Choices {
					if cs.NextNode != "" && cs.NextNode != "__exit__" {
						if !nodeIDs[cs.NextNode] {
							errs = append(errs, fmt.Sprintf("npc %q: dialogue choice references unknown node %q", npcID, cs.NextNode))
						}
					}
				}
			}
		}
	}

	return errs
}

// ValidateGameplay runs graph-based gameplay checks on a StorySpec.
// These detect reachability issues, puzzle logic bugs, and dialogue problems.
func ValidateGameplay(spec *StorySpec) *ValidationResult {
	vr := &ValidationResult{}

	// 1. Self-referential key_lock detection
	for _, ps := range spec.Puzzles {
		if ps.Type == "key_lock" && ps.UnlockRoom == ps.Room {
			vr.addError(fmt.Sprintf("puzzle %q: key_lock unlock_room %q is the same as the puzzle room — creates an infinite loop", ps.ID, ps.Room))
		}
	}

	// 2. Room reachability from start_room (accounting for key_lock unlockable doors)
	checkRoomReachability(spec, vr)

	// 3. Key reachable before its lock
	checkKeyBeforeLock(spec, vr)

	// 4. Win item not behind any gate (skip path detection)
	checkWinItemGated(spec, vr)

	// 5. Circular puzzle dependencies
	checkCircularDeps(spec, vr)

	// 6. Dead-end rooms (no exit, warning)
	checkDeadEnds(spec, vr)

	// 7. Missing reverse connections (warning)
	checkMissingReverseConnections(spec, vr)

	// 8. Orphan dialogue nodes (warning)
	checkOrphanDialogueNodes(spec, vr)

	// 9. NPC using simple topics with no discoverability (warning)
	checkSimpleTopicNpcs(spec, vr)

	// 10. Ending-specific checks
	checkEndings(spec, vr)

	return vr
}

// reachableRooms returns the set of rooms reachable from start via connections,
// treating rooms behind key_lock puzzles as locked unless their key is reachable.
// It iteratively unlocks doors as keys become reachable.
func reachableRooms(spec *StorySpec) map[string]bool {
	// Build the set of locked connections: room+direction → puzzle
	type lockedDoor struct {
		puzzleID  string
		keyItem   string
		fromRoom  string
		direction string
		toRoom    string
	}

	var locks []lockedDoor
	for _, ps := range spec.Puzzles {
		if ps.Type == "key_lock" && ps.Room != "" && ps.UnlockRoom != "" {
			locks = append(locks, lockedDoor{
				puzzleID:  ps.ID,
				keyItem:   ps.KeyItem,
				fromRoom:  ps.Room,
				direction: ps.UnlockDirection,
				toRoom:    ps.UnlockRoom,
			})
		}
	}

	// key_lock puzzles define a locked connection from fromRoom→toRoom.
	// The spec may or may not include this connection explicitly.
	// We treat it as a connection that exists but is locked.
	lockedConns := make(map[string]map[string]string) // roomID → direction → targetRoom (locked)
	for _, l := range locks {
		if lockedConns[l.fromRoom] == nil {
			lockedConns[l.fromRoom] = make(map[string]string)
		}
		lockedConns[l.fromRoom][l.direction] = l.toRoom
	}

	// Find which room each item is in
	itemRoom := make(map[string]string)
	for roomID, room := range spec.Rooms {
		for _, itemID := range room.Items {
			itemRoom[itemID] = roomID
		}
	}

	// Build full connection map: spec connections + locked connections (all potential paths)
	// We'll track which locked connections have been unlocked
	unlocked := make(map[string]map[string]bool) // roomID → direction → unlocked?

	// Iterative BFS: reach rooms, unlock doors when key is reachable, repeat
	reached := make(map[string]bool)
	changed := true
	for changed {
		changed = false

		// BFS from start_room using spec connections + unlocked locked connections
		queue := []string{spec.StartRoom}
		visited := make(map[string]bool)
		visited[spec.StartRoom] = true

		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]

			room, ok := spec.Rooms[cur]
			if !ok {
				continue
			}
			// Follow spec connections (skip directions that are locked and not yet unlocked)
			for dir, target := range room.Connections {
				if _, isLocked := lockedConns[cur][dir]; isLocked && (unlocked[cur] == nil || !unlocked[cur][dir]) {
					continue // this direction is locked
				}
				if !visited[target] {
					visited[target] = true
					queue = append(queue, target)
				}
			}
			// Follow unlocked key_lock connections (these may not be in spec connections)
			if unlocked[cur] != nil {
				for dir, isUnlocked := range unlocked[cur] {
					if isUnlocked {
						target := lockedConns[cur][dir]
						if target != "" && !visited[target] {
							visited[target] = true
							queue = append(queue, target)
						}
					}
				}
			}
		}

		// Check if we can unlock any new doors
		for _, l := range locks {
			if unlocked[l.fromRoom] != nil && unlocked[l.fromRoom][l.direction] {
				continue // already unlocked
			}
			keyRoom, keyPlaced := itemRoom[l.keyItem]
			if keyPlaced && visited[keyRoom] && visited[l.fromRoom] {
				// Key is reachable AND the room with the lock is reachable → unlock
				if unlocked[l.fromRoom] == nil {
					unlocked[l.fromRoom] = make(map[string]bool)
				}
				unlocked[l.fromRoom][l.direction] = true
				changed = true
			}
		}

		// Update reached set
		for r := range visited {
			if !reached[r] {
				reached[r] = true
				changed = true
			}
		}
	}

	return reached
}

func checkRoomReachability(spec *StorySpec, vr *ValidationResult) {
	reached := reachableRooms(spec)
	var unreachable []string
	for roomID := range spec.Rooms {
		if !reached[roomID] {
			unreachable = append(unreachable, roomID)
		}
	}
	sort.Strings(unreachable)
	for _, roomID := range unreachable {
		vr.addError(fmt.Sprintf("room %q is not reachable from start_room %q", roomID, spec.StartRoom))
	}
}

func checkKeyBeforeLock(spec *StorySpec, vr *ValidationResult) {
	// For each key_lock, check: is the key reachable WITHOUT the door it unlocks?
	// i.e., remove this specific lock and see if the key room is still reachable
	itemRoom := make(map[string]string)
	for roomID, room := range spec.Rooms {
		for _, itemID := range room.Items {
			itemRoom[itemID] = roomID
		}
	}

	for _, ps := range spec.Puzzles {
		if ps.Type != "key_lock" {
			continue
		}
		keyRoom, keyPlaced := itemRoom[ps.KeyItem]
		if !keyPlaced {
			continue // item not in any room, probably given by NPC or other mechanic
		}
		// Check if keyRoom is behind the locked door
		if keyRoom == ps.UnlockRoom {
			vr.addError(fmt.Sprintf("puzzle %q: key %q is in room %q which is behind the door it unlocks — key is unreachable", ps.ID, ps.KeyItem, keyRoom))
		}
	}
}

func checkWinItemGated(spec *StorySpec, vr *ValidationResult) {
	// Find win items and their rooms
	itemRoom := make(map[string]string)
	for roomID, room := range spec.Rooms {
		for _, itemID := range room.Items {
			itemRoom[itemID] = roomID
		}
	}

	// Find rooms that are behind key_locks (locked rooms)
	lockedRooms := make(map[string]bool)
	for _, ps := range spec.Puzzles {
		if ps.Type == "key_lock" && ps.UnlockRoom != "" {
			lockedRooms[ps.UnlockRoom] = true
		}
	}

	for _, ps := range spec.Puzzles {
		if ps.Type != "win_condition" {
			continue
		}
		winRoom, winPlaced := itemRoom[ps.WinItem]
		if !winPlaced {
			continue // not in a room (given by NPC dialogue, etc)
		}

		// Check if win item is directly accessible from start with no puzzle gate
		// Simple check: is the win room the start room, or reachable without any locks?
		if !lockedRooms[winRoom] && winRoom == spec.StartRoom {
			vr.addWarning(fmt.Sprintf("puzzle %q: win item %q is in the start room with no puzzle gate — player can win immediately", ps.ID, ps.WinItem))
		}
	}
}

func checkCircularDeps(spec *StorySpec, vr *ValidationResult) {
	// Build a dependency graph: puzzle A depends on puzzle B if A's key/item
	// is behind a door that B unlocks.
	// For key_lock puzzles: the key_item's room matters.
	itemRoom := make(map[string]string)
	for roomID, room := range spec.Rooms {
		for _, itemID := range room.Items {
			itemRoom[itemID] = roomID
		}
	}

	// Map: room → puzzle that unlocks it
	roomUnlockedBy := make(map[string]string)
	for _, ps := range spec.Puzzles {
		if ps.Type == "key_lock" && ps.UnlockRoom != "" {
			roomUnlockedBy[ps.UnlockRoom] = ps.ID
		}
	}

	// Build adjacency: puzzle → puzzles it depends on
	deps := make(map[string][]string)
	for _, ps := range spec.Puzzles {
		if ps.Type != "key_lock" {
			continue
		}
		keyRoom := itemRoom[ps.KeyItem]
		if dep, ok := roomUnlockedBy[keyRoom]; ok && dep != ps.ID {
			deps[ps.ID] = append(deps[ps.ID], dep)
		}
	}

	// Detect cycles via DFS
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make(map[string]int)
	var cycleNodes []string

	var dfs func(node string) bool
	dfs = func(node string) bool {
		color[node] = gray
		for _, dep := range deps[node] {
			if color[dep] == gray {
				cycleNodes = append(cycleNodes, node, dep)
				return true
			}
			if color[dep] == white {
				if dfs(dep) {
					return true
				}
			}
		}
		color[node] = black
		return false
	}

	for puzzleID := range deps {
		if color[puzzleID] == white {
			if dfs(puzzleID) {
				vr.addError(fmt.Sprintf("circular puzzle dependency detected involving: %s", strings.Join(cycleNodes, " → ")))
				return
			}
		}
	}
}

func checkDeadEnds(spec *StorySpec, vr *ValidationResult) {
	// A room with no connections at all (after accounting for locked directions
	// that might be added back) is a potential dead end.
	// Rooms that are unlock targets get a reverse connection from the expander,
	// so we only warn about rooms with truly no way out.
	unlockTargets := make(map[string]bool)
	for _, ps := range spec.Puzzles {
		if ps.UnlockRoom != "" {
			unlockTargets[ps.UnlockRoom] = true
		}
	}

	// Also check which rooms have incoming connections (they get reverse connections)
	hasIncoming := make(map[string]bool)
	for _, room := range spec.Rooms {
		for _, target := range room.Connections {
			hasIncoming[target] = true
		}
	}

	for roomID, room := range spec.Rooms {
		if roomID == spec.StartRoom {
			continue // start room is special
		}
		outgoing := len(room.Connections)
		if outgoing == 0 && !unlockTargets[roomID] {
			// This room has no connections at all and isn't an unlock target
			// (which would get a reverse connection added by the expander)
			vr.addWarning(fmt.Sprintf("room %q has no connections — potential dead end", roomID))
		}
	}
}

func checkMissingReverseConnections(spec *StorySpec, vr *ValidationResult) {
	// Check for one-way connections: A→B exists but B has no connection back to A.
	// Skip rooms that are key_lock unlock targets (they get reverse connections automatically).
	unlockTargets := make(map[string]bool)
	for _, ps := range spec.Puzzles {
		if ps.UnlockRoom != "" {
			unlockTargets[ps.UnlockRoom] = true
		}
	}

	for roomID, room := range spec.Rooms {
		for dir, targetID := range room.Connections {
			targetRoom, ok := spec.Rooms[targetID]
			if !ok {
				continue
			}
			// Check if target has any connection back to roomID
			hasReverse := false
			for _, backTarget := range targetRoom.Connections {
				if backTarget == roomID {
					hasReverse = true
					break
				}
			}
			if !hasReverse && !unlockTargets[targetID] {
				rev := reverseDirection[dir]
				if rev == "" {
					rev = "???"
				}
				vr.addWarning(fmt.Sprintf("room %q connects %s to %q, but %q has no connection back", roomID, dir, targetID, targetID))
			}
		}
	}
}

func checkOrphanDialogueNodes(spec *StorySpec, vr *ValidationResult) {
	for npcID, npc := range spec.Npcs {
		if len(npc.Dialogue) == 0 {
			continue
		}

		// Build set of all node IDs
		nodeIDs := make(map[string]bool)
		for _, dn := range npc.Dialogue {
			nodeIDs[dn.NodeID] = true
		}

		// Build set of reachable nodes from greeting + topic nodes
		reachable := make(map[string]bool)
		var walk func(nodeID string)
		walk = func(nodeID string) {
			if reachable[nodeID] {
				return
			}
			reachable[nodeID] = true
			for _, dn := range npc.Dialogue {
				if dn.NodeID == nodeID {
					for _, cs := range dn.Choices {
						if cs.NextNode != "" && cs.NextNode != "__exit__" {
							walk(cs.NextNode)
						}
					}
				}
			}
		}

		// Start from greeting nodes (NodeID containing "greeting" or first node)
		// and any nodes with a Topic set (reachable via "ask about")
		for _, dn := range npc.Dialogue {
			if dn.Topic != "" || strings.Contains(dn.NodeID, "greeting") {
				walk(dn.NodeID)
			}
		}
		// Also walk from the first node (default greeting)
		if len(npc.Dialogue) > 0 {
			walk(npc.Dialogue[0].NodeID)
		}

		for nodeID := range nodeIDs {
			if !reachable[nodeID] {
				vr.addWarning(fmt.Sprintf("npc %q: dialogue node %q is not reachable from any greeting or topic node", npcID, nodeID))
			}
		}
	}
}

func checkEndings(spec *StorySpec, vr *ValidationResult) {
	// Collect all variables that can be set in the story:
	// - Dialogue choice set_var fields
	// - Puzzle-implicit variables from expansion
	settableVars := make(map[string]bool)
	for _, npc := range spec.Npcs {
		for _, dn := range npc.Dialogue {
			for _, cs := range dn.Choices {
				if cs.SetVar != "" {
					if parts := strings.SplitN(cs.SetVar, "=", 2); len(parts) == 2 {
						settableVars[parts[0]] = true
					}
				}
			}
		}
	}
	// Puzzle-generated variables
	for _, ps := range spec.Puzzles {
		settableVars[ps.ID+"_unlocked"] = true
		settableVars[ps.ID+"_learned"] = true
		settableVars[ps.ID+"_solved"] = true
		settableVars[ps.ID+"_placed"] = true
		settableVars[ps.ID+"_combined"] = true
		settableVars[ps.ID+"_reached"] = true
		settableVars[ps.ID+"_started"] = true
		settableVars["puzzle."+ps.ID+".started"] = true
		settableVars["puzzle."+ps.ID+".complete"] = true
		settableVars["puzzle."+ps.ID+".failed"] = true
	}

	reached := reachableRooms(spec)
	itemRoom := make(map[string]string)
	for roomID, room := range spec.Rooms {
		for _, itemID := range room.Items {
			itemRoom[itemID] = roomID
		}
	}

	for _, ps := range spec.Puzzles {
		if ps.Type != "win_condition" {
			continue
		}

		// Check win item reachability for each win_condition
		if winRoom, placed := itemRoom[ps.WinItem]; placed {
			if !reached[winRoom] {
				vr.addError(fmt.Sprintf("puzzle %q: win item %q is in unreachable room %q — this ending can never be triggered", ps.ID, ps.WinItem, winRoom))
			}
		}

		// Check ending condition variables are settable
		for _, ending := range ps.Endings {
			for varKey := range ending.Conditions {
				if !settableVars[varKey] {
					vr.addWarning(fmt.Sprintf("puzzle %q: ending %q checks variable %q which is not set by any dialogue choice or puzzle — may be unreachable", ps.ID, ending.ID, varKey))
				}
			}
		}
	}
}

func checkSimpleTopicNpcs(spec *StorySpec, vr *ValidationResult) {
	for npcID, npc := range spec.Npcs {
		if len(npc.Dialogue) == 0 && len(npc.Topics) > 0 {
			vr.addWarning(fmt.Sprintf("npc %q uses simple topics — players cannot discover topic names without hints; consider using dialogue trees instead", npcID))
		}
	}
}

func checkItemRef(spec *StorySpec, puzzleID, field, itemID string) []string {
	if itemID == "" {
		return []string{fmt.Sprintf("puzzle %q: %s is required", puzzleID, field)}
	}
	if _, ok := spec.Items[itemID]; !ok {
		return []string{fmt.Sprintf("puzzle %q: %s references unknown item %q", puzzleID, field, itemID)}
	}
	return nil
}

func checkRoomRef(spec *StorySpec, puzzleID, field, roomID string) []string {
	if roomID == "" {
		return []string{fmt.Sprintf("puzzle %q: %s is required", puzzleID, field)}
	}
	if _, ok := spec.Rooms[roomID]; !ok {
		return []string{fmt.Sprintf("puzzle %q: %s references unknown room %q", puzzleID, field, roomID)}
	}
	return nil
}

// ValidateWorldDeep performs semantic validation on an expanded WorldDefinition.
func ValidateWorldDeep(world *engine.WorldDefinition, startRoom string) []string {
	var errs []string

	// Basic structural checks
	if err := world.ValidateWithStartRoom(startRoom); err != nil {
		errs = append(errs, err.Error())
		return errs // stop early on basic failures
	}

	// Collect all variables set by effects
	setVars := collectSetVars(world)

	// Add engine-implicit variables
	for puzzleID := range world.Puzzles {
		setVars["puzzle."+puzzleID+".step"] = true
		setVars["puzzle."+puzzleID+".complete"] = true
		setVars["puzzle."+puzzleID+".failed"] = true
		setVars["puzzle."+puzzleID+".started"] = true
		setVars["puzzle."+puzzleID+".start_turn"] = true
	}

	// Check variables in conditions that are never set
	checkedVars := collectCheckedVars(world)
	for v := range checkedVars {
		if !setVars[v] {
			errs = append(errs, fmt.Sprintf("variable %q is checked in conditions but never set by any effect", v))
		}
	}

	// Check items referenced in effects
	errs = append(errs, checkEffectItemRefs(world)...)

	// Check puzzles with no steps
	for id, puzzle := range world.Puzzles {
		if len(puzzle.Steps) == 0 {
			errs = append(errs, fmt.Sprintf("puzzle %q has no steps", id))
		}
	}

	// Check unlock_connection targets
	errs = append(errs, checkUnlockTargets(world)...)

	// Check dialogue tree NextNode references
	for npcID, npc := range world.Npcs {
		nodeIDs := make(map[string]bool)
		for _, dl := range npc.Dialogue {
			if dl.NodeID != "" {
				nodeIDs[dl.NodeID] = true
			}
		}
		for _, dl := range npc.Dialogue {
			for _, choice := range dl.Choices {
				if choice.NextNode != "" && choice.NextNode != "__exit__" {
					if !nodeIDs[choice.NextNode] {
						errs = append(errs, fmt.Sprintf("npc %q: dialogue choice references unknown node %q", npcID, choice.NextNode))
					}
				}
			}
		}
	}

	// Check win condition exists
	if !hasWinCondition(world) {
		errs = append(errs, "no effect sets game status to 'completed' — story has no win condition")
	}

	return errs
}

func collectSetVars(world *engine.WorldDefinition) map[string]bool {
	vars := make(map[string]bool)
	walkEffects(world, func(e engine.Effect) {
		if e.Type == "set_var" || e.Type == "increment_var" {
			vars[e.Key] = true
		}
	})
	return vars
}

func collectCheckedVars(world *engine.WorldDefinition) map[string]bool {
	vars := make(map[string]bool)
	walkConditions(world, func(c engine.Condition) {
		if c.Type == "var_equals" || c.Type == "var_gte" || c.Type == "var_lte" {
			vars[c.Key] = true
		}
	})
	return vars
}

func checkEffectItemRefs(world *engine.WorldDefinition) []string {
	var errs []string
	walkEffects(world, func(e engine.Effect) {
		switch e.Type {
		case "add_item", "remove_item":
			if _, ok := world.Items[e.Key]; !ok {
				errs = append(errs, fmt.Sprintf("effect %s references unknown item %q", e.Type, e.Key))
			}
		case "add_room_item", "remove_room_item":
			if _, ok := world.Items[e.Key]; !ok {
				errs = append(errs, fmt.Sprintf("effect %s references unknown item %q", e.Type, e.Key))
			}
		}
	})
	return errs
}

func checkUnlockTargets(world *engine.WorldDefinition) []string {
	var errs []string
	walkEffects(world, func(e engine.Effect) {
		if e.Type != "unlock_connection" {
			return
		}
		parts := strings.SplitN(e.Key, ".", 2)
		if len(parts) != 2 {
			errs = append(errs, fmt.Sprintf("unlock_connection has malformed key %q", e.Key))
			return
		}
		roomID := parts[0]
		if _, ok := world.Rooms[roomID]; !ok {
			errs = append(errs, fmt.Sprintf("unlock_connection key references unknown room %q", roomID))
		}
		if targetRoom, ok := e.Value.(string); ok && targetRoom != "" {
			if _, ok := world.Rooms[targetRoom]; !ok {
				errs = append(errs, fmt.Sprintf("unlock_connection value references unknown room %q", targetRoom))
			}
		}
	})
	return errs
}

func hasWinCondition(world *engine.WorldDefinition) bool {
	found := false
	walkEffects(world, func(e engine.Effect) {
		if e.Type == "set_status" {
			if v, ok := e.Value.(string); ok && v == "completed" {
				found = true
			}
		}
	})
	return found
}

// walkEffects calls fn for every effect in the world (items + puzzles + npcs).
func walkEffects(world *engine.WorldDefinition, fn func(engine.Effect)) {
	for _, item := range world.Items {
		for _, inter := range item.Interactions {
			for _, eff := range inter.Effects {
				fn(eff)
			}
		}
	}
	for _, puzzle := range world.Puzzles {
		for _, step := range puzzle.Steps {
			for _, eff := range step.Effects {
				fn(eff)
			}
		}
		for _, eff := range puzzle.FailureEffects {
			fn(eff)
		}
	}
	for _, npc := range world.Npcs {
		for _, dl := range npc.Dialogue {
			for _, eff := range dl.Effects {
				fn(eff)
			}
			for _, choice := range dl.Choices {
				for _, eff := range choice.Effects {
					fn(eff)
				}
			}
		}
	}
}

// walkConditions calls fn for every condition in the world.
func walkConditions(world *engine.WorldDefinition, fn func(engine.Condition)) {
	for _, item := range world.Items {
		for _, inter := range item.Interactions {
			for _, cond := range inter.Conditions {
				fn(cond)
			}
		}
		for _, cd := range item.ConditionalDescriptions {
			fn(cd.Condition)
		}
	}
	for _, room := range world.Rooms {
		for _, cd := range room.ConditionalDescriptions {
			fn(cd.Condition)
		}
		for _, h := range room.Hints {
			if h.Condition != nil {
				fn(*h.Condition)
			}
		}
	}
	for _, puzzle := range world.Puzzles {
		for _, step := range puzzle.Steps {
			for _, cond := range step.Conditions {
				fn(cond)
			}
		}
	}
	for _, npc := range world.Npcs {
		for _, dl := range npc.Dialogue {
			for _, cond := range dl.Conditions {
				fn(cond)
			}
			for _, choice := range dl.Choices {
				for _, cond := range choice.Conditions {
					fn(cond)
				}
			}
		}
		for _, cd := range npc.ConditionalDescriptions {
			fn(cd.Condition)
		}
	}
}
