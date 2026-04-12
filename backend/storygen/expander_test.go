package storygen

import (
	"testing"
)

func TestExpandKeyLock(t *testing.T) {
	spec := &StorySpec{
		StartRoom: "hall",
		Rooms: map[string]RoomSpec{
			"hall":   {Name: "Hall", Description: "A grand hall.", Items: []string{"rusty_key", "iron_door"}},
			"cellar": {Name: "Cellar", Description: "A dark cellar."},
		},
		Items: map[string]ItemSpec{
			"rusty_key": {Name: "rusty key", Aliases: []string{"key"}, Portable: true},
			"iron_door": {Name: "iron door", Aliases: []string{"door"}, Portable: false},
		},
		Puzzles: []PuzzleSpec{
			{
				ID: "door_puzzle", Type: "key_lock", Name: "Unlock the Door",
				Description: "Use the key on the door.",
				Room: "hall", KeyItem: "rusty_key", LockTarget: "iron_door",
				UnlockDirection: "north", UnlockRoom: "cellar",
				CompletionText: "The door creaks open!", LockFailText: "It's locked.",
			},
		},
	}

	world, err := Expand(spec)
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	// Puzzle should exist
	puzzle := world.Puzzles["door_puzzle"]
	if puzzle == nil {
		t.Fatal("puzzle door_puzzle not created")
	}
	if len(puzzle.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(puzzle.Steps))
	}

	// Connection should be removed from base room (starts locked)
	if _, ok := world.Rooms["hall"].Connections["north"]; ok {
		t.Fatal("north connection should be removed (locked)")
	}

	// Reverse connection should exist
	if world.Rooms["cellar"].Connections["south"] != "hall" {
		t.Fatal("cellar should have south→hall connection")
	}

	// Item should have interactions
	door := world.Items["iron_door"]
	if len(door.Interactions) < 1 {
		t.Fatal("door should have interactions")
	}
	if door.Interactions[0].Verb != "use" {
		t.Fatalf("expected verb 'use', got %q", door.Interactions[0].Verb)
	}
	if len(door.Interactions[0].Conditions) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(door.Interactions[0].Conditions))
	}

	// Room should reference puzzle
	if len(world.Rooms["hall"].Puzzles) != 1 || world.Rooms["hall"].Puzzles[0] != "door_puzzle" {
		t.Fatal("hall should reference door_puzzle")
	}
}

func TestExpandExamineLearn(t *testing.T) {
	spec := &StorySpec{
		StartRoom: "library",
		Rooms: map[string]RoomSpec{
			"library": {Name: "Library", Description: "Dusty bookshelves.", Items: []string{"old_book", "cipher_wheel"}},
		},
		Items: map[string]ItemSpec{
			"old_book":     {Name: "old book", Aliases: []string{"book"}, Portable: true},
			"cipher_wheel": {Name: "cipher wheel", Aliases: []string{"wheel", "cipher"}, Portable: false},
		},
		Puzzles: []PuzzleSpec{
			{
				ID: "cipher_puzzle", Type: "examine_learn", Name: "Crack the Cipher",
				Description: "Read the book to learn the code, then use the wheel.",
				Room: "library",
				SourceItem: "old_book", SourceLearnText: "The book reveals a three-digit code: 4-7-2.",
				TargetItem: "cipher_wheel", TargetVerb: "turn",
				TargetSuccessText: "The wheel clicks into place!", TargetFailText: "You don't know the code.",
			},
		},
	}

	world, err := Expand(spec)
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	puzzle := world.Puzzles["cipher_puzzle"]
	if puzzle == nil {
		t.Fatal("puzzle not created")
	}
	if len(puzzle.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(puzzle.Steps))
	}

	// Book should have examine interaction
	book := world.Items["old_book"]
	found := false
	for _, inter := range book.Interactions {
		if inter.Verb == "examine" {
			found = true
			if len(inter.Effects) == 0 {
				t.Fatal("examine should set learned var")
			}
		}
	}
	if !found {
		t.Fatal("book should have examine interaction")
	}

	// Wheel should have turn interaction with condition
	wheel := world.Items["cipher_wheel"]
	found = false
	for _, inter := range wheel.Interactions {
		if inter.Verb == "turn" {
			found = true
			if len(inter.Conditions) == 0 {
				t.Fatal("turn should require learned var")
			}
			if inter.FailResponse != "You don't know the code." {
				t.Fatalf("wrong fail response: %q", inter.FailResponse)
			}
		}
	}
	if !found {
		t.Fatal("wheel should have turn interaction")
	}
}

func TestExpandExamineLearnWithUnlock(t *testing.T) {
	spec := &StorySpec{
		StartRoom: "entrance",
		Rooms: map[string]RoomSpec{
			"entrance": {
				Name: "Entrance", Description: "A temple entrance.",
				Items:                  []string{"tablet", "panel"},
				DescriptionAfterPuzzle: "The door stands open.",
			},
			"inner": {Name: "Inner Chamber", Description: "A dark chamber."},
		},
		Items: map[string]ItemSpec{
			"tablet": {Name: "stone tablet", Aliases: []string{"tablet"}, Portable: true},
			"panel":  {Name: "glyph panel", Aliases: []string{"panel"}, Portable: false},
		},
		Puzzles: []PuzzleSpec{
			{
				ID: "gate_puzzle", Type: "examine_learn", Name: "The Gate",
				Room:        "entrance",
				SourceItem:  "tablet", SourceLearnText: "serpent, sun, serpent",
				TargetItem:  "panel", TargetVerb: "turn",
				TargetSuccessText: "Glyphs align!", TargetFailText: "No clue.",
				UnlockDirection: "north", UnlockRoom: "inner",
				LockVerb: "use", CompletionText: "The door opens!",
				LockFailText: "Not ready.",
			},
		},
	}

	world, err := Expand(spec)
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	puzzle := world.Puzzles["gate_puzzle"]
	if len(puzzle.Steps) != 3 {
		t.Fatalf("expected 3 steps (learn, solve, unlock), got %d", len(puzzle.Steps))
	}

	// North should be locked
	if _, ok := world.Rooms["entrance"].Connections["north"]; ok {
		t.Fatal("north should be locked initially")
	}

	// Inner should have south→entrance
	if world.Rooms["inner"].Connections["south"] != "entrance" {
		t.Fatal("inner should have south→entrance")
	}

	// Panel should have both turn (solve) and use (unlock) interactions
	panel := world.Items["panel"]
	verbs := map[string]bool{}
	for _, inter := range panel.Interactions {
		verbs[inter.Verb] = true
	}
	if !verbs["turn"] || !verbs["use"] {
		t.Fatalf("panel should have turn and use interactions, got %v", verbs)
	}

	// Conditional description should be added
	if len(world.Rooms["entrance"].ConditionalDescriptions) == 0 {
		t.Fatal("entrance should have conditional description")
	}
	cd := world.Rooms["entrance"].ConditionalDescriptions[0]
	if !cd.Replace {
		t.Fatal("conditional description should be replace:true")
	}
}

func TestExpandFetchQuest(t *testing.T) {
	spec := &StorySpec{
		StartRoom: "garden",
		Rooms: map[string]RoomSpec{
			"garden": {Name: "Garden", Description: "A garden.", Items: []string{"gem", "statue"}},
			"vault":  {Name: "Vault", Description: "A vault."},
		},
		Items: map[string]ItemSpec{
			"gem":    {Name: "ruby gem", Aliases: []string{"gem", "ruby"}, Portable: true},
			"statue": {Name: "statue", Aliases: []string{"statue"}, Portable: false},
		},
		Puzzles: []PuzzleSpec{
			{
				ID: "gem_puzzle", Type: "fetch_quest", Name: "Place the Gem",
				Description: "Place the gem in the statue.",
				Room: "garden", FetchItem: "gem", FetchRoom: "garden",
				FetchTarget: "statue", FetchVerb: "use",
				FetchSuccessText: "The gem fits perfectly!",
				FetchConsumeItem: true,
				UnlockDirection: "east", UnlockRoom: "vault",
				CompletionText: "A door opens to the east!",
			},
		},
	}

	world, err := Expand(spec)
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	puzzle := world.Puzzles["gem_puzzle"]
	if puzzle == nil {
		t.Fatal("puzzle not created")
	}

	// East should be locked
	if _, ok := world.Rooms["garden"].Connections["east"]; ok {
		t.Fatal("east should be locked")
	}

	// Gem should have use interaction
	gem := world.Items["gem"]
	if len(gem.Interactions) == 0 {
		t.Fatal("gem should have interaction")
	}
	inter := gem.Interactions[0]
	if inter.Verb != "use" {
		t.Fatalf("expected 'use', got %q", inter.Verb)
	}
	// Should consume item
	hasRemove := false
	for _, eff := range inter.Effects {
		if eff.Type == "remove_item" && eff.Key == "gem" {
			hasRemove = true
		}
	}
	if !hasRemove {
		t.Fatal("fetch with consume should remove item")
	}
}

func TestExpandTimedChallenge(t *testing.T) {
	spec := &StorySpec{
		StartRoom: "chamber",
		Rooms: map[string]RoomSpec{
			"chamber":  {Name: "Chamber", Description: "A chamber.", Items: []string{"lever", "mirror"}},
			"entrance": {Name: "Entrance", Description: "The entrance."},
			"vault":    {Name: "Vault", Description: "The vault."},
		},
		Items: map[string]ItemSpec{
			"lever":  {Name: "lever", Aliases: []string{"lever"}, Portable: false},
			"mirror": {Name: "mirror", Aliases: []string{"mirror"}, Portable: true},
		},
		Puzzles: []PuzzleSpec{
			{
				ID: "timed_puzzle", Type: "timed_challenge", Name: "Race Against Time",
				Description: "Solve before time runs out!",
				Room: "chamber",
				TriggerItem: "lever", TriggerVerb: "pull",
				TriggerText: "A countdown begins!", TurnLimit: 5,
				FailureText: "Time's up! The room collapses.",
				FailureEffects: []FailureEffectSpec{
					{Type: "move_player", Room: "entrance"},
					{Type: "lock_connection", Room: "entrance", Direction: "north"},
				},
				FetchItem: "mirror", FetchVerb: "use", FetchRoom: "chamber",
				FetchSuccessText: "You place the mirror just in time!",
				FetchConsumeItem: true,
				UnlockDirection: "down", UnlockRoom: "vault",
				CompletionText: "A staircase appears!",
			},
		},
	}

	world, err := Expand(spec)
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	puzzle := world.Puzzles["timed_puzzle"]
	if puzzle == nil {
		t.Fatal("puzzle not created")
	}
	if puzzle.TimedWindow == nil {
		t.Fatal("should have timed window")
	}
	if puzzle.TimedWindow.TurnLimit != 5 {
		t.Fatalf("expected turn limit 5, got %d", puzzle.TimedWindow.TurnLimit)
	}
	if len(puzzle.Steps) != 2 {
		t.Fatalf("expected 2 steps (activate, solve), got %d", len(puzzle.Steps))
	}
	if len(puzzle.FailureEffects) != 2 {
		t.Fatalf("expected 2 failure effects, got %d", len(puzzle.FailureEffects))
	}

	// Lever should have pull interactions
	lever := world.Items["lever"]
	pullCount := 0
	for _, inter := range lever.Interactions {
		if inter.Verb == "pull" {
			pullCount++
		}
	}
	if pullCount != 2 {
		t.Fatalf("expected 2 pull interactions (start + already active), got %d", pullCount)
	}

	// Mirror should have use interaction requiring timer started
	mirror := world.Items["mirror"]
	found := false
	for _, inter := range mirror.Interactions {
		if inter.Verb == "use" {
			found = true
			hasTimerCond := false
			for _, c := range inter.Conditions {
				if c.Key == "puzzle.timed_puzzle.started" {
					hasTimerCond = true
				}
			}
			if !hasTimerCond {
				t.Fatal("fetch interaction should require timer started")
			}
		}
	}
	if !found {
		t.Fatal("mirror should have use interaction")
	}
}

func TestExpandWinCondition(t *testing.T) {
	spec := &StorySpec{
		StartRoom: "throne",
		Rooms: map[string]RoomSpec{
			"throne": {Name: "Throne Room", Description: "A throne.", Items: []string{"crown"}},
		},
		Items: map[string]ItemSpec{
			"crown": {Name: "golden crown", Aliases: []string{"crown"}, Portable: true},
		},
		Puzzles: []PuzzleSpec{
			{
				ID: "win", Type: "win_condition", Name: "Claim the Crown",
				Room: "throne", WinItem: "crown", WinVerb: "take",
				WinText: "You win!",
			},
		},
	}

	world, err := Expand(spec)
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	// No puzzle def for win_condition
	if len(world.Puzzles) != 0 {
		t.Fatalf("win_condition should not create a puzzle def, got %d", len(world.Puzzles))
	}

	// Crown should have take interaction with set_status
	crown := world.Items["crown"]
	if len(crown.Interactions) == 0 {
		t.Fatal("crown should have interaction")
	}
	hasStatus := false
	for _, eff := range crown.Interactions[0].Effects {
		if eff.Type == "set_status" {
			hasStatus = true
			if eff.Value != "completed" {
				t.Fatalf("expected 'completed', got %v", eff.Value)
			}
		}
	}
	if !hasStatus {
		t.Fatal("win interaction should set status to completed")
	}
}

func TestExpandTempleStory(t *testing.T) {
	spec := templeStorySpec()

	world, err := Expand(spec)
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	// Should have 3 rooms
	if len(world.Rooms) != 3 {
		t.Fatalf("expected 3 rooms, got %d", len(world.Rooms))
	}

	// Should have all items
	expectedItems := []string{"stone_tablet", "flint", "vine_rope", "glyph_panel", "sun_disc", "mirror_pedestal", "golden_mirror", "sun_crown"}
	for _, id := range expectedItems {
		if world.Items[id] == nil {
			t.Fatalf("missing item %q", id)
		}
	}

	// Should have 2 puzzles (win_condition doesn't create one)
	if len(world.Puzzles) != 2 {
		t.Fatalf("expected 2 puzzles, got %d", len(world.Puzzles))
	}

	// Entrance north should be locked (gate puzzle)
	if _, ok := world.Rooms["entrance"].Connections["north"]; ok {
		t.Fatal("entrance.north should be locked initially")
	}

	// Sun chamber should have south→entrance
	if world.Rooms["sun_chamber"].Connections["south"] != "entrance" {
		t.Fatal("sun_chamber should have south→entrance")
	}

	// Validate the world
	if err := world.ValidateWithStartRoom("entrance"); err != nil {
		t.Fatalf("expanded world fails validation: %v", err)
	}
}

func TestExpandCombinationLock(t *testing.T) {
	spec := &StorySpec{
		StartRoom: "vault",
		Rooms: map[string]RoomSpec{
			"vault":  {Name: "Vault", Description: "A vault.", Items: []string{"dial"}, DescriptionAfterPuzzle: "The safe is open."},
			"secret": {Name: "Secret Room", Description: "A hidden room."},
		},
		Items: map[string]ItemSpec{
			"dial": {Name: "combination dial", Aliases: []string{"dial"}, Portable: false},
		},
		Puzzles: []PuzzleSpec{
			{
				ID: "safe_puzzle", Type: "combination_lock", Name: "Crack the Safe",
				Description: "Turn the dial the right number of times.",
				Room: "vault", CombinationTarget: "dial",
				CombinationSteps: 3,
				CombinationTexts: []string{"Click. First tumbler.", "Click. Second tumbler.", "The safe swings open!"},
				UnlockDirection: "east", UnlockRoom: "secret",
				CompletionText: "The safe swings open!",
			},
		},
	}

	world, err := Expand(spec)
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	// Puzzle should exist
	puzzle := world.Puzzles["safe_puzzle"]
	if puzzle == nil {
		t.Fatal("puzzle not created")
	}
	if len(puzzle.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(puzzle.Steps))
	}

	// East should be locked
	if _, ok := world.Rooms["vault"].Connections["east"]; ok {
		t.Fatal("east should be locked initially")
	}

	// Dial should have interactions: solved guard + step 0 + step 1 + final step = 4
	dial := world.Items["dial"]
	if len(dial.Interactions) != 4 {
		t.Fatalf("expected 4 interactions, got %d", len(dial.Interactions))
	}

	// First interaction is the solved guard
	if dial.Interactions[0].Conditions[0].Key != "safe_puzzle_solved" {
		t.Fatal("first interaction should be solved guard")
	}

	// Step 0 uses negated _started
	step0 := dial.Interactions[1]
	if !step0.Conditions[0].Negate {
		t.Fatal("step 0 should negate _started condition")
	}
	if step0.Response != "Click. First tumbler." {
		t.Fatalf("wrong step 0 response: %q", step0.Response)
	}

	// Final step sets solved
	final := dial.Interactions[3]
	hasSolved := false
	for _, eff := range final.Effects {
		if eff.Key == "safe_puzzle_solved" && eff.Value == true {
			hasSolved = true
		}
	}
	if !hasSolved {
		t.Fatal("final step should set _solved")
	}

	// Conditional description
	if len(world.Rooms["vault"].ConditionalDescriptions) == 0 {
		t.Fatal("vault should have conditional description")
	}
}

func TestExpandCombinationLockSingleStep(t *testing.T) {
	spec := &StorySpec{
		StartRoom: "room",
		Rooms: map[string]RoomSpec{
			"room": {Name: "Room", Description: "A room.", Items: []string{"button"}},
		},
		Items: map[string]ItemSpec{
			"button": {Name: "button", Portable: false},
		},
		Puzzles: []PuzzleSpec{
			{
				ID: "press_puzzle", Type: "combination_lock", Name: "Press Button",
				Room: "room", CombinationTarget: "button", CombinationVerb: "press",
				CombinationSteps: 1, CompletionText: "Done!",
			},
		},
	}

	world, err := Expand(spec)
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	btn := world.Items["button"]
	// Solved guard + single step = 2 interactions
	if len(btn.Interactions) != 2 {
		t.Fatalf("expected 2 interactions for single-step combo, got %d", len(btn.Interactions))
	}

	// Single step should set both started and solved
	step := btn.Interactions[1]
	hasStarted, hasSolved := false, false
	for _, eff := range step.Effects {
		if eff.Key == "press_puzzle_started" {
			hasStarted = true
		}
		if eff.Key == "press_puzzle_solved" {
			hasSolved = true
		}
	}
	if !hasStarted || !hasSolved {
		t.Fatal("single step should set both _started and _solved")
	}
}

func TestExpandItemCombine(t *testing.T) {
	spec := &StorySpec{
		StartRoom: "workshop",
		Rooms: map[string]RoomSpec{
			"workshop": {Name: "Workshop", Description: "A workshop.", Items: []string{"rope", "hook"}},
		},
		Items: map[string]ItemSpec{
			"rope":            {Name: "rope", Aliases: []string{"rope"}, Portable: true},
			"hook":            {Name: "iron hook", Aliases: []string{"hook"}, Portable: true},
			"grappling_hook":  {Name: "grappling hook", Aliases: []string{"grapple"}, Portable: true},
		},
		Puzzles: []PuzzleSpec{
			{
				ID: "craft_grapple", Type: "item_combine", Name: "Craft Grappling Hook",
				Description: "Combine the rope and hook.",
				Room: "workshop",
				CombineItemA: "rope", CombineItemB: "hook", CombineResult: "grappling_hook",
				CombineConsumeA: true, CombineConsumeB: true,
				CombineText: "You tie the rope to the hook, creating a grappling hook!",
				CombineFailText: "You need something to combine this with.",
			},
		},
	}

	world, err := Expand(spec)
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	// Result item should exist but not be in any room
	if world.Items["grappling_hook"] == nil {
		t.Fatal("result item should exist")
	}
	for _, room := range world.Rooms {
		for _, itemID := range room.Items {
			if itemID == "grappling_hook" {
				t.Fatal("result item should not be in any room")
			}
		}
	}

	// Rope should have success + fail interactions
	rope := world.Items["rope"]
	if len(rope.Interactions) != 2 {
		t.Fatalf("expected 2 interactions on rope, got %d", len(rope.Interactions))
	}

	// Success: requires both items
	success := rope.Interactions[0]
	if len(success.Conditions) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(success.Conditions))
	}
	if success.Conditions[0].Type != "has_item" || success.Conditions[1].Type != "has_item" {
		t.Fatal("both conditions should be has_item")
	}

	// Should add result and remove both ingredients
	hasAdd, hasRemoveA, hasRemoveB := false, false, false
	for _, eff := range success.Effects {
		if eff.Type == "add_item" && eff.Key == "grappling_hook" {
			hasAdd = true
		}
		if eff.Type == "remove_item" && eff.Key == "rope" {
			hasRemoveA = true
		}
		if eff.Type == "remove_item" && eff.Key == "hook" {
			hasRemoveB = true
		}
	}
	if !hasAdd || !hasRemoveA || !hasRemoveB {
		t.Fatal("should add result and remove both ingredients")
	}

	// Fail interaction
	fail := rope.Interactions[1]
	if fail.FailResponse != "You need something to combine this with." {
		t.Fatalf("wrong fail response: %q", fail.FailResponse)
	}

	// Puzzle should exist
	puzzle := world.Puzzles["craft_grapple"]
	if puzzle == nil {
		t.Fatal("puzzle not created")
	}
}

func TestExpandCounterPuzzle(t *testing.T) {
	spec := &StorySpec{
		StartRoom: "cave",
		Rooms: map[string]RoomSpec{
			"cave":   {Name: "Crystal Cave", Description: "A cave.", Items: []string{"red_gem", "blue_gem", "green_gem"}, DescriptionAfterPuzzle: "The altar glows."},
			"shrine": {Name: "Shrine", Description: "A shrine."},
		},
		Items: map[string]ItemSpec{
			"red_gem":   {Name: "red gem", Aliases: []string{"red"}, Portable: true},
			"blue_gem":  {Name: "blue gem", Aliases: []string{"blue"}, Portable: true},
			"green_gem": {Name: "green gem", Aliases: []string{"green"}, Portable: true},
		},
		Puzzles: []PuzzleSpec{
			{
				ID: "gem_collect", Type: "counter_puzzle", Name: "Collect the Gems",
				Description:    "Use all three gems.",
				Room:           "cave",
				CounterItems:   []string{"red_gem", "blue_gem", "green_gem"},
				CounterVerb:    "use",
				CounterTarget:  3,
				CounterItemTexts: map[string]string{
					"red_gem":  "The red gem glows!",
					"blue_gem": "The blue gem shimmers!",
				},
				CounterDefaultText:  "The gem activates!",
				CounterConsumeItems: true,
				UnlockDirection:     "north",
				UnlockRoom:          "shrine",
				CompletionText:      "All gems placed! The shrine opens!",
			},
		},
	}

	world, err := Expand(spec)
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	// Each gem should have 2 interactions (guard + use)
	for _, gemID := range []string{"red_gem", "blue_gem", "green_gem"} {
		gem := world.Items[gemID]
		if len(gem.Interactions) != 2 {
			t.Fatalf("%s: expected 2 interactions, got %d", gemID, len(gem.Interactions))
		}

		// Guard should check done=true
		guard := gem.Interactions[0]
		if guard.Conditions[1].Key != "gem_collect_"+gemID+"_done" {
			t.Fatalf("%s: guard should check done var, got %q", gemID, guard.Conditions[1].Key)
		}

		// Success should increment counter
		success := gem.Interactions[1]
		hasIncrement := false
		hasRemove := false
		for _, eff := range success.Effects {
			if eff.Type == "increment_var" && eff.Key == "gem_collect_count" {
				hasIncrement = true
			}
			if eff.Type == "remove_item" && eff.Key == gemID {
				hasRemove = true
			}
		}
		if !hasIncrement {
			t.Fatalf("%s: should increment counter", gemID)
		}
		if !hasRemove {
			t.Fatalf("%s: should remove item (consume)", gemID)
		}
	}

	// Custom response texts
	red := world.Items["red_gem"]
	if red.Interactions[1].Response != "The red gem glows!" {
		t.Fatalf("red_gem should have custom text, got %q", red.Interactions[1].Response)
	}
	green := world.Items["green_gem"]
	if green.Interactions[1].Response != "The gem activates!" {
		t.Fatalf("green_gem should use default text, got %q", green.Interactions[1].Response)
	}

	// Puzzle step should use var_gte
	puzzle := world.Puzzles["gem_collect"]
	if puzzle == nil {
		t.Fatal("puzzle not created")
	}
	if puzzle.Steps[0].Conditions[0].Type != "var_gte" {
		t.Fatalf("puzzle step should use var_gte, got %q", puzzle.Steps[0].Conditions[0].Type)
	}
	if puzzle.Steps[0].Conditions[0].Value != 3 {
		t.Fatalf("puzzle step target should be 3, got %v", puzzle.Steps[0].Conditions[0].Value)
	}

	// North should be locked
	if _, ok := world.Rooms["cave"].Connections["north"]; ok {
		t.Fatal("north should be locked initially")
	}

	// Conditional description
	if len(world.Rooms["cave"].ConditionalDescriptions) == 0 {
		t.Fatal("cave should have conditional description")
	}
}

// templeStorySpec returns the existing Temple of the Sun as a StorySpec.
func templeStorySpec() *StorySpec {
	return &StorySpec{
		Title:       "Temple of the Sun",
		Slug:        "temple-of-the-sun",
		Description: "An adventure in a lost Inca temple.",
		Author:      "Temple Adventure Team",
		StartRoom:   "entrance",
		Rooms: map[string]RoomSpec{
			"entrance": {
				Name: "Temple Entrance",
				Description: "You stand before the gaping maw of an ancient Inca temple carved into the\n" +
					"mountainside. Weathered stone steps lead up to a massive doorway flanked by\n" +
					"two serpent statues.\n\n" +
					"A heavy stone door blocks the passage north. Strange glyphs are carved\n" +
					"around its frame.",
				Connections:            map[string]string{},
				Items:                  []string{"stone_tablet", "flint", "vine_rope", "glyph_panel"},
				DescriptionAfterPuzzle: "The massive stone door stands open, revealing a dark passage north.",
			},
			"sun_chamber": {
				Name: "The Sun Chamber",
				Description: "A vast circular chamber stretches before you. A golden sun disc is mounted\n" +
					"on the ceiling. A pedestal stands in the center with an empty indent.",
				Connections:            map[string]string{"south": "entrance"},
				Items:                  []string{"sun_disc", "mirror_pedestal", "golden_mirror"},
				DescriptionAfterPuzzle: "The beam strikes the golden mirror, revealing a staircase down.",
			},
			"treasure_vault": {
				Name: "The Treasure Vault",
				Description: "Golden artifacts line the walls. At the center, atop a stone altar,\n" +
					"rests the legendary Sun Crown of the Inca.",
				Connections: map[string]string{},
				Items:       []string{"sun_crown"},
			},
		},
		Items: map[string]ItemSpec{
			"stone_tablet": {
				Name: "stone tablet", Aliases: []string{"tablet"}, Portable: true,
				Description: "A weathered stone tablet covered in Inca glyphs.",
			},
			"flint": {
				Name: "piece of flint", Aliases: []string{"flint"}, Portable: true,
				Description: "A sharp piece of flint.",
			},
			"vine_rope": {
				Name: "vine rope", Aliases: []string{"rope", "vine"}, Portable: true,
				Description: "A sturdy length of vine.",
			},
			"glyph_panel": {
				Name: "glyph panel", Aliases: []string{"panel"}, Portable: false,
				Description: "A panel of three rotating stone discs beside the door.",
			},
			"sun_disc": {
				Name: "sun disc", Aliases: []string{"disc"}, Portable: false,
				Description: "A large golden disc on the ceiling depicting Inti.",
			},
			"mirror_pedestal": {
				Name: "stone pedestal", Aliases: []string{"pedestal", "altar"}, Portable: false,
				Description: "A stone pedestal with a circular indent.",
			},
			"golden_mirror": {
				Name: "golden mirror", Aliases: []string{"mirror"}, Portable: true,
				Description: "An ornate mirror with a polished golden surface.",
			},
			"sun_crown": {
				Name: "Sun Crown of the Inca", Aliases: []string{"crown", "sun crown"}, Portable: true,
				Description: "A magnificent golden headdress studded with emeralds and rubies.",
			},
		},
		Puzzles: []PuzzleSpec{
			{
				ID: "gate_puzzle", Type: "examine_learn", Name: "The Serpent Gate",
				Description:       "Decipher the glyphs and open the stone door.",
				Room:              "entrance",
				SourceItem:        "stone_tablet",
				SourceLearnText:   "The glyphs tell a story... sequence: serpent, sun, serpent.",
				TargetItem:        "glyph_panel",
				TargetVerb:        "turn",
				TargetSuccessText: "You rotate each disc to match: serpent, sun, serpent.",
				TargetFailText:    "You turn the discs randomly but nothing happens.",
				UnlockDirection:   "north",
				UnlockRoom:        "sun_chamber",
				LockVerb:          "use",
				CompletionText:    "The massive stone door grinds open!",
				LockFailText:      "The panel doesn't respond.",
			},
			{
				ID: "light_puzzle", Type: "timed_challenge", Name: "The Sun's Gaze",
				Description:    "Redirect the beam of sunlight to reveal the vault.",
				Room:           "sun_chamber",
				TriggerItem:    "sun_disc",
				TriggerVerb:    "turn",
				TriggerText:    "A brilliant beam of light shoots down and strikes the pedestal.",
				TurnLimit:      8,
				FailureText:    "The chamber collapses! You barely escape back to the entrance.",
				FailureEffects: []FailureEffectSpec{{Type: "move_player", Room: "entrance"}, {Type: "lock_connection", Room: "entrance", Direction: "north"}},
				FetchItem:       "golden_mirror",
				FetchVerb:       "use",
				FetchRoom:       "sun_chamber",
				FetchSuccessText: "You place the golden mirror on the pedestal. Light reflects brilliantly!",
				FetchConsumeItem: true,
				UnlockDirection:  "down",
				UnlockRoom:       "treasure_vault",
				CompletionText:   "A staircase descends into darkness below.",
			},
			{
				ID: "win_crown", Type: "win_condition", Name: "Claim the Crown",
				Room: "treasure_vault", WinItem: "sun_crown",
				WinText: "You lift the Sun Crown. Congratulations, adventurer!",
			},
		},
	}
}

func TestExpandNpcs(t *testing.T) {
	spec := &StorySpec{
		StartRoom: "hall",
		Rooms: map[string]RoomSpec{
			"hall": {Name: "Hall", Description: "A grand hall.", Items: []string{"crown"}},
		},
		Items: map[string]ItemSpec{
			"crown": {Name: "Crown", Portable: true},
		},
		Puzzles: []PuzzleSpec{
			{ID: "win", Type: "win_condition", Name: "Win", Room: "hall", WinItem: "crown", WinText: "You win!"},
		},
		Npcs: map[string]NpcSpec{
			"merchant": {
				Name:        "Merchant",
				Description: "A traveling merchant.",
				Aliases:     []string{"trader"},
				Room:        "hall",
				Greeting:    "Hello there!",
				Topics: map[string]string{
					"prices": "Everything is cheap!",
					"rumors": "I've heard strange things...",
				},
			},
		},
	}

	world, err := Expand(spec)
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	npc := world.Npcs["merchant"]
	if npc == nil {
		t.Fatal("NPC merchant not created")
	}
	if npc.Name != "Merchant" {
		t.Errorf("expected name Merchant, got %s", npc.Name)
	}
	if npc.Room != "hall" {
		t.Errorf("expected room hall, got %s", npc.Room)
	}
	if len(npc.Aliases) != 1 || npc.Aliases[0] != "trader" {
		t.Errorf("unexpected aliases: %v", npc.Aliases)
	}

	// Should have greeting + 2 topics = 3 dialogue lines
	if len(npc.Dialogue) != 3 {
		t.Fatalf("expected 3 dialogue lines, got %d", len(npc.Dialogue))
	}

	// Check greeting
	hasGreeting := false
	for _, dl := range npc.Dialogue {
		if dl.Topic == "" && dl.Response == "Hello there!" {
			hasGreeting = true
		}
	}
	if !hasGreeting {
		t.Error("expected greeting dialogue line")
	}

	// Check topics
	hasPrices := false
	hasRumors := false
	for _, dl := range npc.Dialogue {
		if dl.Topic == "prices" {
			hasPrices = true
		}
		if dl.Topic == "rumors" {
			hasRumors = true
		}
	}
	if !hasPrices {
		t.Error("expected prices topic")
	}
	if !hasRumors {
		t.Error("expected rumors topic")
	}
}

// --- Multiple endings tests ---

func TestExpandWinConditionWithEndings(t *testing.T) {
	spec := &StorySpec{
		Title: "Test", Slug: "test", StartRoom: "room1",
		Rooms: map[string]RoomSpec{
			"room1": {Name: "Room 1", Description: "Start.", Items: []string{"gem"}},
		},
		Items: map[string]ItemSpec{
			"gem": {Name: "gem", Portable: true},
		},
		Puzzles: []PuzzleSpec{
			{
				ID: "win", Type: "win_condition", Room: "room1", WinItem: "gem",
				Endings: []EndingSpec{
					{ID: "good", Title: "The Good Ending", Conditions: map[string]string{"helped_npc": "true"}, Text: "You helped everyone!"},
					{ID: "neutral", Title: "The Neutral Ending", Text: "You got the gem."},
				},
			},
		},
	}

	world, err := Expand(spec)
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	gem := world.Items["gem"]
	if gem == nil {
		t.Fatal("gem not found")
	}

	// Should have 2 interactions: good ending (with condition) and neutral (fallback)
	if len(gem.Interactions) < 2 {
		t.Fatalf("expected at least 2 interactions on gem, got %d", len(gem.Interactions))
	}

	// First interaction should have conditions (the good ending)
	goodInteraction := gem.Interactions[0]
	if len(goodInteraction.Conditions) == 0 {
		t.Error("good ending interaction should have conditions")
	}
	// Check it has set_ending_id effect
	hasEndingID := false
	for _, eff := range goodInteraction.Effects {
		if eff.Type == "set_ending_id" && eff.Value == "good" {
			hasEndingID = true
		}
	}
	if !hasEndingID {
		t.Error("good ending should have set_ending_id effect with value 'good'")
	}

	// Second interaction should be the fallback (no conditions)
	neutralInteraction := gem.Interactions[1]
	if len(neutralInteraction.Conditions) != 0 {
		t.Error("neutral ending (fallback) should have no conditions")
	}
	hasNeutralTitle := false
	for _, eff := range neutralInteraction.Effects {
		if eff.Type == "set_ending_title" && eff.Value == "The Neutral Ending" {
			hasNeutralTitle = true
		}
	}
	if !hasNeutralTitle {
		t.Error("neutral ending should have set_ending_title effect")
	}
}

func TestExpandWinConditionWithEndingID(t *testing.T) {
	spec := &StorySpec{
		Title: "Test", Slug: "test", StartRoom: "room1",
		Rooms: map[string]RoomSpec{
			"room1": {Name: "Room 1", Description: "Start.", Items: []string{"gem"}},
		},
		Items: map[string]ItemSpec{
			"gem": {Name: "gem", Portable: true},
		},
		Puzzles: []PuzzleSpec{
			{
				ID: "win", Type: "win_condition", Room: "room1", WinItem: "gem",
				WinText: "You got the gem!", EndingID: "gem_ending", EndingTitle: "The Gem Collector",
			},
		},
	}

	world, err := Expand(spec)
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	gem := world.Items["gem"]
	if len(gem.Interactions) != 1 {
		t.Fatalf("expected 1 interaction, got %d", len(gem.Interactions))
	}

	inter := gem.Interactions[0]
	hasEndingID := false
	hasEndingTitle := false
	for _, eff := range inter.Effects {
		if eff.Type == "set_ending_id" && eff.Value == "gem_ending" {
			hasEndingID = true
		}
		if eff.Type == "set_ending_title" && eff.Value == "The Gem Collector" {
			hasEndingTitle = true
		}
	}
	if !hasEndingID {
		t.Error("expected set_ending_id effect")
	}
	if !hasEndingTitle {
		t.Error("expected set_ending_title effect")
	}
}

func TestExpandWinConditionBackwardCompat(t *testing.T) {
	// Existing WinText-only format should still work
	spec := &StorySpec{
		Title: "Test", Slug: "test", StartRoom: "room1",
		Rooms: map[string]RoomSpec{
			"room1": {Name: "Room 1", Description: "Start.", Items: []string{"gem"}},
		},
		Items: map[string]ItemSpec{
			"gem": {Name: "gem", Portable: true},
		},
		Puzzles: []PuzzleSpec{
			{ID: "win", Type: "win_condition", Room: "room1", WinItem: "gem", WinText: "You win!"},
		},
	}

	world, err := Expand(spec)
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	gem := world.Items["gem"]
	if len(gem.Interactions) != 1 {
		t.Fatalf("expected 1 interaction, got %d", len(gem.Interactions))
	}

	inter := gem.Interactions[0]
	if inter.Response != "You win!" {
		t.Errorf("expected response 'You win!', got %q", inter.Response)
	}
	// Should NOT have ending effects
	for _, eff := range inter.Effects {
		if eff.Type == "set_ending_id" || eff.Type == "set_ending_title" {
			t.Errorf("backward-compatible win should not have ending effects, got %s", eff.Type)
		}
	}
}

func TestExpandNpcsEmpty(t *testing.T) {
	spec := &StorySpec{
		StartRoom: "hall",
		Rooms: map[string]RoomSpec{
			"hall": {Name: "Hall", Description: "A hall.", Items: []string{"crown"}},
		},
		Items: map[string]ItemSpec{
			"crown": {Name: "Crown", Portable: true},
		},
		Puzzles: []PuzzleSpec{
			{ID: "win", Type: "win_condition", Name: "Win", Room: "hall", WinItem: "crown", WinText: "You win!"},
		},
	}

	world, err := Expand(spec)
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	if len(world.Npcs) != 0 {
		t.Errorf("expected 0 NPCs, got %d", len(world.Npcs))
	}
}
