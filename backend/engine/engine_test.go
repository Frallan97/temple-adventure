package engine

import (
	"strings"
	"testing"
)

func setupTestEngine(t *testing.T) (*Engine, *WorldState) {
	t.Helper()
	engine, err := NewEngine("../content")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	state := engine.World.NewWorldState("test-session")
	return engine, state
}

func TestWorldLoader(t *testing.T) {
	engine, _ := setupTestEngine(t)

	if len(engine.World.Rooms) != 3 {
		t.Errorf("Expected 3 rooms, got %d", len(engine.World.Rooms))
	}
	if len(engine.World.Puzzles) != 2 {
		t.Errorf("Expected 2 puzzles, got %d", len(engine.World.Puzzles))
	}
	if _, ok := engine.World.Rooms["entrance"]; !ok {
		t.Error("Missing entrance room")
	}
	if _, ok := engine.World.Rooms["sun_chamber"]; !ok {
		t.Error("Missing sun_chamber room")
	}
	if _, ok := engine.World.Rooms["treasure_vault"]; !ok {
		t.Error("Missing treasure_vault room")
	}
}

func TestCommandParser(t *testing.T) {
	parser := NewCommandParser()

	tests := []struct {
		input  string
		verb   string
		target string
	}{
		{"look", "look", ""},
		{"l", "look", ""},
		{"move north", "move", "north"},
		{"n", "move", "north"},
		{"take sword", "take", "sword"},
		{"get sword", "take", "sword"},
		{"i", "inventory", ""},
		{"inventory", "inventory", ""},
		{"help", "help", ""},
		{"?", "help", ""},
		{"push lever", "push", "lever"},
		{"use key", "use", "key"},
		{"examine tablet", "look", "tablet"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cmd := parser.Parse(tt.input)
			if cmd.Verb != tt.verb {
				t.Errorf("Parse(%q): verb = %q, want %q", tt.input, cmd.Verb, tt.verb)
			}
			if cmd.Target != tt.target {
				t.Errorf("Parse(%q): target = %q, want %q", tt.input, cmd.Target, tt.target)
			}
		})
	}
}

func TestLookCommand(t *testing.T) {
	engine, state := setupTestEngine(t)

	result := engine.ProcessCommand(state, "look")
	if !strings.Contains(result.Text, "Temple Entrance") && !strings.Contains(result.Text, "ancient Inca temple") {
		t.Errorf("Look should describe entrance, got: %s", result.Text)
	}
	if result.GameOver {
		t.Error("Game should not be over")
	}
}

func TestMoveBlocked(t *testing.T) {
	engine, state := setupTestEngine(t)

	result := engine.ProcessCommand(state, "move north")
	if !strings.Contains(result.Text, "can't go") {
		t.Errorf("Should not be able to go north initially, got: %s", result.Text)
	}
}

func TestTakeItem(t *testing.T) {
	engine, state := setupTestEngine(t)

	result := engine.ProcessCommand(state, "take flint")
	if !strings.Contains(result.Text, "pick up") {
		t.Errorf("Should pick up flint, got: %s", result.Text)
	}
	if !state.Inventory["flint"] {
		t.Error("Flint should be in inventory")
	}
}

func TestInventory(t *testing.T) {
	engine, state := setupTestEngine(t)

	result := engine.ProcessCommand(state, "inventory")
	if !strings.Contains(result.Text, "nothing") {
		t.Errorf("Should carry nothing, got: %s", result.Text)
	}

	engine.ProcessCommand(state, "take flint")
	result = engine.ProcessCommand(state, "inventory")
	if !strings.Contains(result.Text, "flint") {
		t.Errorf("Should show flint, got: %s", result.Text)
	}
}

func TestFullGameWin(t *testing.T) {
	engine, state := setupTestEngine(t)

	// Step 1: Examine the stone tablet (sets read_tablet variable)
	result := engine.ProcessCommand(state, "look tablet")
	if !strings.Contains(result.Text, "serpent, sun, serpent") {
		t.Fatalf("Should show clue, got: %s", result.Text)
	}

	// Step 2: Turn the glyph panel (using recalled tablet clue)
	result = engine.ProcessCommand(state, "turn panel")
	if !strings.Contains(result.Text, "serpent, sun, serpent") {
		t.Fatalf("Should reference the sequence, got: %s", result.Text)
	}
	v, ok := state.Variables["glyph_sequence_correct"]
	if !ok || !v.BoolVal {
		t.Fatal("glyph_sequence_correct should be true")
	}

	// Step 3: Use the glyph panel to open the gate
	result = engine.ProcessCommand(state, "use panel")
	if !strings.Contains(result.Text, "door grinds open") {
		t.Fatalf("Should open the door, got: %s", result.Text)
	}

	// Gate puzzle should now be complete (both steps satisfied)
	// Check we can go north
	result = engine.ProcessCommand(state, "move north")
	if state.CurrentRoom != "sun_chamber" {
		t.Fatalf("Should be in sun_chamber, in %s. Result: %s", state.CurrentRoom, result.Text)
	}

	// Step 4: Take the golden mirror
	result = engine.ProcessCommand(state, "take mirror")
	if !state.Inventory["golden_mirror"] {
		t.Fatalf("Should have mirror, got: %s", result.Text)
	}

	// Step 5: Turn the sun disc to activate the beam
	result = engine.ProcessCommand(state, "turn disc")
	if !strings.Contains(result.Text, "beam of light") {
		t.Fatalf("Should describe beam, got: %s", result.Text)
	}
	v, ok = state.Variables["beam_active"]
	if !ok || !v.BoolVal {
		t.Fatal("beam_active should be true")
	}

	// Step 6: Use the mirror to redirect the beam
	result = engine.ProcessCommand(state, "use mirror")
	if !strings.Contains(result.Text, "mirror") {
		t.Fatalf("Should place mirror, got: %s", result.Text)
	}

	// Light puzzle should complete — check for completion text
	if !strings.Contains(result.Text, "staircase") {
		t.Fatalf("Should describe vault opening, got: %s", result.Text)
	}

	// Step 7: Go down to the treasure vault
	result = engine.ProcessCommand(state, "move down")
	if state.CurrentRoom != "treasure_vault" {
		t.Fatalf("Should be in treasure_vault, in %s. Result: %s", state.CurrentRoom, result.Text)
	}

	// Step 8: Take the crown to win
	result = engine.ProcessCommand(state, "take crown")
	if !strings.Contains(result.Text, "Sun Crown") {
		t.Fatalf("Should describe taking the crown, got: %s", result.Text)
	}
	if state.Status != "completed" {
		t.Fatalf("Game should be completed, status: %s", state.Status)
	}
	if !result.GameOver {
		t.Fatal("GameOver should be true")
	}
}

func TestTimedPuzzleFailure(t *testing.T) {
	engine, state := setupTestEngine(t)

	// Open the gate first
	engine.ProcessCommand(state, "look tablet")
	engine.ProcessCommand(state, "turn panel")
	engine.ProcessCommand(state, "use panel")
	engine.ProcessCommand(state, "move north")

	// Activate the beam (starts the timer)
	engine.ProcessCommand(state, "turn disc")

	// Verify the timer started
	_, ok := state.Variables["puzzle.light_puzzle.start_turn"]
	if !ok {
		t.Fatal("Timer should have started")
	}

	// Waste turns until the timer expires (limit is 8)
	for i := 0; i < 9; i++ {
		engine.ProcessCommand(state, "look")
	}

	// The timed failure should have triggered
	failed, ok := state.Variables["puzzle.light_puzzle.failed"]
	if !ok || !failed.BoolVal {
		t.Fatal("Puzzle should have failed after timer expired")
	}

	if state.CurrentRoom != "entrance" {
		t.Fatalf("Should have been moved back to entrance, in %s", state.CurrentRoom)
	}

	// Verify the entrance-north connection is now blocked
	connections := GetRoomConnections(state, engine.World, "entrance")
	if _, ok := connections["north"]; ok {
		t.Fatal("North connection from entrance should be blocked after collapse")
	}
}

func TestHelpCommand(t *testing.T) {
	engine, state := setupTestEngine(t)

	result := engine.ProcessCommand(state, "help")
	if !strings.Contains(result.Text, "Available commands") {
		t.Errorf("Help should list commands, got: %s", result.Text)
	}
}

func TestUnknownCommand(t *testing.T) {
	engine, state := setupTestEngine(t)

	result := engine.ProcessCommand(state, "dance")
	if !strings.Contains(result.Text, "don't understand") {
		t.Errorf("Should not understand 'dance', got: %s", result.Text)
	}
}

func TestDropItem(t *testing.T) {
	engine, state := setupTestEngine(t)

	engine.ProcessCommand(state, "take flint")
	result := engine.ProcessCommand(state, "drop flint")
	if !strings.Contains(result.Text, "drop") {
		t.Errorf("Should drop flint, got: %s", result.Text)
	}
	if state.Inventory["flint"] {
		t.Error("Flint should not be in inventory after dropping")
	}
}
