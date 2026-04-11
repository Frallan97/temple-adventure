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
