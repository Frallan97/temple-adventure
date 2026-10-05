package storygen

import (
	"strings"
	"testing"

	"temple-adventure/engine"
)

// A spec with a conditional entry node listed AFTER the default one.
func reTalkSpec() *StorySpec {
	return &StorySpec{
		Title: "ReTalk", Slug: "retalk", StartRoom: "camp",
		Rooms: map[string]RoomSpec{
			"camp": {Name: "Camp", Description: "A camp.", Items: []string{"relic"}},
		},
		Items: map[string]ItemSpec{
			"relic": {Name: "Relic", Description: "A relic.", Portable: true},
		},
		Npcs: map[string]NpcSpec{
			"sage": {
				Name: "Sage", Description: "A sage.", Room: "camp",
				Dialogue: []DialogueNodeSpec{
					{
						NodeID: "greet_first",
						Text:   "FIRST: We have not met.",
						Choices: []DialogueChoiceSpec{
							{Text: "Promise to help", NextNode: "", SetVar: "promised=yes"},
						},
					},
					{
						NodeID:     "greet_again",
						Text:       "SECOND: You kept your word.",
						Conditions: map[string]string{"promised": "yes"},
						Choices: []DialogueChoiceSpec{
							{Text: "Ask for the reward", NextNode: "reward", NeedVar: "promised=yes"},
							{Text: "Ask the impossible", NextNode: "reward", NeedVar: "promised=never"},
						},
					},
					{NodeID: "reward", Text: "DEEP: Here is your reward."},
				},
			},
		},
		Puzzles: []PuzzleSpec{{
			ID: "win", Type: "win_condition", Name: "Win", Room: "camp",
			WinItem: "relic", WinVerb: "take", WinText: "done",
		}},
	}
}

// Conditional entry nodes must be emitted before the unconditional default,
// and interior nodes last, whatever order the author used.
func TestConditionalEntryNodeHoistedAboveDefault(t *testing.T) {
	world, err := Expand(reTalkSpec())
	if err != nil {
		t.Fatalf("expand: %v", err)
	}

	var order []string
	for _, dl := range world.Npcs["sage"].Dialogue {
		order = append(order, dl.NodeID)
	}
	want := []string{"greet_again", "greet_first", "reward"}
	if strings.Join(order, ",") != strings.Join(want, ",") {
		t.Errorf("dialogue order = %v, want %v", order, want)
	}

	if len(world.Npcs["sage"].Dialogue[0].Conditions) != 1 {
		t.Error("conditional entry node should carry its condition")
	}
}

// need_var must become a var_equals condition on the choice.
func TestNeedVarBecomesChoiceCondition(t *testing.T) {
	world, err := Expand(reTalkSpec())
	if err != nil {
		t.Fatalf("expand: %v", err)
	}

	var node *engine.DialogueLine
	for i := range world.Npcs["sage"].Dialogue {
		if world.Npcs["sage"].Dialogue[i].NodeID == "greet_again" {
			node = &world.Npcs["sage"].Dialogue[i]
		}
	}
	if node == nil {
		t.Fatal("greet_again missing")
	}
	if len(node.Choices) != 2 {
		t.Fatalf("expected 2 choices, got %d", len(node.Choices))
	}
	c := node.Choices[0].Conditions
	if len(c) != 1 || c[0].Type != "var_equals" || c[0].Key != "promised" || c[0].Value != "yes" {
		t.Errorf("need_var should produce var_equals promised=yes, got %+v", c)
	}
}

// The whole point: talking a second time gives a different line, and choices
// gated on need_var are filtered.
func TestSecondConversationDiffersEndToEnd(t *testing.T) {
	world, err := Expand(reTalkSpec())
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	if errs := ValidateWorldDeep(world, "camp"); len(errs) > 0 {
		t.Fatalf("deep validation: %v", errs)
	}

	eng := engine.NewEngineFromWorld(world)
	state := &engine.WorldState{
		CurrentRoom: "camp", Status: "active",
		Inventory: map[string]bool{},
		Variables: map[string]engine.Variable{},
		RoomStates: map[string]*engine.RoomState{"camp": {
			AddedItems: map[string]bool{}, RemovedItems: map[string]bool{},
			BlockedConnections: map[string]bool{}, AddedConnections: map[string]string{},
		}},
		NpcStates: map[string]*engine.NpcState{"sage": {CurrentRoom: "camp"}},
	}

	first := eng.ProcessCommand(state, "talk sage")
	if !strings.Contains(first.Text, "FIRST") {
		t.Fatalf("first talk should use the default entry, got %q", first.Text)
	}

	eng.ProcessCommand(state, "say 1") // promise, ends conversation

	second := eng.ProcessCommand(state, "talk sage")
	if !strings.Contains(second.Text, "SECOND") {
		t.Fatalf("second talk should use the conditional entry, got %q", second.Text)
	}
	// Only the satisfiable choice should be offered.
	if len(second.Choices) != 1 {
		t.Errorf("expected 1 visible choice (need_var filtered), got %d: %+v",
			len(second.Choices), second.Choices)
	}

	deep := eng.ProcessCommand(state, "say 1")
	if !strings.Contains(deep.Text, "DEEP") {
		t.Errorf("should descend to the reward node, got %q", deep.Text)
	}
}

func TestValidateSpecRejectsTwoUnconditionalEntryNodes(t *testing.T) {
	spec := reTalkSpec()
	npc := spec.Npcs["sage"]
	npc.Dialogue[1].Conditions = nil // greet_again becomes a second bare entry
	spec.Npcs["sage"] = npc

	errs := ValidateSpec(spec)
	if !hasErrContaining(errs, "entry nodes have no conditions") {
		t.Errorf("expected an error about multiple unconditional entry nodes, got %v", errs)
	}
}

func TestValidateSpecRequiresUnconditionalEntryNode(t *testing.T) {
	spec := reTalkSpec()
	npc := spec.Npcs["sage"]
	npc.Dialogue[0].Conditions = map[string]string{"never": "true"}
	spec.Npcs["sage"] = npc

	errs := ValidateSpec(spec)
	if !hasErrContaining(errs, "every entry node is conditional") {
		t.Errorf("expected an error about no fallback entry node, got %v", errs)
	}
}

func TestValidateSpecRejectsMalformedNeedVar(t *testing.T) {
	spec := reTalkSpec()
	npc := spec.Npcs["sage"]
	npc.Dialogue[1].Choices[0].NeedVar = "promised"
	spec.Npcs["sage"] = npc

	errs := ValidateSpec(spec)
	if !hasErrContaining(errs, "need_var") {
		t.Errorf("expected an error about need_var form, got %v", errs)
	}
}

func hasErrContaining(errs []string, substr string) bool {
	for _, e := range errs {
		if strings.Contains(e, substr) {
			return true
		}
	}
	return false
}
