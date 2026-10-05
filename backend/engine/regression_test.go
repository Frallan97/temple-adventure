package engine

import "testing"

func newBareState() *WorldState {
	return &WorldState{
		CurrentRoom: "room_a",
		Status:      "active",
		Inventory:   map[string]bool{},
		Variables:   map[string]Variable{},
		RoomStates: map[string]*RoomState{
			"room_a": {
				AddedItems:         map[string]bool{},
				RemovedItems:       map[string]bool{},
				BlockedConnections: map[string]bool{},
				AddedConnections:   map[string]string{},
			},
		},
		NpcStates: map[string]*NpcState{},
	}
}

// A typo'd condition type must never open a gate, negated or not.
func TestUnknownConditionTypeIsAlwaysFalse(t *testing.T) {
	state := newBareState()

	if EvaluateCondition(state, Condition{Type: "has_itm", Key: "sword"}) {
		t.Error("unknown condition type should be false")
	}
	if EvaluateCondition(state, Condition{Type: "has_itm", Key: "sword", Negate: true}) {
		t.Error("negated unknown condition type must stay false, not invert into an open gate")
	}
	// A correctly spelled negated condition still inverts.
	if !EvaluateCondition(state, Condition{Type: "has_item", Key: "sword", Negate: true}) {
		t.Error("negate on a known type should still work")
	}
}

// Numeric comparisons must work when the threshold or the variable is a string.
func TestNumericComparisonsCoerceStrings(t *testing.T) {
	state := newBareState()
	state.Variables["hp"] = Variable{Type: "int", IntVal: 5}

	if EvaluateCondition(state, Condition{Type: "var_gte", Key: "hp", Value: "10"}) {
		t.Error(`hp=5 >= "10" must be false; a string threshold used to coerce to 0`)
	}
	if !EvaluateCondition(state, Condition{Type: "var_gte", Key: "hp", Value: "3"}) {
		t.Error(`hp=5 >= "3" should be true`)
	}

	// Dialogue set_var stores everything as a string.
	state.Variables["score"] = Variable{Type: "string", StrVal: "100"}
	if !EvaluateCondition(state, Condition{Type: "var_gte", Key: "score", Value: 50}) {
		t.Error(`string-typed score="100" >= 50 should be true`)
	}
	if !EvaluateCondition(state, Condition{Type: "var_lte", Key: "score", Value: 100}) {
		t.Error(`string-typed score="100" <= 100 should be true`)
	}

	// Non-numeric strings degrade to 0 rather than blowing up.
	state.Variables["name"] = Variable{Type: "string", StrVal: "gandalf"}
	if !EvaluateCondition(state, Condition{Type: "var_gte", Key: "name", Value: 0}) {
		t.Error("non-numeric string should compare as 0")
	}
}

// add_room_item should default to the current room, like remove_room_item.
func TestAddRoomItemDefaultsToCurrentRoom(t *testing.T) {
	state := newBareState()

	ApplyEffect(state, Effect{Type: "add_room_item", Key: "gem"})
	if !IsItemInRoom(state, "gem", "room_a") {
		t.Error("add_room_item with no Value should place the item in the current room")
	}

	// An explicit room still wins.
	state.RoomStates["room_b"] = &RoomState{
		AddedItems: map[string]bool{}, RemovedItems: map[string]bool{},
		BlockedConnections: map[string]bool{}, AddedConnections: map[string]string{},
	}
	ApplyEffect(state, Effect{Type: "add_room_item", Key: "coin", Value: "room_b"})
	if !IsItemInRoom(state, "coin", "room_b") {
		t.Error("explicit room should still be honoured")
	}
	if IsItemInRoom(state, "coin", "room_a") {
		t.Error("explicit room must not also place the item in the current room")
	}
}

// Natural phrasing should resolve to the same target as the bare form.
func TestParserStripsFillerWords(t *testing.T) {
	p := NewCommandParser()

	cases := []struct{ input, verb, target string }{
		{"take the relic", "take", "relic"},
		{"take relic", "take", "relic"},
		{"talk to keeper", "talk", "keeper"},
		{"look at altar", "look", "altar"},
		{"use the key on the door", "use", "key on the door"},
		{"examine the tome", "look", "tome"},
		{"move north", "move", "north"},
		{"go to the library", "move", "library"},
		{"n", "move", "north"},
		{"ask keeper about relic", "ask", "keeper about relic"},
	}

	for _, c := range cases {
		got := p.Parse(c.input)
		if got.Verb != c.verb || got.Target != c.target {
			t.Errorf("Parse(%q) = verb %q target %q; want verb %q target %q",
				c.input, got.Verb, got.Target, c.verb, c.target)
		}
	}
}

// Stripping must never reduce a target to nothing.
func TestParserKeepsTargetWhenAllFiller(t *testing.T) {
	p := NewCommandParser()
	got := p.Parse("take the")
	if got.Target == "" {
		t.Error("an all-filler target should be left alone, not emptied")
	}
}
