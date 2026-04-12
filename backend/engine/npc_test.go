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

// --- Dialogue tree tests ---

func dialogueTreeWorld() *WorldDefinition {
	return &WorldDefinition{
		Rooms: map[string]*RoomDef{
			"tavern": {
				ID: "tavern", Name: "The Tavern",
				Description: "A cozy tavern.",
				Connections: map[string]string{},
			},
		},
		Items: map[string]*ItemDef{
			"coin": {ID: "coin", Name: "Gold Coin", Description: "A shiny coin.", Portable: true},
		},
		Puzzles: map[string]*PuzzleDef{},
		Npcs: map[string]*NpcDef{
			"barkeep": {
				ID: "barkeep", Name: "Barkeep", Description: "A burly bartender.",
				Aliases: []string{"bartender"},
				Room:    "tavern",
				Dialogue: []DialogueLine{
					{
						NodeID:   "greeting",
						Topic:    "",
						Response: "Welcome! What can I do for you?",
						Choices: []DialogueChoice{
							{Text: "Tell me about the specials", NextNode: "specials"},
							{Text: "I heard you know a secret", NextNode: "secret_check"},
							{Text: "Nothing, goodbye", NextNode: "__exit__"},
						},
					},
					{
						NodeID:   "specials",
						Response: "We have ale and stew today. Both excellent!",
						Choices: []DialogueChoice{
							{Text: "I'll have the ale", NextNode: "ale_choice", Effects: []Effect{{Type: "set_var", Key: "ordered_ale", Value: true}}},
							{Text: "Tell me more about the stew", NextNode: "stew_info"},
						},
					},
					{
						NodeID:   "ale_choice",
						Response: "Good choice! Here's your ale.",
						Effects:  []Effect{{Type: "set_var", Key: "has_ale", Value: true}},
						// No choices — conversation ends automatically
					},
					{
						NodeID:   "stew_info",
						Response: "The stew is made with fresh vegetables from the market.",
					},
					{
						NodeID:   "secret_check",
						Response: "A secret, you say?",
						Choices: []DialogueChoice{
							{
								Text:       "Yes, I have coin to pay for info",
								NextNode:   "secret_reveal",
								Conditions: []Condition{{Type: "has_item", Key: "coin"}},
								Effects:    []Effect{{Type: "remove_item", Key: "coin"}},
							},
							{Text: "Nevermind", NextNode: "__exit__"},
						},
					},
					{
						NodeID:   "secret_reveal",
						Response: "The treasure is buried under the old oak tree!",
						Effects:  []Effect{{Type: "set_var", Key: "knows_treasure", Value: true}},
					},
					// Flat dialogue line (backward compat — no NodeID)
					{Topic: "weather", Response: "Lovely day, isn't it?"},
				},
			},
		},
	}
}

func TestDialogueTreeTalkShowsChoices(t *testing.T) {
	eng := NewEngineFromWorld(dialogueTreeWorld())
	state := eng.World.NewWorldState("test", "tavern")

	result := eng.ProcessCommand(state, "talk barkeep")
	if !strings.Contains(result.Text, "Welcome") {
		t.Errorf("Expected greeting text, got: %s", result.Text)
	}
	if len(result.Choices) != 3 {
		t.Fatalf("Expected 3 choices, got %d", len(result.Choices))
	}
	if result.Choices[0].Text != "Tell me about the specials" {
		t.Errorf("Choice 1 wrong: %s", result.Choices[0].Text)
	}
	if result.Choices[0].Index != 1 {
		t.Errorf("Choice 1 index should be 1, got %d", result.Choices[0].Index)
	}
}

func TestDialogueTreeSayAdvancesToNextNode(t *testing.T) {
	eng := NewEngineFromWorld(dialogueTreeWorld())
	state := eng.World.NewWorldState("test", "tavern")

	// Start conversation
	eng.ProcessCommand(state, "talk barkeep")

	// Choose "Tell me about the specials"
	result := eng.ProcessCommand(state, "say 1")
	if !strings.Contains(result.Text, "ale and stew") {
		t.Errorf("Expected specials response, got: %s", result.Text)
	}
	if len(result.Choices) != 2 {
		t.Fatalf("Expected 2 choices at specials node, got %d", len(result.Choices))
	}
}

func TestDialogueTreeAutoExitOnNoChoices(t *testing.T) {
	eng := NewEngineFromWorld(dialogueTreeWorld())
	state := eng.World.NewWorldState("test", "tavern")

	// Start conversation → choose specials → choose ale (no choices = auto-exit)
	eng.ProcessCommand(state, "talk barkeep")
	eng.ProcessCommand(state, "say 1") // specials
	result := eng.ProcessCommand(state, "say 1") // ale_choice

	if !strings.Contains(result.Text, "Good choice") {
		t.Errorf("Expected ale response, got: %s", result.Text)
	}
	if len(result.Choices) != 0 {
		t.Errorf("Should have no choices (auto-exit), got %d", len(result.Choices))
	}

	// Verify conversation ended — say should fail
	result = eng.ProcessCommand(state, "say 1")
	if !strings.Contains(result.Text, "not in a conversation") {
		t.Errorf("Should not be in a conversation after auto-exit, got: %s", result.Text)
	}
}

func TestDialogueTreeExitNode(t *testing.T) {
	eng := NewEngineFromWorld(dialogueTreeWorld())
	state := eng.World.NewWorldState("test", "tavern")

	eng.ProcessCommand(state, "talk barkeep")
	// Choose "Nothing, goodbye" → __exit__
	result := eng.ProcessCommand(state, "say 3")

	if !strings.Contains(result.Text, "end your conversation") {
		t.Errorf("Expected exit message, got: %s", result.Text)
	}

	// Verify conversation cleared
	result = eng.ProcessCommand(state, "say 1")
	if !strings.Contains(result.Text, "not in a conversation") {
		t.Errorf("Should not be in conversation after exit, got: %s", result.Text)
	}
}

func TestDialogueTreeConditionalChoices(t *testing.T) {
	eng := NewEngineFromWorld(dialogueTreeWorld())
	state := eng.World.NewWorldState("test", "tavern")

	// Start conversation → choose "secret"
	eng.ProcessCommand(state, "talk barkeep")
	result := eng.ProcessCommand(state, "say 2") // secret_check

	// Without coin, the "pay for info" choice should be hidden
	if len(result.Choices) != 1 {
		t.Fatalf("Without coin, should only see 1 choice (Nevermind), got %d", len(result.Choices))
	}
	if result.Choices[0].Text != "Nevermind" {
		t.Errorf("Only visible choice should be Nevermind, got: %s", result.Choices[0].Text)
	}
}

func TestDialogueTreeConditionalChoicesWithItem(t *testing.T) {
	eng := NewEngineFromWorld(dialogueTreeWorld())
	state := eng.World.NewWorldState("test", "tavern")

	// Give player the coin
	state.Inventory["coin"] = true

	eng.ProcessCommand(state, "talk barkeep")
	result := eng.ProcessCommand(state, "say 2") // secret_check

	// With coin, should see both choices
	if len(result.Choices) != 2 {
		t.Fatalf("With coin, should see 2 choices, got %d", len(result.Choices))
	}

	// Choose "pay for info" — should remove coin and reveal secret
	result = eng.ProcessCommand(state, "say 1")
	if !strings.Contains(result.Text, "treasure is buried") {
		t.Errorf("Expected secret reveal, got: %s", result.Text)
	}

	// Coin should be removed (choice effect)
	if state.Inventory["coin"] {
		t.Error("Coin should have been removed by choice effect")
	}

	// knows_treasure should be set (node effect)
	v, ok := state.Variables["knows_treasure"]
	if !ok || !v.BoolVal {
		t.Error("knows_treasure should be set to true")
	}
}

func TestDialogueTreeEffectsOnChoices(t *testing.T) {
	eng := NewEngineFromWorld(dialogueTreeWorld())
	state := eng.World.NewWorldState("test", "tavern")

	eng.ProcessCommand(state, "talk barkeep")
	eng.ProcessCommand(state, "say 1") // specials
	eng.ProcessCommand(state, "say 1") // ale (has effect: ordered_ale=true)

	v, ok := state.Variables["ordered_ale"]
	if !ok || !v.BoolVal {
		t.Error("ordered_ale should be set by choice effect")
	}

	v2, ok := state.Variables["has_ale"]
	if !ok || !v2.BoolVal {
		t.Error("has_ale should be set by node effect")
	}
}

func TestDialogueTreeSayNotInConversation(t *testing.T) {
	eng := NewEngineFromWorld(dialogueTreeWorld())
	state := eng.World.NewWorldState("test", "tavern")

	result := eng.ProcessCommand(state, "say 1")
	if !strings.Contains(result.Text, "not in a conversation") {
		t.Errorf("Say without conversation should error, got: %s", result.Text)
	}
}

func TestDialogueTreeBareNumber(t *testing.T) {
	eng := NewEngineFromWorld(dialogueTreeWorld())
	state := eng.World.NewWorldState("test", "tavern")

	eng.ProcessCommand(state, "talk barkeep")
	// Bare "1" should work as "say 1"
	result := eng.ProcessCommand(state, "1")
	if !strings.Contains(result.Text, "ale and stew") {
		t.Errorf("Bare number should work as say, got: %s", result.Text)
	}
}

func TestDialogueTreeFlatDialogueStillWorks(t *testing.T) {
	eng := NewEngineFromWorld(dialogueTreeWorld())
	state := eng.World.NewWorldState("test", "tavern")

	// Flat topic-based dialogue (no NodeID) should work as before
	result := eng.ProcessCommand(state, "ask barkeep about weather")
	if !strings.Contains(result.Text, "Lovely day") {
		t.Errorf("Flat dialogue should still work, got: %s", result.Text)
	}
	if len(result.Choices) != 0 {
		t.Errorf("Flat dialogue should have no choices, got %d", len(result.Choices))
	}
}

func TestDialogueTreeTalkResumesConversation(t *testing.T) {
	eng := NewEngineFromWorld(dialogueTreeWorld())
	state := eng.World.NewWorldState("test", "tavern")

	// Start conversation
	eng.ProcessCommand(state, "talk barkeep")
	// Navigate to specials
	eng.ProcessCommand(state, "say 1")

	// Talk again should resume at specials node (not restart)
	result := eng.ProcessCommand(state, "talk barkeep")
	if !strings.Contains(result.Text, "ale and stew") {
		t.Errorf("Talk mid-conversation should resume, got: %s", result.Text)
	}
}

func TestDialogueTreeVisitedVariable(t *testing.T) {
	eng := NewEngineFromWorld(dialogueTreeWorld())
	state := eng.World.NewWorldState("test", "tavern")

	eng.ProcessCommand(state, "talk barkeep")

	// greeting node should be marked visited
	v, ok := state.Variables["dlg.barkeep.visited.greeting"]
	if !ok || !v.BoolVal {
		t.Error("greeting node should be marked as visited")
	}

	// Navigate to specials
	eng.ProcessCommand(state, "say 1")
	v2, ok := state.Variables["dlg.barkeep.visited.specials"]
	if !ok || !v2.BoolVal {
		t.Error("specials node should be marked as visited")
	}
}

func TestDialogueTreeInvalidChoice(t *testing.T) {
	eng := NewEngineFromWorld(dialogueTreeWorld())
	state := eng.World.NewWorldState("test", "tavern")

	eng.ProcessCommand(state, "talk barkeep")
	result := eng.ProcessCommand(state, "say 99")
	if !strings.Contains(result.Text, "Invalid choice") {
		t.Errorf("Should reject invalid choice number, got: %s", result.Text)
	}
}

func TestDialogueTreeAskWithChoices(t *testing.T) {
	// Test that ask about a topic with a branching node enters the tree
	world := dialogueTreeWorld()
	world.Npcs["barkeep"].Dialogue = append(world.Npcs["barkeep"].Dialogue, DialogueLine{
		NodeID:   "rumors_root",
		Topic:    "rumors",
		Response: "I've heard many rumors...",
		Choices: []DialogueChoice{
			{Text: "About the treasure?", NextNode: "rumors_treasure"},
			{Text: "About the king?", NextNode: "__exit__"},
		},
	})
	world.Npcs["barkeep"].Dialogue = append(world.Npcs["barkeep"].Dialogue, DialogueLine{
		NodeID:   "rumors_treasure",
		Response: "They say it's guarded by a dragon!",
	})

	eng := NewEngineFromWorld(world)
	state := eng.World.NewWorldState("test", "tavern")

	result := eng.ProcessCommand(state, "ask barkeep about rumors")
	if !strings.Contains(result.Text, "many rumors") {
		t.Errorf("Expected rumors response, got: %s", result.Text)
	}
	if len(result.Choices) != 2 {
		t.Fatalf("Expected 2 choices, got %d", len(result.Choices))
	}

	// Follow the tree
	result = eng.ProcessCommand(state, "say 1")
	if !strings.Contains(result.Text, "dragon") {
		t.Errorf("Expected dragon response, got: %s", result.Text)
	}
}
