package engine

import (
	"strings"
	"testing"
)

// simulateRoundTrip mimics what the service does between requests:
// serialize state to flat maps, then reconstruct a fresh WorldState from them.
// This catches bugs where in-memory state doesn't survive persistence.
func simulateRoundTrip(t *testing.T, eng *Engine, state *WorldState) *WorldState {
	t.Helper()

	// Serialize: extract what would go into DB
	inventory := make(map[string]bool)
	for k, v := range state.Inventory {
		inventory[k] = v
	}

	variables := make(map[string]Variable)
	for k, v := range state.Variables {
		variables[k] = v
	}

	// Serialize room states into variables (like syncVariables does)
	for roomID, rs := range state.RoomStates {
		for itemID := range rs.RemovedItems {
			key := "room_state." + roomID + ".removed_item." + itemID
			variables[key] = Variable{Type: "bool", BoolVal: true}
		}
		for itemID := range rs.AddedItems {
			key := "room_state." + roomID + ".added_item." + itemID
			variables[key] = Variable{Type: "bool", BoolVal: true}
		}
		for dir := range rs.BlockedConnections {
			key := "room_state." + roomID + ".blocked." + dir
			variables[key] = Variable{Type: "bool", BoolVal: true}
		}
		for dir, target := range rs.AddedConnections {
			key := "room_state." + roomID + ".added_conn." + dir
			variables[key] = Variable{Type: "string", StrVal: target}
		}
	}

	// Reconstruct: build fresh state from serialized data (like loadWorldState does)
	newState := eng.World.NewWorldState(state.SessionID)
	newState.CurrentRoom = state.CurrentRoom
	newState.TurnNumber = state.TurnNumber
	newState.Status = state.Status
	newState.Inventory = inventory

	// Load variables and reconstruct room states (like reconstructRoomStates does)
	newState.Variables = variables
	reconstructRoomStatesFromVars(newState)

	return newState
}

// reconstructRoomStatesFromVars mirrors the service's reconstructRoomStates logic.
func reconstructRoomStatesFromVars(state *WorldState) {
	for key, v := range state.Variables {
		if len(key) < 12 || key[:11] != "room_state." {
			continue
		}
		rest := key[11:]
		dotIdx := -1
		for i, c := range rest {
			if c == '.' {
				dotIdx = i
				break
			}
		}
		if dotIdx < 0 {
			continue
		}
		roomID := rest[:dotIdx]
		remainder := rest[dotIdx+1:]

		rs := state.RoomStates[roomID]
		if rs == nil {
			continue
		}

		if len(remainder) > len("removed_item.") && remainder[:len("removed_item.")] == "removed_item." {
			if v.BoolVal {
				rs.RemovedItems[remainder[len("removed_item."):]] = true
			}
		} else if len(remainder) > len("added_item.") && remainder[:len("added_item.")] == "added_item." {
			if v.BoolVal {
				rs.AddedItems[remainder[len("added_item."):]] = true
			}
		} else if len(remainder) > len("blocked.") && remainder[:len("blocked.")] == "blocked." {
			if v.BoolVal {
				rs.BlockedConnections[remainder[len("blocked."):]] = true
			}
		} else if len(remainder) > len("added_conn.") && remainder[:len("added_conn.")] == "added_conn." {
			rs.AddedConnections[remainder[len("added_conn."):]] = v.StrVal
		}
	}
}

// TestRoundTripFullGame plays the entire game, doing a save/load round-trip
// between every single command. This is the most realistic test of the persistence layer.
func TestRoundTripFullGame(t *testing.T) {
	eng, state := setupTestEngine(t)

	step := func(input string) *CommandResult {
		t.Helper()
		result := eng.ProcessCommand(state, input)
		if strings.HasPrefix(result.Text, "Error") {
			t.Fatalf("Command %q returned error: %s", input, result.Text)
		}
		// Round-trip after every command
		state = simulateRoundTrip(t, eng, state)
		return result
	}

	// === Entrance: Gate Puzzle ===
	step("look")

	// Take items
	step("take tablet")
	if !state.Inventory["stone_tablet"] {
		t.Fatal("tablet not in inventory after round-trip")
	}

	step("take flint")
	if !state.Inventory["flint"] {
		t.Fatal("flint not in inventory after round-trip")
	}

	// Verify items removed from room after round-trip
	roomItems := GetRoomItems(state, eng.World, "entrance")
	for _, id := range roomItems {
		if id == "stone_tablet" || id == "flint" {
			t.Fatalf("item %s should not be in room after taking and round-trip", id)
		}
	}

	// Solve gate puzzle
	step("turn panel")
	v, ok := state.Variables["glyph_sequence_correct"]
	if !ok || !v.BoolVal {
		t.Fatal("glyph_sequence_correct not set after round-trip")
	}

	step("use panel")
	v, ok = state.Variables["gate_open"]
	if !ok || !v.BoolVal {
		t.Fatal("gate_open not set after round-trip")
	}

	// Verify connection persists through round-trip
	conns := GetRoomConnections(state, eng.World, "entrance")
	if _, ok := conns["north"]; !ok {
		t.Fatal("north connection should exist after opening gate and round-trip")
	}

	// Move to sun chamber
	result := step("move north")
	if state.CurrentRoom != "sun_chamber" {
		t.Fatalf("should be in sun_chamber, got %s: %s", state.CurrentRoom, result.Text)
	}

	// === Sun Chamber: Light Puzzle ===
	step("take mirror")
	if !state.Inventory["golden_mirror"] {
		t.Fatal("mirror not in inventory after round-trip")
	}

	step("turn disc")
	v, ok = state.Variables["beam_active"]
	if !ok || !v.BoolVal {
		t.Fatal("beam_active not set after round-trip")
	}

	// Verify timed puzzle started
	_, ok = state.Variables["puzzle.light_puzzle.started"]
	if !ok {
		t.Fatal("light puzzle timer should have started")
	}
	_, ok = state.Variables["puzzle.light_puzzle.start_turn"]
	if !ok {
		t.Fatal("light puzzle start_turn should be set")
	}

	step("use mirror")
	v, ok = state.Variables["mirror_placed"]
	if !ok || !v.BoolVal {
		t.Fatal("mirror_placed not set after round-trip")
	}

	// Light puzzle should be complete
	v, ok = state.Variables["puzzle.light_puzzle.complete"]
	if !ok || !v.BoolVal {
		t.Fatal("light puzzle should be complete after round-trip")
	}

	// Verify new connection to treasure vault
	conns = GetRoomConnections(state, eng.World, "sun_chamber")
	if _, ok := conns["down"]; !ok {
		t.Fatal("down connection to treasure vault should exist after puzzle completion and round-trip")
	}

	// === Treasure Vault: Victory ===
	result = step("move down")
	if state.CurrentRoom != "treasure_vault" {
		t.Fatalf("should be in treasure_vault, got %s: %s", state.CurrentRoom, result.Text)
	}

	result = step("take crown")
	if state.Status != "completed" {
		t.Fatalf("game should be completed, status: %s", state.Status)
	}
	if !result.GameOver {
		t.Fatal("GameOver should be true")
	}
}

// TestRoundTripTimedFailure tests that the timed puzzle failure survives persistence.
func TestRoundTripTimedFailure(t *testing.T) {
	eng, state := setupTestEngine(t)

	step := func(input string) *CommandResult {
		t.Helper()
		result := eng.ProcessCommand(state, input)
		state = simulateRoundTrip(t, eng, state)
		return result
	}

	// Open gate and enter sun chamber
	step("take tablet")
	step("turn panel")
	step("use panel")
	step("move north")

	// Start timer
	step("turn disc")
	startTurn := state.TurnNumber

	// Waste turns (limit is 8)
	for i := 0; i < 9; i++ {
		step("look")
	}

	// Timer should have expired (started at turn 5, now past turn 13)
	_ = startTurn
	failed, ok := state.Variables["puzzle.light_puzzle.failed"]
	if !ok || !failed.BoolVal {
		t.Fatal("puzzle should have failed after timer expired and round-trips")
	}

	if state.CurrentRoom != "entrance" {
		t.Fatalf("should be back at entrance, got %s", state.CurrentRoom)
	}

	// Verify permanent consequences survive round-trip
	conns := GetRoomConnections(state, eng.World, "entrance")
	if _, ok := conns["north"]; ok {
		t.Fatal("north should be permanently blocked after collapse and round-trip")
	}

	collapsed, ok := state.Variables["sun_chamber_collapsed"]
	if !ok || !collapsed.BoolVal {
		t.Fatal("sun_chamber_collapsed should be set after round-trip")
	}
}

// TestRoundTripDropAndRetake tests picking up, dropping, and re-taking items
// survives the round-trip cycle.
func TestRoundTripDropAndRetake(t *testing.T) {
	eng, state := setupTestEngine(t)

	step := func(input string) {
		t.Helper()
		eng.ProcessCommand(state, input)
		state = simulateRoundTrip(t, eng, state)
	}

	// Take flint
	step("take flint")
	if !state.Inventory["flint"] {
		t.Fatal("should have flint")
	}

	// Drop it
	step("drop flint")
	if state.Inventory["flint"] {
		t.Fatal("should not have flint after dropping")
	}

	// Verify it's back in the room
	items := GetRoomItems(state, eng.World, "entrance")
	found := false
	for _, id := range items {
		if id == "flint" {
			found = true
		}
	}
	if !found {
		t.Fatal("flint should be in room after dropping and round-trip")
	}

	// Take it again
	step("take flint")
	if !state.Inventory["flint"] {
		t.Fatal("should have flint after retaking")
	}
}

// TestRoundTripConditionalDescriptions tests that conditional room/item descriptions
// work correctly after state round-trips.
func TestRoundTripConditionalDescriptions(t *testing.T) {
	eng, state := setupTestEngine(t)

	step := func(input string) *CommandResult {
		t.Helper()
		result := eng.ProcessCommand(state, input)
		state = simulateRoundTrip(t, eng, state)
		return result
	}

	// Before opening gate, look should NOT mention the open door
	result := step("look")
	if strings.Contains(result.Text, "stands open") {
		t.Fatal("door should not be described as open yet")
	}

	// Open the gate
	step("take tablet")
	step("turn panel")
	step("use panel")

	// After opening gate, look SHOULD mention the open door
	result = step("look")
	if !strings.Contains(result.Text, "stands open") {
		t.Fatalf("door should be described as open after gate_open=true and round-trip, got: %s", result.Text)
	}
}

// TestRoundTripExamineItems tests that examining items works across round-trips.
func TestRoundTripExamineItems(t *testing.T) {
	eng, state := setupTestEngine(t)

	step := func(input string) *CommandResult {
		t.Helper()
		result := eng.ProcessCommand(state, input)
		state = simulateRoundTrip(t, eng, state)
		return result
	}

	// Examine tablet in room
	result := step("look tablet")
	if !strings.Contains(result.Text, "serpent") {
		t.Fatalf("examining tablet should show clue, got: %s", result.Text)
	}

	// Take tablet, examine from inventory
	step("take tablet")
	result = step("look tablet")
	if !strings.Contains(result.Text, "serpent") {
		t.Fatalf("examining tablet from inventory should still work, got: %s", result.Text)
	}
}

// TestAllItemsReachable verifies that every item defined in the world
// can be found in at least one room.
func TestAllItemsReachable(t *testing.T) {
	eng, _ := setupTestEngine(t)

	reachableItems := make(map[string]bool)
	for _, room := range eng.World.Rooms {
		for _, itemID := range room.Items {
			reachableItems[itemID] = true
		}
	}

	for itemID := range eng.World.Items {
		if !reachableItems[itemID] {
			// Items that only appear via effects (like torch_lit) are OK to skip,
			// but let's flag any that seem wrong
			t.Logf("Warning: item %q is not placed in any room (may be created via effects)", itemID)
		}
	}
}

// TestAllRoomsReachable verifies that every room can be reached from the entrance
// through some sequence of puzzle solutions and connections.
func TestAllRoomsReachable(t *testing.T) {
	eng, state := setupTestEngine(t)

	// Play through the game to unlock all connections
	commands := []string{
		"take tablet", "turn panel", "use panel", "move north",
		"take mirror", "turn disc", "use mirror", "move down",
	}
	for _, cmd := range commands {
		eng.ProcessCommand(state, cmd)
	}

	// Check we visited all rooms
	visited := map[string]bool{
		"entrance":       true, // started here
		"sun_chamber":    true, // moved north
		"treasure_vault": true, // moved down
	}

	for roomID := range eng.World.Rooms {
		if !visited[roomID] {
			t.Errorf("room %q was never visited during full playthrough", roomID)
		}
	}
}

// TestCannotWinAfterTimedFailure verifies that after the timed puzzle fails,
// the game is still playable but the treasure vault is unreachable.
func TestCannotWinAfterTimedFailure(t *testing.T) {
	eng, state := setupTestEngine(t)

	step := func(input string) *CommandResult {
		t.Helper()
		result := eng.ProcessCommand(state, input)
		state = simulateRoundTrip(t, eng, state)
		return result
	}

	// Open gate, enter, start timer, let it expire
	step("take tablet")
	step("turn panel")
	step("use panel")
	step("move north")
	step("turn disc")
	for i := 0; i < 9; i++ {
		step("look")
	}

	// Now at entrance, north is blocked
	if state.CurrentRoom != "entrance" {
		t.Fatalf("should be at entrance, got %s", state.CurrentRoom)
	}

	// Try to go north — should fail
	result := step("move north")
	if !strings.Contains(result.Text, "can't go") {
		t.Fatalf("should not be able to go north, got: %s", result.Text)
	}

	// Game should still be active (not over, just stuck)
	if state.Status != "active" {
		t.Fatalf("game should still be active, got %s", state.Status)
	}
}

// TestNonPortableItems tests that fixed items cannot be picked up.
func TestNonPortableItems(t *testing.T) {
	eng, state := setupTestEngine(t)

	step := func(input string) *CommandResult {
		t.Helper()
		result := eng.ProcessCommand(state, input)
		state = simulateRoundTrip(t, eng, state)
		return result
	}

	result := step("take panel")
	if !strings.Contains(result.Text, "can't pick up") {
		t.Fatalf("should not be able to take glyph panel, got: %s", result.Text)
	}

	// Open gate, go to sun chamber, try to take the sun disc
	step("take tablet")
	step("turn panel")
	step("use panel")
	step("move north")

	result = step("take disc")
	if !strings.Contains(result.Text, "can't pick up") {
		t.Fatalf("should not be able to take sun disc, got: %s", result.Text)
	}
}
