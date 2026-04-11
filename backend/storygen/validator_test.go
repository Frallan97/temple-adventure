package storygen

import (
	"strings"
	"testing"

	"temple-adventure/engine"
)

// --- ValidateSpec tests ---

func TestValidateSpecValid(t *testing.T) {
	spec := templeStorySpec()
	errs := ValidateSpec(spec)
	if len(errs) > 0 {
		t.Fatalf("valid spec should have no errors, got: %v", errs)
	}
}

func TestValidateSpecMissingTitle(t *testing.T) {
	spec := minimalSpec()
	spec.Title = ""
	errs := ValidateSpec(spec)
	assertContains(t, errs, "title is required")
}

func TestValidateSpecMissingSlug(t *testing.T) {
	spec := minimalSpec()
	spec.Slug = ""
	errs := ValidateSpec(spec)
	assertContains(t, errs, "slug is required")
}

func TestValidateSpecMissingStartRoom(t *testing.T) {
	spec := minimalSpec()
	spec.StartRoom = "nonexistent"
	errs := ValidateSpec(spec)
	assertContains(t, errs, "start_room")
}

func TestValidateSpecBadItemRef(t *testing.T) {
	spec := minimalSpec()
	spec.Rooms["room1"] = RoomSpec{
		Name: "Room", Description: "A room.", Items: []string{"ghost_item"},
	}
	errs := ValidateSpec(spec)
	assertContains(t, errs, "unknown item")
}

func TestValidateSpecBadConnectionRef(t *testing.T) {
	spec := minimalSpec()
	spec.Rooms["room1"] = RoomSpec{
		Name: "Room", Description: "A room.",
		Connections: map[string]string{"north": "missing_room"},
	}
	errs := ValidateSpec(spec)
	assertContains(t, errs, "unknown room")
}

func TestValidateSpecNoWinCondition(t *testing.T) {
	spec := minimalSpec()
	spec.Puzzles = []PuzzleSpec{} // remove win condition
	errs := ValidateSpec(spec)
	assertContains(t, errs, "win_condition")
}

func TestValidateSpecDuplicatePuzzleID(t *testing.T) {
	spec := minimalSpec()
	spec.Puzzles = append(spec.Puzzles, PuzzleSpec{
		ID: "win", Type: "win_condition", WinItem: "crown",
	})
	errs := ValidateSpec(spec)
	assertContains(t, errs, "duplicate puzzle id")
}

func TestValidateSpecInvalidPuzzleType(t *testing.T) {
	spec := minimalSpec()
	spec.Puzzles = append(spec.Puzzles, PuzzleSpec{
		ID: "bad", Type: "teleport", Room: "room1",
	})
	errs := ValidateSpec(spec)
	assertContains(t, errs, "invalid type")
}

func TestValidateSpecKeyLockMissingFields(t *testing.T) {
	spec := minimalSpec()
	spec.Puzzles = append(spec.Puzzles, PuzzleSpec{
		ID: "lock1", Type: "key_lock", Room: "room1",
		// Missing key_item, lock_target, unlock_room, unlock_direction
	})
	errs := ValidateSpec(spec)
	assertContains(t, errs, "key_item is required")
	assertContains(t, errs, "lock_target is required")
	assertContains(t, errs, "unlock_room is required")
	assertContains(t, errs, "unlock_direction is required")
}

func TestValidateSpecTimedBadTurnLimit(t *testing.T) {
	spec := minimalSpec()
	spec.Puzzles = append(spec.Puzzles, PuzzleSpec{
		ID: "timed1", Type: "timed_challenge", Room: "room1",
		TriggerItem: "crown", TurnLimit: 0,
	})
	errs := ValidateSpec(spec)
	assertContains(t, errs, "turn_limit must be > 0")
}

func TestValidateSpecBadFailureEffectType(t *testing.T) {
	spec := minimalSpec()
	spec.Puzzles = append(spec.Puzzles, PuzzleSpec{
		ID: "timed2", Type: "timed_challenge", Room: "room1",
		TriggerItem: "crown", TurnLimit: 5,
		FailureEffects: []FailureEffectSpec{{Type: "explode"}},
	})
	errs := ValidateSpec(spec)
	assertContains(t, errs, "invalid failure effect type")
}

// --- ValidateWorldDeep tests ---

func TestValidateWorldDeepValid(t *testing.T) {
	spec := templeStorySpec()
	world, err := Expand(spec)
	if err != nil {
		t.Fatal(err)
	}
	errs := ValidateWorldDeep(world, "entrance")
	if len(errs) > 0 {
		t.Fatalf("valid world should pass deep validation, got: %v", errs)
	}
}

func TestValidateWorldDeepMissingStartRoom(t *testing.T) {
	world := &engine.WorldDefinition{
		Rooms: map[string]*engine.RoomDef{
			"hall": {ID: "hall", Name: "Hall"},
		},
		Items:   map[string]*engine.ItemDef{},
		Puzzles: map[string]*engine.PuzzleDef{},
	}
	errs := ValidateWorldDeep(world, "nonexistent")
	assertContains(t, errs, "start room")
}

func TestValidateWorldDeepUncheckedVar(t *testing.T) {
	world := &engine.WorldDefinition{
		Rooms: map[string]*engine.RoomDef{
			"hall": {ID: "hall", Name: "Hall", Connections: map[string]string{}},
		},
		Items: map[string]*engine.ItemDef{
			"lever": {
				ID: "lever", Name: "lever",
				Interactions: []engine.Interaction{
					{
						Verb: "pull",
						Conditions: []engine.Condition{
							{Type: "var_equals", Key: "mystery_var", Value: true},
						},
					},
				},
			},
		},
		Puzzles: map[string]*engine.PuzzleDef{},
	}
	errs := ValidateWorldDeep(world, "hall")
	assertContains(t, errs, "mystery_var")
	assertContains(t, errs, "never set")
}

func TestValidateWorldDeepBadItemInEffect(t *testing.T) {
	world := &engine.WorldDefinition{
		Rooms: map[string]*engine.RoomDef{
			"hall": {ID: "hall", Name: "Hall", Connections: map[string]string{}},
		},
		Items: map[string]*engine.ItemDef{
			"wand": {
				ID: "wand", Name: "wand",
				Interactions: []engine.Interaction{
					{
						Verb: "use",
						Effects: []engine.Effect{
							{Type: "add_item", Key: "phantom_item"},
						},
					},
				},
			},
		},
		Puzzles: map[string]*engine.PuzzleDef{},
	}
	errs := ValidateWorldDeep(world, "hall")
	assertContains(t, errs, "phantom_item")
	assertContains(t, errs, "unknown item")
}

func TestValidateWorldDeepPuzzleNoSteps(t *testing.T) {
	world := &engine.WorldDefinition{
		Rooms: map[string]*engine.RoomDef{
			"hall": {ID: "hall", Name: "Hall", Connections: map[string]string{}},
		},
		Items:   map[string]*engine.ItemDef{},
		Puzzles: map[string]*engine.PuzzleDef{"empty": {ID: "empty", Name: "Empty"}},
	}
	errs := ValidateWorldDeep(world, "hall")
	assertContains(t, errs, "no steps")
}

func TestValidateWorldDeepBadUnlockTarget(t *testing.T) {
	world := &engine.WorldDefinition{
		Rooms: map[string]*engine.RoomDef{
			"hall": {ID: "hall", Name: "Hall", Connections: map[string]string{}},
		},
		Items: map[string]*engine.ItemDef{
			"key": {
				ID: "key", Name: "key",
				Interactions: []engine.Interaction{
					{
						Verb: "use",
						Effects: []engine.Effect{
							{Type: "unlock_connection", Key: "hall.north", Value: "ghost_room"},
						},
					},
				},
			},
		},
		Puzzles: map[string]*engine.PuzzleDef{},
	}
	errs := ValidateWorldDeep(world, "hall")
	assertContains(t, errs, "ghost_room")
}

func TestValidateWorldDeepNoWinCondition(t *testing.T) {
	world := &engine.WorldDefinition{
		Rooms: map[string]*engine.RoomDef{
			"hall": {ID: "hall", Name: "Hall", Connections: map[string]string{}},
		},
		Items:   map[string]*engine.ItemDef{},
		Puzzles: map[string]*engine.PuzzleDef{},
	}
	errs := ValidateWorldDeep(world, "hall")
	assertContains(t, errs, "no win condition")
}

// --- combination_lock ---

func TestValidateSpecCombinationLockValid(t *testing.T) {
	spec := minimalSpec()
	spec.Items["dial"] = ItemSpec{Name: "dial", Portable: false}
	spec.Rooms["room1"] = RoomSpec{Name: "Room 1", Items: []string{"crown", "dial"}}
	spec.Puzzles = append(spec.Puzzles, PuzzleSpec{
		ID: "combo", Type: "combination_lock", Room: "room1",
		CombinationTarget: "dial", CombinationSteps: 3,
	})
	errs := ValidateSpec(spec)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got: %v", errs)
	}
}

func TestValidateSpecCombinationLockMissingTarget(t *testing.T) {
	spec := minimalSpec()
	spec.Puzzles = append(spec.Puzzles, PuzzleSpec{
		ID: "combo", Type: "combination_lock", Room: "room1",
		CombinationSteps: 3,
	})
	errs := ValidateSpec(spec)
	assertContains(t, errs, "combination_target is required")
}

func TestValidateSpecCombinationLockBadSteps(t *testing.T) {
	spec := minimalSpec()
	spec.Items["dial"] = ItemSpec{Name: "dial", Portable: false}
	spec.Rooms["room1"] = RoomSpec{Name: "Room 1", Items: []string{"crown", "dial"}}
	spec.Puzzles = append(spec.Puzzles, PuzzleSpec{
		ID: "combo", Type: "combination_lock", Room: "room1",
		CombinationTarget: "dial", CombinationSteps: 0,
	})
	errs := ValidateSpec(spec)
	assertContains(t, errs, "combination_steps must be > 0")
}

func TestValidateSpecCombinationLockTextsMismatch(t *testing.T) {
	spec := minimalSpec()
	spec.Items["dial"] = ItemSpec{Name: "dial", Portable: false}
	spec.Rooms["room1"] = RoomSpec{Name: "Room 1", Items: []string{"crown", "dial"}}
	spec.Puzzles = append(spec.Puzzles, PuzzleSpec{
		ID: "combo", Type: "combination_lock", Room: "room1",
		CombinationTarget: "dial", CombinationSteps: 3,
		CombinationTexts: []string{"one", "two"},
	})
	errs := ValidateSpec(spec)
	assertContains(t, errs, "combination_texts length")
}

// --- item_combine ---

func TestValidateSpecItemCombineValid(t *testing.T) {
	spec := minimalSpec()
	spec.Items["rope"] = ItemSpec{Name: "rope", Portable: true}
	spec.Items["hook"] = ItemSpec{Name: "hook", Portable: true}
	spec.Items["grapple"] = ItemSpec{Name: "grappling hook", Portable: true}
	spec.Rooms["room1"] = RoomSpec{Name: "Room 1", Items: []string{"crown", "rope", "hook"}}
	spec.Puzzles = append(spec.Puzzles, PuzzleSpec{
		ID: "craft", Type: "item_combine", Room: "room1",
		CombineItemA: "rope", CombineItemB: "hook", CombineResult: "grapple",
	})
	errs := ValidateSpec(spec)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got: %v", errs)
	}
}

func TestValidateSpecItemCombineMissingItems(t *testing.T) {
	spec := minimalSpec()
	spec.Puzzles = append(spec.Puzzles, PuzzleSpec{
		ID: "craft", Type: "item_combine", Room: "room1",
	})
	errs := ValidateSpec(spec)
	assertContains(t, errs, "combine_item_a is required")
	assertContains(t, errs, "combine_item_b is required")
	assertContains(t, errs, "combine_result is required")
}

func TestValidateSpecItemCombineResultNotPortable(t *testing.T) {
	spec := minimalSpec()
	spec.Items["rope"] = ItemSpec{Name: "rope", Portable: true}
	spec.Items["hook"] = ItemSpec{Name: "hook", Portable: true}
	spec.Items["statue"] = ItemSpec{Name: "statue", Portable: false}
	spec.Rooms["room1"] = RoomSpec{Name: "Room 1", Items: []string{"crown", "rope", "hook"}}
	spec.Puzzles = append(spec.Puzzles, PuzzleSpec{
		ID: "craft", Type: "item_combine", Room: "room1",
		CombineItemA: "rope", CombineItemB: "hook", CombineResult: "statue",
	})
	errs := ValidateSpec(spec)
	assertContains(t, errs, "should be portable")
}

func TestValidateSpecItemCombineResultInRoom(t *testing.T) {
	spec := minimalSpec()
	spec.Items["rope"] = ItemSpec{Name: "rope", Portable: true}
	spec.Items["hook"] = ItemSpec{Name: "hook", Portable: true}
	spec.Items["grapple"] = ItemSpec{Name: "grappling hook", Portable: true}
	spec.Rooms["room1"] = RoomSpec{Name: "Room 1", Items: []string{"crown", "rope", "hook", "grapple"}}
	spec.Puzzles = append(spec.Puzzles, PuzzleSpec{
		ID: "craft", Type: "item_combine", Room: "room1",
		CombineItemA: "rope", CombineItemB: "hook", CombineResult: "grapple",
	})
	errs := ValidateSpec(spec)
	assertContains(t, errs, "should not be placed in room")
}

// --- counter_puzzle ---

func TestValidateSpecCounterPuzzleValid(t *testing.T) {
	spec := minimalSpec()
	spec.Items["gem_a"] = ItemSpec{Name: "gem a", Portable: true}
	spec.Items["gem_b"] = ItemSpec{Name: "gem b", Portable: true}
	spec.Rooms["room1"] = RoomSpec{Name: "Room 1", Items: []string{"crown", "gem_a", "gem_b"}}
	spec.Puzzles = append(spec.Puzzles, PuzzleSpec{
		ID: "collect", Type: "counter_puzzle", Room: "room1",
		CounterItems: []string{"gem_a", "gem_b"}, CounterTarget: 2,
	})
	errs := ValidateSpec(spec)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got: %v", errs)
	}
}

func TestValidateSpecCounterPuzzleEmpty(t *testing.T) {
	spec := minimalSpec()
	spec.Puzzles = append(spec.Puzzles, PuzzleSpec{
		ID: "collect", Type: "counter_puzzle", Room: "room1",
		CounterTarget: 1,
	})
	errs := ValidateSpec(spec)
	assertContains(t, errs, "counter_items is required")
}

func TestValidateSpecCounterPuzzleBadTarget(t *testing.T) {
	spec := minimalSpec()
	spec.Items["gem"] = ItemSpec{Name: "gem", Portable: true}
	spec.Rooms["room1"] = RoomSpec{Name: "Room 1", Items: []string{"crown", "gem"}}
	spec.Puzzles = append(spec.Puzzles, PuzzleSpec{
		ID: "collect", Type: "counter_puzzle", Room: "room1",
		CounterItems: []string{"gem"}, CounterTarget: 0,
	})
	errs := ValidateSpec(spec)
	assertContains(t, errs, "counter_target must be > 0")
}

func TestValidateSpecCounterPuzzleTargetExceedsItems(t *testing.T) {
	spec := minimalSpec()
	spec.Items["gem"] = ItemSpec{Name: "gem", Portable: true}
	spec.Rooms["room1"] = RoomSpec{Name: "Room 1", Items: []string{"crown", "gem"}}
	spec.Puzzles = append(spec.Puzzles, PuzzleSpec{
		ID: "collect", Type: "counter_puzzle", Room: "room1",
		CounterItems: []string{"gem"}, CounterTarget: 5,
	})
	errs := ValidateSpec(spec)
	assertContains(t, errs, "exceeds counter_items count")
}

func TestValidateSpecCounterPuzzleNonPortable(t *testing.T) {
	spec := minimalSpec()
	spec.Items["lever"] = ItemSpec{Name: "lever", Portable: false}
	spec.Rooms["room1"] = RoomSpec{Name: "Room 1", Items: []string{"crown", "lever"}}
	spec.Puzzles = append(spec.Puzzles, PuzzleSpec{
		ID: "collect", Type: "counter_puzzle", Room: "room1",
		CounterItems: []string{"lever"}, CounterTarget: 1,
	})
	errs := ValidateSpec(spec)
	assertContains(t, errs, "must be portable")
}

// --- helpers ---

func minimalSpec() *StorySpec {
	return &StorySpec{
		Title:     "Test",
		Slug:      "test",
		StartRoom: "room1",
		Rooms: map[string]RoomSpec{
			"room1": {Name: "Room 1", Description: "A room.", Items: []string{"crown"}},
		},
		Items: map[string]ItemSpec{
			"crown": {Name: "crown", Portable: true},
		},
		Puzzles: []PuzzleSpec{
			{ID: "win", Type: "win_condition", Room: "room1", WinItem: "crown", WinText: "You win!"},
		},
	}
}

func assertContains(t *testing.T, errs []string, substr string) {
	t.Helper()
	for _, e := range errs {
		if strings.Contains(e, substr) {
			return
		}
	}
	t.Errorf("expected error containing %q, got: %v", substr, errs)
}

func TestValidateSpecNpcValid(t *testing.T) {
	spec := minimalSpec()
	spec.Npcs = map[string]NpcSpec{
		"merchant": {Name: "Merchant", Room: "room1"},
	}
	errs := ValidateSpec(spec)
	if len(errs) > 0 {
		t.Errorf("expected no errors for valid NPC, got: %v", errs)
	}
}

func TestValidateSpecNpcBadRoom(t *testing.T) {
	spec := minimalSpec()
	spec.Npcs = map[string]NpcSpec{
		"merchant": {Name: "Merchant", Room: "nonexistent"},
	}
	errs := ValidateSpec(spec)
	assertContains(t, errs, "unknown room")
}

func TestValidateSpecNpcNoName(t *testing.T) {
	spec := minimalSpec()
	spec.Npcs = map[string]NpcSpec{
		"merchant": {Room: "room1"},
	}
	errs := ValidateSpec(spec)
	assertContains(t, errs, "name is required")
}

func TestValidateSpecNpcNoRoom(t *testing.T) {
	spec := minimalSpec()
	spec.Npcs = map[string]NpcSpec{
		"merchant": {Name: "Merchant"},
	}
	errs := ValidateSpec(spec)
	assertContains(t, errs, "room is required")
}
