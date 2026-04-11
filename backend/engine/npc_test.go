package engine

import (
	"strings"
	"testing"
)

func npcTestWorld() *WorldDefinition {
	return &WorldDefinition{
		Rooms: map[string]*RoomDef{
			"tavern": {
				ID: "tavern", Name: "The Tavern",
				Description: "A cozy tavern.",
				Connections: map[string]string{"north": "market"},
				Items:       []string{"mug"},
			},
			"market": {
				ID: "market", Name: "The Market",
				Description: "A bustling market square.",
				Connections: map[string]string{"south": "tavern"},
			},
		},
		Items: map[string]*ItemDef{
			"mug": {
				ID: "mug", Name: "Mug", Description: "A wooden mug.", Portable: true,
			},
		},
		Puzzles: map[string]*PuzzleDef{},
		Npcs: map[string]*NpcDef{
			"merchant": {
				ID: "merchant", Name: "Merchant", Description: "A friendly merchant with a wide smile.",
				Aliases: []string{"trader", "shopkeeper"},
				Room:    "tavern",
				Dialogue: []DialogueLine{
					{Topic: "", Response: "Welcome, traveler! What brings you here?"},
					{Topic: "prices", Response: "My prices are fair, I assure you."},
					{
						Topic:      "secret",
						Conditions: []Condition{{Type: "has_item", Key: "mug"}},
						Response:   "Since you bought a drink... the treasure is hidden in the market.",
						Effects:    []Effect{{Type: "set_var", Key: "knows_secret", Value: true}},
					},
					{
						Topic:    "secret",
						Response: "I don't talk to strangers without a drink.",
					},
				},
			},
			"guard": {
				ID: "guard", Name: "Guard", Description: "A stern-looking guard.",
				Room: "market",
				Movement: []NpcMovement{
					{
						Conditions: []Condition{{Type: "var_equals", Key: "alarm", Value: true}},
						TargetRoom: "tavern",
					},
				},
			},
		},
	}
}

func TestNpcTalk(t *testing.T) {
	eng := NewEngineFromWorld(npcTestWorld())
	state := eng.World.NewWorldState("test", "tavern")

	result := eng.ProcessCommand(state, "talk merchant")
	if !strings.Contains(result.Text, "Welcome, traveler") {
		t.Errorf("Expected greeting, got: %s", result.Text)
	}
}

func TestNpcTalkAlias(t *testing.T) {
	eng := NewEngineFromWorld(npcTestWorld())
	state := eng.World.NewWorldState("test", "tavern")

	result := eng.ProcessCommand(state, "talk trader")
	if !strings.Contains(result.Text, "Welcome, traveler") {
		t.Errorf("Expected greeting via alias, got: %s", result.Text)
	}
}

func TestNpcTalkNotPresent(t *testing.T) {
	eng := NewEngineFromWorld(npcTestWorld())
	state := eng.World.NewWorldState("test", "tavern")

	result := eng.ProcessCommand(state, "talk guard")
	if !strings.Contains(result.Text, "don't see") {
		t.Errorf("Should not see guard in tavern, got: %s", result.Text)
	}
}

func TestNpcAskTopic(t *testing.T) {
	eng := NewEngineFromWorld(npcTestWorld())
	state := eng.World.NewWorldState("test", "tavern")

	result := eng.ProcessCommand(state, "ask merchant about prices")
	if !strings.Contains(result.Text, "fair") {
		t.Errorf("Expected prices response, got: %s", result.Text)
	}
}

func TestNpcAskConditionalTopic(t *testing.T) {
	eng := NewEngineFromWorld(npcTestWorld())
	state := eng.World.NewWorldState("test", "tavern")

	// Without mug, should get fallback
	result := eng.ProcessCommand(state, "ask merchant about secret")
	if !strings.Contains(result.Text, "without a drink") {
		t.Errorf("Without mug should get refusal, got: %s", result.Text)
	}

	// Take mug then ask again
	eng.ProcessCommand(state, "take mug")
	result = eng.ProcessCommand(state, "ask merchant about secret")
	if !strings.Contains(result.Text, "treasure is hidden") {
		t.Errorf("With mug should get secret, got: %s", result.Text)
	}

	// Check effect was applied
	v, ok := state.Variables["knows_secret"]
	if !ok || !v.BoolVal {
		t.Error("knows_secret variable should be set to true")
	}
}

func TestNpcAskUnknownTopic(t *testing.T) {
	eng := NewEngineFromWorld(npcTestWorld())
	state := eng.World.NewWorldState("test", "tavern")

	result := eng.ProcessCommand(state, "ask merchant about weather")
	if !strings.Contains(result.Text, "doesn't have anything to say") {
		t.Errorf("Unknown topic should get default response, got: %s", result.Text)
	}
}

func TestNpcInRoomDescription(t *testing.T) {
	eng := NewEngineFromWorld(npcTestWorld())
	state := eng.World.NewWorldState("test", "tavern")

	result := eng.ProcessCommand(state, "look")
	if !strings.Contains(result.Text, "Present: Merchant") {
		t.Errorf("Room description should show NPC, got: %s", result.Text)
	}
}

func TestNpcExamine(t *testing.T) {
	eng := NewEngineFromWorld(npcTestWorld())
	state := eng.World.NewWorldState("test", "tavern")

	result := eng.ProcessCommand(state, "look merchant")
	if !strings.Contains(result.Text, "friendly merchant") {
		t.Errorf("Examine NPC should show description, got: %s", result.Text)
	}
}

func TestNpcInRoomCondition(t *testing.T) {
	eng := NewEngineFromWorld(npcTestWorld())
	state := eng.World.NewWorldState("test", "tavern")

	// merchant is in tavern
	cond := Condition{Type: "npc_in_room", Key: "merchant"}
	if !EvaluateCondition(state, cond) {
		t.Error("npc_in_room should be true for merchant in tavern")
	}

	// guard is in market, not tavern
	cond2 := Condition{Type: "npc_in_room", Key: "guard"}
	if EvaluateCondition(state, cond2) {
		t.Error("npc_in_room should be false for guard in tavern")
	}

	// Check specific room
	cond3 := Condition{Type: "npc_in_room", Key: "guard", Value: "market"}
	if !EvaluateCondition(state, cond3) {
		t.Error("npc_in_room with explicit room should be true for guard in market")
	}
}

func TestMoveNpcEffect(t *testing.T) {
	eng := NewEngineFromWorld(npcTestWorld())
	state := eng.World.NewWorldState("test", "tavern")

	// Guard starts in market
	if state.NpcStates["guard"].CurrentRoom != "market" {
		t.Fatalf("Guard should start in market, got %s", state.NpcStates["guard"].CurrentRoom)
	}

	// Apply move_npc effect
	ApplyEffect(state, Effect{Type: "move_npc", Key: "guard", Value: "tavern"})

	if state.NpcStates["guard"].CurrentRoom != "tavern" {
		t.Errorf("Guard should now be in tavern, got %s", state.NpcStates["guard"].CurrentRoom)
	}

	// Guard should now appear in tavern room description
	desc := describeRoom(state, eng.World)
	if !strings.Contains(desc, "Guard") {
		t.Errorf("Guard should appear in tavern description after move, got: %s", desc)
	}
}

func TestNpcMovementRules(t *testing.T) {
	eng := NewEngineFromWorld(npcTestWorld())
	state := eng.World.NewWorldState("test", "tavern")

	// Guard starts in market
	if state.NpcStates["guard"].CurrentRoom != "market" {
		t.Fatal("Guard should start in market")
	}

	// Set alarm variable — guard's movement rule should trigger
	state.Variables["alarm"] = Variable{Type: "bool", BoolVal: true}

	// Process any command to trigger NPC movement evaluation
	eng.ProcessCommand(state, "look")

	if state.NpcStates["guard"].CurrentRoom != "tavern" {
		t.Errorf("Guard should have moved to tavern after alarm, got %s", state.NpcStates["guard"].CurrentRoom)
	}
}

func TestSpeakAlias(t *testing.T) {
	eng := NewEngineFromWorld(npcTestWorld())
	state := eng.World.NewWorldState("test", "tavern")

	result := eng.ProcessCommand(state, "speak merchant")
	if !strings.Contains(result.Text, "Welcome, traveler") {
		t.Errorf("speak alias should work like talk, got: %s", result.Text)
	}
}

func TestTalkNoTarget(t *testing.T) {
	eng := NewEngineFromWorld(npcTestWorld())
	state := eng.World.NewWorldState("test", "tavern")

	result := eng.ProcessCommand(state, "talk")
	if !strings.Contains(result.Text, "Talk to whom") {
		t.Errorf("talk with no target should prompt, got: %s", result.Text)
	}
}

func TestAskNoAbout(t *testing.T) {
	eng := NewEngineFromWorld(npcTestWorld())
	state := eng.World.NewWorldState("test", "tavern")

	result := eng.ProcessCommand(state, "ask merchant")
	if !strings.Contains(result.Text, "about") {
		t.Errorf("ask without about should show usage, got: %s", result.Text)
	}
}
