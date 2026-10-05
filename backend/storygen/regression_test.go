package storygen

import (
	"strings"
	"testing"

	"temple-adventure/engine"
)

func endingProbeSpec(endings []EndingSpec) *StorySpec {
	return &StorySpec{
		Title: "Probe", Slug: "probe", StartRoom: "shrine",
		Rooms: map[string]RoomSpec{
			"shrine": {Name: "Shrine", Description: "A shrine.", Items: []string{"relic"}},
		},
		Items: map[string]ItemSpec{
			"relic": {Name: "Relic", Description: "A relic.", Portable: true},
		},
		Puzzles: []PuzzleSpec{{
			ID: "win", Type: "win_condition", Name: "Win", Room: "shrine",
			WinItem: "relic", WinVerb: "take", Endings: endings,
		}},
	}
}

// The unconditional fallback must be expanded last regardless of spec order,
// because the engine takes the first interaction whose conditions pass.
func TestFallbackEndingIsAlwaysExpandedLast(t *testing.T) {
	endings := []EndingSpec{
		{ID: "fallback", Title: "Fallback", Text: "fallback"},
		{ID: "noble", Title: "Noble", Conditions: map[string]string{"path": "noble"}, Text: "noble"},
		{ID: "greedy", Title: "Greedy", Conditions: map[string]string{"path": "greedy"}, Text: "greedy"},
	}

	world, err := Expand(endingProbeSpec(endings))
	if err != nil {
		t.Fatalf("expand: %v", err)
	}

	var order []string
	for _, inter := range world.Items["relic"].Interactions {
		if inter.Verb != "take" {
			continue
		}
		for _, e := range inter.Effects {
			if e.Type == "set_ending_id" {
				order = append(order, e.Value.(string))
			}
		}
	}

	if len(order) != 3 {
		t.Fatalf("expected 3 ending interactions, got %d (%v)", len(order), order)
	}
	if order[len(order)-1] != "fallback" {
		t.Errorf("fallback must be last, got order %v", order)
	}
	// Conditional endings keep their relative author order.
	if order[0] != "noble" || order[1] != "greedy" {
		t.Errorf("conditional endings should keep author order, got %v", order)
	}
}

// Expansion must be deterministic even though EndingSpec.Conditions is a map.
func TestEndingConditionsAreDeterministic(t *testing.T) {
	endings := []EndingSpec{
		{ID: "multi", Title: "Multi", Text: "multi", Conditions: map[string]string{
			"zeta": "1", "alpha": "2", "mid": "3",
		}},
		{ID: "fallback", Title: "Fallback", Text: "fallback"},
	}

	var first []string
	for run := 0; run < 8; run++ {
		world, err := Expand(endingProbeSpec(endings))
		if err != nil {
			t.Fatalf("expand: %v", err)
		}
		var keys []string
		for _, inter := range world.Items["relic"].Interactions {
			if len(inter.Conditions) > 0 {
				for _, c := range inter.Conditions {
					keys = append(keys, c.Key)
				}
				break
			}
		}
		if run == 0 {
			first = keys
			continue
		}
		if strings.Join(keys, ",") != strings.Join(first, ",") {
			t.Fatalf("condition order varies between expansions: %v vs %v", first, keys)
		}
	}
	if strings.Join(first, ",") != "alpha,mid,zeta" {
		t.Errorf("conditions should be sorted, got %v", first)
	}
}

// Multiple unconditional endings mean only the first is reachable.
func TestValidateSpecRejectsMultipleFallbacks(t *testing.T) {
	spec := endingProbeSpec([]EndingSpec{
		{ID: "a", Title: "A", Text: "a"},
		{ID: "b", Title: "B", Text: "b"},
	})

	errs := ValidateSpec(spec)
	found := false
	for _, e := range errs {
		if strings.Contains(e, "fallback endings") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected an error about multiple fallback endings, got %v", errs)
	}
}

// key_lock must not print its completion text twice.
func TestKeyLockCompletionTextNotDuplicated(t *testing.T) {
	spec := &StorySpec{
		Title: "T", Slug: "t", StartRoom: "gate",
		Rooms: map[string]RoomSpec{
			"gate": {Name: "Gate", Description: "A gate.", Items: []string{"key", "door"}},
			"hall": {Name: "Hall", Description: "A hall.", Items: []string{"prize"}},
		},
		Items: map[string]ItemSpec{
			"key":   {Name: "Key", Description: "A key.", Portable: true},
			"door":  {Name: "Door", Description: "A door."},
			"prize": {Name: "Prize", Description: "A prize.", Portable: true},
		},
		Puzzles: []PuzzleSpec{
			{
				ID: "unlock", Type: "key_lock", Name: "Unlock", Room: "gate",
				KeyItem: "key", LockTarget: "door", UnlockDirection: "north",
				UnlockRoom: "hall", CompletionText: "The door opens!",
			},
			{ID: "win", Type: "win_condition", Name: "Win", Room: "hall",
				WinItem: "prize", WinVerb: "take", WinText: "You win."},
		},
	}

	world, err := Expand(spec)
	if err != nil {
		t.Fatalf("expand: %v", err)
	}

	// The interaction carries the text; the PuzzleDef must not repeat it.
	var interactionText string
	for _, inter := range world.Items["door"].Interactions {
		if len(inter.Effects) > 0 {
			interactionText = inter.Response
		}
	}
	if interactionText != "The door opens!" {
		t.Errorf("unlock interaction should respond with the completion text, got %q", interactionText)
	}
	if got := world.Puzzles["unlock"].CompletionText; got != "" {
		t.Errorf("PuzzleDef.CompletionText should be empty to avoid double-printing, got %q", got)
	}
}

// Every puzzle should contribute a room hint, so `hint` is never dead.
func TestPuzzlesGenerateRoomHints(t *testing.T) {
	world, err := Expand(endingProbeSpec([]EndingSpec{
		{ID: "only", Title: "Only", Text: "only"},
	}))
	if err != nil {
		t.Fatalf("expand: %v", err)
	}

	hints := world.Rooms["shrine"].Hints
	if len(hints) == 0 {
		t.Fatal("expected at least one generated hint in the puzzle's room")
	}
	h := hints[0]
	if h.Text == "" {
		t.Error("generated hint should have text")
	}
	if h.Condition == nil || !h.Condition.Negate {
		t.Error("hint should be gated on the puzzle being unsolved")
	}
	// win_condition has no PuzzleDef, so it must gate on game_won.
	if h.Condition.Key != "game_won" {
		t.Errorf("win_condition hint should gate on game_won, got %q", h.Condition.Key)
	}

	var _ engine.ConditionalHint = h
}
