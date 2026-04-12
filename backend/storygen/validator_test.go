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

// --- ValidateGameplay tests ---

func assertResultContains(t *testing.T, list []string, substr string) {
	t.Helper()
	for _, e := range list {
		if strings.Contains(e, substr) {
			return
		}
	}
	t.Errorf("expected entry containing %q, got: %v", substr, list)
}

func TestGameplaySelfReferentialKeyLock(t *testing.T) {
	spec := minimalSpec()
	spec.Rooms["room2"] = RoomSpec{Name: "Room 2", Description: "Room 2.", Connections: map[string]string{"south": "room1"}}
	spec.Rooms["room1"] = RoomSpec{Name: "Room 1", Description: "Room 1.", Items: []string{"crown", "key", "door"}, Connections: map[string]string{"north": "room2"}}
	spec.Items["key"] = ItemSpec{Name: "key", Portable: true}
	spec.Items["door"] = ItemSpec{Name: "door", Portable: false}
	spec.Puzzles = append(spec.Puzzles, PuzzleSpec{
		ID: "selflock", Type: "key_lock", Room: "room1",
		KeyItem: "key", LockTarget: "door",
		UnlockDirection: "down", UnlockRoom: "room1", // self-referential!
	})
	vr := ValidateGameplay(spec)
	assertResultContains(t, vr.Errors, "same as the puzzle room")
}

func TestGameplayUnreachableRoom(t *testing.T) {
	spec := minimalSpec()
	spec.Rooms["island"] = RoomSpec{Name: "Island", Description: "No connections lead here."}
	vr := ValidateGameplay(spec)
	assertResultContains(t, vr.Errors, "not reachable")
}

func TestGameplayAllRoomsReachable(t *testing.T) {
	spec := minimalSpec()
	spec.Rooms["room1"] = RoomSpec{
		Name: "Room 1", Description: "Start.", Items: []string{"crown"},
		Connections: map[string]string{"north": "room2"},
	}
	spec.Rooms["room2"] = RoomSpec{
		Name: "Room 2", Description: "End.",
		Connections: map[string]string{"south": "room1"},
	}
	vr := ValidateGameplay(spec)
	for _, e := range vr.Errors {
		if strings.Contains(e, "not reachable") {
			t.Errorf("unexpected reachability error: %s", e)
		}
	}
}

func TestGameplayReachableViaKeyLock(t *testing.T) {
	// room1 has key, room1 has locked door to room2
	spec := &StorySpec{
		Title: "Test", Slug: "test", StartRoom: "room1",
		Rooms: map[string]RoomSpec{
			"room1": {Name: "R1", Description: "Start.", Items: []string{"crown", "key", "door"}, Connections: map[string]string{}},
			"room2": {Name: "R2", Description: "Locked room.", Items: []string{}, Connections: map[string]string{}},
		},
		Items: map[string]ItemSpec{
			"crown": {Name: "crown", Portable: true},
			"key":   {Name: "key", Portable: true},
			"door":  {Name: "door", Portable: false},
		},
		Puzzles: []PuzzleSpec{
			{ID: "win", Type: "win_condition", Room: "room1", WinItem: "crown", WinText: "You win!"},
			{ID: "lock1", Type: "key_lock", Room: "room1", KeyItem: "key", LockTarget: "door", UnlockDirection: "north", UnlockRoom: "room2"},
		},
	}
	vr := ValidateGameplay(spec)
	for _, e := range vr.Errors {
		if strings.Contains(e, "not reachable") {
			t.Errorf("room2 should be reachable via key_lock unlock, got: %s", e)
		}
	}
}

func TestGameplayKeyBehindLock(t *testing.T) {
	spec := &StorySpec{
		Title: "Test", Slug: "test", StartRoom: "room1",
		Rooms: map[string]RoomSpec{
			"room1": {Name: "R1", Description: "Start.", Items: []string{"crown", "door"}, Connections: map[string]string{}},
			"room2": {Name: "R2", Description: "Locked.", Items: []string{"key"}, Connections: map[string]string{}},
		},
		Items: map[string]ItemSpec{
			"crown": {Name: "crown", Portable: true},
			"key":   {Name: "key", Portable: true},
			"door":  {Name: "door", Portable: false},
		},
		Puzzles: []PuzzleSpec{
			{ID: "win", Type: "win_condition", Room: "room1", WinItem: "crown", WinText: "You win!"},
			{ID: "lock1", Type: "key_lock", Room: "room1", KeyItem: "key", LockTarget: "door", UnlockDirection: "north", UnlockRoom: "room2"},
		},
	}
	vr := ValidateGameplay(spec)
	assertResultContains(t, vr.Errors, "key is unreachable")
}

func TestGameplayWinItemInStartRoom(t *testing.T) {
	spec := minimalSpec()
	vr := ValidateGameplay(spec)
	assertResultContains(t, vr.Warnings, "win item")
	assertResultContains(t, vr.Warnings, "start room")
}

func TestGameplayCircularPuzzleDeps(t *testing.T) {
	// lock A's key is in room B (locked by B), lock B's key is in room A (locked by A)
	spec := &StorySpec{
		Title: "Test", Slug: "test", StartRoom: "lobby",
		Rooms: map[string]RoomSpec{
			"lobby":  {Name: "Lobby", Description: "Start.", Items: []string{"crown", "door_a", "door_b"}, Connections: map[string]string{}},
			"room_a": {Name: "Room A", Description: "A.", Items: []string{"key_b"}, Connections: map[string]string{}},
			"room_b": {Name: "Room B", Description: "B.", Items: []string{"key_a"}, Connections: map[string]string{}},
		},
		Items: map[string]ItemSpec{
			"crown":  {Name: "crown", Portable: true},
			"key_a":  {Name: "key a", Portable: true},
			"key_b":  {Name: "key b", Portable: true},
			"door_a": {Name: "door a", Portable: false},
			"door_b": {Name: "door b", Portable: false},
		},
		Puzzles: []PuzzleSpec{
			{ID: "win", Type: "win_condition", Room: "lobby", WinItem: "crown", WinText: "You win!"},
			{ID: "lock_a", Type: "key_lock", Room: "lobby", KeyItem: "key_a", LockTarget: "door_a", UnlockDirection: "north", UnlockRoom: "room_a"},
			{ID: "lock_b", Type: "key_lock", Room: "lobby", KeyItem: "key_b", LockTarget: "door_b", UnlockDirection: "east", UnlockRoom: "room_b"},
		},
	}
	vr := ValidateGameplay(spec)
	assertResultContains(t, vr.Errors, "circular puzzle dependency")
}

func TestGameplayDeadEnd(t *testing.T) {
	spec := minimalSpec()
	spec.Rooms["room1"] = RoomSpec{
		Name: "Room 1", Description: "Start.", Items: []string{"crown"},
		Connections: map[string]string{"north": "dead"},
	}
	spec.Rooms["dead"] = RoomSpec{Name: "Dead End", Description: "No way out."}
	vr := ValidateGameplay(spec)
	assertResultContains(t, vr.Warnings, "no connections")
}

func TestGameplayMissingReverseConnection(t *testing.T) {
	spec := minimalSpec()
	spec.Rooms["room1"] = RoomSpec{
		Name: "Room 1", Description: "Start.", Items: []string{"crown"},
		Connections: map[string]string{"north": "room2"},
	}
	spec.Rooms["room2"] = RoomSpec{
		Name: "Room 2", Description: "One way.",
		Connections: map[string]string{}, // no south back to room1
	}
	vr := ValidateGameplay(spec)
	assertResultContains(t, vr.Warnings, "no connection back")
}

func TestGameplayOrphanDialogueNode(t *testing.T) {
	spec := minimalSpec()
	spec.Npcs = map[string]NpcSpec{
		"bob": {
			Name: "Bob", Room: "room1",
			Dialogue: []DialogueNodeSpec{
				{NodeID: "greeting", Text: "Hello!", Choices: []DialogueChoiceSpec{
					{Text: "Bye", NextNode: "__exit__"},
				}},
				{NodeID: "orphan", Text: "Nobody can reach me."},
			},
		},
	}
	vr := ValidateGameplay(spec)
	assertResultContains(t, vr.Warnings, "orphan")
}

func TestGameplaySimpleTopicWarning(t *testing.T) {
	spec := minimalSpec()
	spec.Npcs = map[string]NpcSpec{
		"bob": {
			Name: "Bob", Room: "room1",
			Greeting: "Hello!",
			Topics:   map[string]string{"treasure": "It's hidden."},
		},
	}
	vr := ValidateGameplay(spec)
	assertResultContains(t, vr.Warnings, "simple topics")
}

// --- Ending validation tests ---

func TestValidateSpecEndingsValid(t *testing.T) {
	spec := minimalSpec()
	spec.Puzzles = []PuzzleSpec{
		{
			ID: "win", Type: "win_condition", Room: "room1", WinItem: "crown",
			Endings: []EndingSpec{
				{ID: "good", Title: "Good End", Conditions: map[string]string{"helped": "true"}, Text: "You helped!"},
				{ID: "neutral", Title: "Neutral End", Text: "You survived."},
			},
		},
	}
	errs := ValidateSpec(spec)
	if len(errs) > 0 {
		t.Fatalf("expected no errors, got: %v", errs)
	}
}

func TestValidateSpecEndingsMissingID(t *testing.T) {
	spec := minimalSpec()
	spec.Puzzles = []PuzzleSpec{
		{
			ID: "win", Type: "win_condition", Room: "room1", WinItem: "crown",
			Endings: []EndingSpec{
				{Title: "Bad", Text: "Oops."},
				{ID: "fallback", Title: "Ok", Text: "Fine."},
			},
		},
	}
	errs := ValidateSpec(spec)
	assertContains(t, errs, "missing 'id'")
}

func TestValidateSpecEndingsMissingText(t *testing.T) {
	spec := minimalSpec()
	spec.Puzzles = []PuzzleSpec{
		{
			ID: "win", Type: "win_condition", Room: "room1", WinItem: "crown",
			Endings: []EndingSpec{
				{ID: "empty", Title: "Empty"},
				{ID: "fallback", Title: "Ok", Text: "Fine."},
			},
		},
	}
	errs := ValidateSpec(spec)
	assertContains(t, errs, "missing 'text'")
}

func TestValidateSpecEndingsNoFallback(t *testing.T) {
	spec := minimalSpec()
	spec.Puzzles = []PuzzleSpec{
		{
			ID: "win", Type: "win_condition", Room: "room1", WinItem: "crown",
			Endings: []EndingSpec{
				{ID: "good", Title: "Good", Conditions: map[string]string{"x": "true"}, Text: "Good!"},
				{ID: "bad", Title: "Bad", Conditions: map[string]string{"y": "true"}, Text: "Bad!"},
			},
		},
	}
	errs := ValidateSpec(spec)
	assertContains(t, errs, "no fallback ending")
}

func TestValidateSpecEndingsMutuallyExclusive(t *testing.T) {
	spec := minimalSpec()
	spec.Puzzles = []PuzzleSpec{
		{
			ID: "win", Type: "win_condition", Room: "room1", WinItem: "crown",
			EndingID: "oops",
			Endings: []EndingSpec{
				{ID: "good", Title: "Good", Text: "Good!"},
			},
		},
	}
	errs := ValidateSpec(spec)
	assertContains(t, errs, "mutually exclusive")
}

func TestValidateSpecEndingsWinTextIgnored(t *testing.T) {
	spec := minimalSpec()
	spec.Puzzles = []PuzzleSpec{
		{
			ID: "win", Type: "win_condition", Room: "room1", WinItem: "crown",
			WinText: "should not be here",
			Endings: []EndingSpec{
				{ID: "good", Title: "Good", Text: "Good!"},
			},
		},
	}
	errs := ValidateSpec(spec)
	assertContains(t, errs, "win_text")
}

func TestValidateSpecDuplicateEndingIDs(t *testing.T) {
	spec := minimalSpec()
	spec.Items["sword"] = ItemSpec{Name: "sword", Portable: true}
	spec.Rooms["room2"] = RoomSpec{Name: "Room 2", Description: "R2.", Items: []string{"sword"}, Connections: map[string]string{"south": "room1"}}
	spec.Rooms["room1"] = RoomSpec{Name: "Room 1", Description: "R1.", Items: []string{"crown"}, Connections: map[string]string{"north": "room2"}}
	spec.Puzzles = []PuzzleSpec{
		{ID: "win1", Type: "win_condition", Room: "room1", WinItem: "crown", WinText: "Win1!", EndingID: "same_id"},
		{ID: "win2", Type: "win_condition", Room: "room2", WinItem: "sword", WinText: "Win2!", EndingID: "same_id"},
	}
	errs := ValidateSpec(spec)
	assertContains(t, errs, "already used")
}

func TestGameplayEndingUnreachableWinItem(t *testing.T) {
	spec := &StorySpec{
		Title: "Test", Slug: "test", StartRoom: "room1",
		Rooms: map[string]RoomSpec{
			"room1": {Name: "R1", Description: "Start.", Items: []string{"gem"}, Connections: map[string]string{}},
			"island": {Name: "Island", Description: "Unreachable.", Items: []string{"sword"}, Connections: map[string]string{}},
		},
		Items: map[string]ItemSpec{
			"gem":   {Name: "gem", Portable: true},
			"sword": {Name: "sword", Portable: true},
		},
		Puzzles: []PuzzleSpec{
			{ID: "win1", Type: "win_condition", Room: "room1", WinItem: "gem", WinText: "Got gem!"},
			{ID: "win2", Type: "win_condition", Room: "island", WinItem: "sword", WinText: "Got sword!"},
		},
	}
	vr := ValidateGameplay(spec)
	assertResultContains(t, vr.Errors, "win item \"sword\" is in unreachable room")
}

func TestGameplayEndingConditionUnknownVar(t *testing.T) {
	spec := &StorySpec{
		Title: "Test", Slug: "test", StartRoom: "room1",
		Rooms: map[string]RoomSpec{
			"room1": {Name: "R1", Description: "Start.", Items: []string{"gem"}},
		},
		Items: map[string]ItemSpec{
			"gem": {Name: "gem", Portable: true},
		},
		Puzzles: []PuzzleSpec{
			{
				ID: "win", Type: "win_condition", Room: "room1", WinItem: "gem",
				Endings: []EndingSpec{
					{ID: "good", Title: "Good", Conditions: map[string]string{"mystery_var": "true"}, Text: "Good!"},
					{ID: "default", Title: "Default", Text: "Ok."},
				},
			},
		},
	}
	vr := ValidateGameplay(spec)
	assertResultContains(t, vr.Warnings, "mystery_var")
	assertResultContains(t, vr.Warnings, "not set by any")
}

func TestGameplayEndingConditionSetByDialogue(t *testing.T) {
	spec := &StorySpec{
		Title: "Test", Slug: "test", StartRoom: "room1",
		Rooms: map[string]RoomSpec{
			"room1": {Name: "R1", Description: "Start.", Items: []string{"gem"}},
		},
		Items: map[string]ItemSpec{
			"gem": {Name: "gem", Portable: true},
		},
		Puzzles: []PuzzleSpec{
			{
				ID: "win", Type: "win_condition", Room: "room1", WinItem: "gem",
				Endings: []EndingSpec{
					{ID: "good", Title: "Good", Conditions: map[string]string{"helped_npc": "true"}, Text: "Good!"},
					{ID: "default", Title: "Default", Text: "Ok."},
				},
			},
		},
		Npcs: map[string]NpcSpec{
			"bob": {
				Name: "Bob", Room: "room1",
				Dialogue: []DialogueNodeSpec{
					{NodeID: "greeting", Text: "Hi!", Choices: []DialogueChoiceSpec{
						{Text: "Help me", NextNode: "__exit__", SetVar: "helped_npc=true"},
					}},
				},
			},
		},
	}
	vr := ValidateGameplay(spec)
	// Should NOT warn about helped_npc since it's set by dialogue
	for _, w := range vr.Warnings {
		if strings.Contains(w, "helped_npc") {
			t.Errorf("should not warn about helped_npc, got: %s", w)
		}
	}
}

func TestGameplayNoWarningsOnCleanSpec(t *testing.T) {
	// A well-formed 2-room spec with proper connections
	spec := &StorySpec{
		Title: "Clean", Slug: "clean", StartRoom: "room1",
		Rooms: map[string]RoomSpec{
			"room1": {Name: "Room 1", Description: "Start.", Items: []string{"key", "door"}, Connections: map[string]string{"north": "room2"}},
			"room2": {Name: "Room 2", Description: "End.", Items: []string{"crown"}, Connections: map[string]string{"south": "room1"}},
		},
		Items: map[string]ItemSpec{
			"crown": {Name: "crown", Portable: true},
			"key":   {Name: "key", Portable: true},
			"door":  {Name: "door", Portable: false},
		},
		Puzzles: []PuzzleSpec{
			{ID: "win", Type: "win_condition", Room: "room2", WinItem: "crown", WinText: "You win!"},
		},
	}
	vr := ValidateGameplay(spec)
	if len(vr.Errors) > 0 {
		t.Errorf("expected no errors, got: %v", vr.Errors)
	}
	if len(vr.Warnings) > 0 {
		t.Errorf("expected no warnings, got: %v", vr.Warnings)
	}
}
