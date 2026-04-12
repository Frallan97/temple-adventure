package storygen

import (
	"fmt"
	"strings"

	"temple-adventure/engine"
)

// Expand converts a StorySpec into a fully wired WorldDefinition.
func Expand(spec *StorySpec) (*engine.WorldDefinition, error) {
	e := &expander{
		spec: spec,
		world: &engine.WorldDefinition{
			Rooms:   make(map[string]*engine.RoomDef),
			Items:   make(map[string]*engine.ItemDef),
			Puzzles: make(map[string]*engine.PuzzleDef),
			Npcs:    make(map[string]*engine.NpcDef),
		},
	}

	e.buildRooms()
	e.buildItems()
	e.buildNpcs()

	for i := range spec.Puzzles {
		ps := &spec.Puzzles[i]
		var err error
		switch ps.Type {
		case "key_lock":
			err = e.expandKeyLock(ps)
		case "examine_learn":
			err = e.expandExamineLearn(ps)
		case "fetch_quest":
			err = e.expandFetchQuest(ps)
		case "timed_challenge":
			err = e.expandTimedChallenge(ps)
		case "win_condition":
			err = e.expandWinCondition(ps)
		case "combination_lock":
			err = e.expandCombinationLock(ps)
		case "item_combine":
			err = e.expandItemCombine(ps)
		case "counter_puzzle":
			err = e.expandCounterPuzzle(ps)
		default:
			err = fmt.Errorf("unknown puzzle type %q", ps.Type)
		}
		if err != nil {
			return nil, fmt.Errorf("expanding puzzle %q: %w", ps.ID, err)
		}
	}

	return e.world, nil
}

type expander struct {
	spec  *StorySpec
	world *engine.WorldDefinition
}

func (e *expander) buildRooms() {
	for id, rs := range e.spec.Rooms {
		conns := make(map[string]string)
		for k, v := range rs.Connections {
			conns[k] = v
		}
		e.world.Rooms[id] = &engine.RoomDef{
			ID:          id,
			Name:        rs.Name,
			Description: rs.Description,
			Connections: conns,
			Items:       append([]string{}, rs.Items...),
		}
	}
}

func (e *expander) buildItems() {
	for id, is := range e.spec.Items {
		item := &engine.ItemDef{
			ID:          id,
			Name:        is.Name,
			Description: is.Description,
			Aliases:     append([]string{}, is.Aliases...),
			Portable:    is.Portable,
		}
		if is.ExamineText != "" {
			item.Interactions = append(item.Interactions, engine.Interaction{
				Verb:     "examine",
				Response: is.ExamineText,
			})
		}
		e.world.Items[id] = item
	}
}

func (e *expander) buildNpcs() {
	for id, ns := range e.spec.Npcs {
		npc := &engine.NpcDef{
			ID:          id,
			Name:        ns.Name,
			Description: ns.Description,
			Aliases:     append([]string{}, ns.Aliases...),
			Room:        ns.Room,
		}

		if len(ns.Dialogue) > 0 {
			// Full dialogue tree mode
			for _, dn := range ns.Dialogue {
				dl := engine.DialogueLine{
					NodeID:   dn.NodeID,
					Topic:    dn.Topic,
					Response: dn.Text,
				}
				for _, cs := range dn.Choices {
					choice := engine.DialogueChoice{
						Text:     cs.Text,
						NextNode: cs.NextNode,
					}
					if cs.NeedItem != "" {
						choice.Conditions = append(choice.Conditions, engine.Condition{
							Type: "has_item", Key: cs.NeedItem,
						})
					}
					if cs.GiveItem != "" {
						choice.Effects = append(choice.Effects, engine.Effect{
							Type: "add_item", Key: cs.GiveItem,
						})
					}
					if cs.SetVar != "" {
						if parts := strings.SplitN(cs.SetVar, "=", 2); len(parts) == 2 {
							choice.Effects = append(choice.Effects, engine.Effect{
								Type: "set_var", Key: parts[0], Value: parts[1],
							})
						}
					}
					dl.Choices = append(dl.Choices, choice)
				}
				npc.Dialogue = append(npc.Dialogue, dl)
			}
		} else {
			// Simple greeting/topics mode (backward compatible)
			if ns.Greeting != "" {
				npc.Dialogue = append(npc.Dialogue, engine.DialogueLine{
					Topic:    "",
					Response: ns.Greeting,
				})
			}
			for topic, response := range ns.Topics {
				npc.Dialogue = append(npc.Dialogue, engine.DialogueLine{
					Topic:    topic,
					Response: response,
				})
			}
		}

		e.world.Npcs[id] = npc
	}
}

func (e *expander) expandKeyLock(ps *PuzzleSpec) error {
	verb := or(ps.LockVerb, "use")
	unlockedVar := ps.ID + "_unlocked"

	item := e.world.Items[ps.LockTarget]
	if item == nil {
		return fmt.Errorf("lock_target item %q not found", ps.LockTarget)
	}

	// Success: has key + in room → unlock
	item.Interactions = append(item.Interactions, engine.Interaction{
		Verb: verb,
		Conditions: []engine.Condition{
			{Type: "has_item", Key: ps.KeyItem},
			{Type: "in_room", Key: ps.Room},
		},
		Effects: []engine.Effect{
			{Type: "set_var", Key: unlockedVar, Value: true},
			{Type: "remove_item", Key: ps.KeyItem},
		},
		Response: ps.CompletionText,
	})

	// Fail: wrong conditions
	if ps.LockFailText != "" {
		item.Interactions = append(item.Interactions, engine.Interaction{
			Verb: verb,
			Conditions: []engine.Condition{
				{Type: "in_room", Key: ps.Room},
			},
			FailResponse: ps.LockFailText,
		})
	}

	e.world.Puzzles[ps.ID] = &engine.PuzzleDef{
		ID:          ps.ID,
		Name:        ps.Name,
		Description: ps.Description,
		Steps: []engine.PuzzleStep{
			{
				StepID: "unlock",
				Prompt: ps.Description,
				Conditions: []engine.Condition{
					{Type: "var_equals", Key: unlockedVar, Value: true},
				},
				Effects: []engine.Effect{
					{Type: "unlock_connection", Key: ps.Room + "." + ps.UnlockDirection, Value: ps.UnlockRoom},
				},
			},
		},
		CompletionText: ps.CompletionText,
	}

	e.removeLockAndRegister(ps.Room, ps.UnlockDirection, ps.UnlockRoom, ps.ID)
	e.addConditionalDesc(ps.Room, unlockedVar)

	return nil
}

func (e *expander) expandExamineLearn(ps *PuzzleSpec) error {
	learnedVar := ps.ID + "_learned"
	solvedVar := ps.ID + "_solved"
	targetVerb := or(ps.TargetVerb, "use")

	srcItem := e.world.Items[ps.SourceItem]
	if srcItem == nil {
		return fmt.Errorf("source_item %q not found", ps.SourceItem)
	}
	tgtItem := e.world.Items[ps.TargetItem]
	if tgtItem == nil {
		return fmt.Errorf("target_item %q not found", ps.TargetItem)
	}

	// Examine source → learn
	srcItem.Interactions = append(srcItem.Interactions, engine.Interaction{
		Verb: "examine",
		Effects: []engine.Effect{
			{Type: "set_var", Key: learnedVar, Value: true},
		},
		Response: ps.SourceLearnText,
	})

	// Interact with target → solve (requires learned)
	solveInteraction := engine.Interaction{
		Verb: targetVerb,
		Conditions: []engine.Condition{
			{Type: "var_equals", Key: learnedVar, Value: true},
		},
		Effects: []engine.Effect{
			{Type: "set_var", Key: solvedVar, Value: true},
		},
		Response:     ps.TargetSuccessText,
		FailResponse: ps.TargetFailText,
	}

	steps := []engine.PuzzleStep{
		{
			StepID:     "learn",
			Prompt:     "Find and examine the clue.",
			Conditions: []engine.Condition{{Type: "var_equals", Key: learnedVar, Value: true}},
		},
		{
			StepID:     "solve",
			Prompt:     "Use what you learned.",
			Conditions: []engine.Condition{{Type: "var_equals", Key: solvedVar, Value: true}},
		},
	}

	condVar := solvedVar

	// If also unlocks a direction, add unlock mechanic
	if ps.UnlockDirection != "" && ps.UnlockRoom != "" {
		unlockVerb := or(ps.LockVerb, "use")

		if unlockVerb != targetVerb {
			// Two separate interactions: solve verb + unlock verb
			// e.g., "turn panel" (solve) then "use panel" (unlock)
			unlockedVar := ps.ID + "_unlocked"
			condVar = unlockedVar

			tgtItem.Interactions = append(tgtItem.Interactions, solveInteraction)
			tgtItem.Interactions = append(tgtItem.Interactions, engine.Interaction{
				Verb: unlockVerb,
				Conditions: []engine.Condition{
					{Type: "var_equals", Key: solvedVar, Value: true},
				},
				Effects: []engine.Effect{
					{Type: "set_var", Key: unlockedVar, Value: true},
					{Type: "unlock_connection", Key: ps.Room + "." + ps.UnlockDirection, Value: ps.UnlockRoom},
				},
				Response:     ps.CompletionText,
				FailResponse: ps.LockFailText,
			})

			steps = append(steps, engine.PuzzleStep{
				StepID:     "unlock",
				Prompt:     "Activate the mechanism.",
				Conditions: []engine.Condition{{Type: "var_equals", Key: unlockedVar, Value: true}},
			})
		} else {
			// Same verb handles solve + unlock in one step
			solveInteraction.Effects = append(solveInteraction.Effects,
				engine.Effect{Type: "unlock_connection", Key: ps.Room + "." + ps.UnlockDirection, Value: ps.UnlockRoom},
			)
			tgtItem.Interactions = append(tgtItem.Interactions, solveInteraction)

			// Add unlock effects to the solve puzzle step
			steps[len(steps)-1].Effects = []engine.Effect{
				{Type: "unlock_connection", Key: ps.Room + "." + ps.UnlockDirection, Value: ps.UnlockRoom},
			}
		}

		e.removeLockAndRegister(ps.Room, ps.UnlockDirection, ps.UnlockRoom, ps.ID)
	} else {
		tgtItem.Interactions = append(tgtItem.Interactions, solveInteraction)
		if room := e.world.Rooms[ps.Room]; room != nil {
			room.Puzzles = append(room.Puzzles, ps.ID)
		}
	}

	e.world.Puzzles[ps.ID] = &engine.PuzzleDef{
		ID:             ps.ID,
		Name:           ps.Name,
		Description:    ps.Description,
		Steps:          steps,
		CompletionText: ps.CompletionText,
	}

	e.addConditionalDesc(ps.Room, condVar)

	return nil
}

func (e *expander) expandFetchQuest(ps *PuzzleSpec) error {
	placedVar := ps.ID + "_placed"
	fetchVerb := or(ps.FetchVerb, "use")
	fetchRoom := or(ps.FetchRoom, ps.Room)

	item := e.world.Items[ps.FetchItem]
	if item == nil {
		return fmt.Errorf("fetch_item %q not found", ps.FetchItem)
	}

	effects := []engine.Effect{
		{Type: "set_var", Key: placedVar, Value: true},
	}
	if ps.FetchConsumeItem {
		effects = append(effects, engine.Effect{Type: "remove_item", Key: ps.FetchItem})
	}

	item.Interactions = append(item.Interactions, engine.Interaction{
		Verb: fetchVerb,
		Conditions: []engine.Condition{
			{Type: "has_item", Key: ps.FetchItem},
			{Type: "in_room", Key: fetchRoom},
		},
		Effects:  effects,
		Response: ps.FetchSuccessText,
	})

	var stepEffects []engine.Effect
	if ps.UnlockDirection != "" && ps.UnlockRoom != "" {
		stepEffects = append(stepEffects, engine.Effect{
			Type: "unlock_connection", Key: fetchRoom + "." + ps.UnlockDirection, Value: ps.UnlockRoom,
		})
		e.removeLockAndRegister(fetchRoom, ps.UnlockDirection, ps.UnlockRoom, ps.ID)
	} else {
		if room := e.world.Rooms[ps.Room]; room != nil {
			room.Puzzles = append(room.Puzzles, ps.ID)
		}
	}

	e.world.Puzzles[ps.ID] = &engine.PuzzleDef{
		ID:          ps.ID,
		Name:        ps.Name,
		Description: ps.Description,
		Steps: []engine.PuzzleStep{
			{
				StepID:     "place",
				Prompt:     ps.Description,
				Conditions: []engine.Condition{{Type: "var_equals", Key: placedVar, Value: true}},
				Effects:    stepEffects,
			},
		},
		CompletionText: ps.CompletionText,
	}

	e.addConditionalDesc(fetchRoom, placedVar)

	return nil
}

func (e *expander) expandTimedChallenge(ps *PuzzleSpec) error {
	triggerVerb := or(ps.TriggerVerb, "turn")
	startedVar := "puzzle." + ps.ID + ".started"
	startTurnVar := "puzzle." + ps.ID + ".start_turn"

	triggerItem := e.world.Items[ps.TriggerItem]
	if triggerItem == nil {
		return fmt.Errorf("trigger_item %q not found", ps.TriggerItem)
	}

	// Start timer interaction
	triggerItem.Interactions = append(triggerItem.Interactions, engine.Interaction{
		Verb: triggerVerb,
		Conditions: []engine.Condition{
			{Type: "var_equals", Key: startedVar, Value: true, Negate: true},
		},
		Effects: []engine.Effect{
			{Type: "set_var", Key: startedVar, Value: true},
			{Type: "set_var", Key: startTurnVar, Value: "__CURRENT_TURN__"},
		},
		Response: ps.TriggerText,
	})

	// Already activated
	triggerItem.Interactions = append(triggerItem.Interactions, engine.Interaction{
		Verb: triggerVerb,
		Conditions: []engine.Condition{
			{Type: "var_equals", Key: startedVar, Value: true},
		},
		Response: "It's already been activated.",
	})

	// Convert failure effects
	var failureEffects []engine.Effect
	for _, fe := range ps.FailureEffects {
		switch fe.Type {
		case "move_player":
			failureEffects = append(failureEffects, engine.Effect{Type: "move_player", Value: fe.Room})
		case "lock_connection":
			failureEffects = append(failureEffects, engine.Effect{Type: "lock_connection", Key: fe.Room + "." + fe.Direction})
		}
	}

	// Steps: activate first
	steps := []engine.PuzzleStep{
		{
			StepID:     "activate",
			Prompt:     "Activate the mechanism.",
			Conditions: []engine.Condition{{Type: "var_equals", Key: startedVar, Value: true}},
		},
	}

	// If fetch fields present, add fetch mechanic
	if ps.FetchItem != "" {
		placedVar := ps.ID + "_placed"
		fetchVerb := or(ps.FetchVerb, "use")
		fetchRoom := or(ps.FetchRoom, ps.Room)

		fetchItem := e.world.Items[ps.FetchItem]
		if fetchItem == nil {
			return fmt.Errorf("fetch_item %q not found", ps.FetchItem)
		}

		fetchEffects := []engine.Effect{
			{Type: "set_var", Key: placedVar, Value: true},
		}
		if ps.FetchConsumeItem {
			fetchEffects = append(fetchEffects, engine.Effect{Type: "remove_item", Key: ps.FetchItem})
		}

		fetchItem.Interactions = append(fetchItem.Interactions, engine.Interaction{
			Verb: fetchVerb,
			Conditions: []engine.Condition{
				{Type: "in_room", Key: fetchRoom},
				{Type: "var_equals", Key: startedVar, Value: true},
			},
			Effects:  fetchEffects,
			Response: ps.FetchSuccessText,
		})

		var solveEffects []engine.Effect
		if ps.UnlockDirection != "" && ps.UnlockRoom != "" {
			solveEffects = append(solveEffects, engine.Effect{
				Type: "unlock_connection", Key: fetchRoom + "." + ps.UnlockDirection, Value: ps.UnlockRoom,
			})
			if room := e.world.Rooms[fetchRoom]; room != nil {
				delete(room.Connections, ps.UnlockDirection)
			}
			addReverseConnection(e.world, ps.UnlockRoom, ps.UnlockDirection, fetchRoom)
		}

		steps = append(steps, engine.PuzzleStep{
			StepID:     "solve",
			Prompt:     "Complete the challenge before time runs out.",
			Conditions: []engine.Condition{{Type: "var_equals", Key: placedVar, Value: true}},
			Effects:    solveEffects,
		})
	}

	e.world.Puzzles[ps.ID] = &engine.PuzzleDef{
		ID:             ps.ID,
		Name:           ps.Name,
		Description:    ps.Description,
		Steps:          steps,
		TimedWindow:    &engine.TimedWindow{StartTrigger: startedVar, TurnLimit: ps.TurnLimit},
		FailureEffects: failureEffects,
		FailureText:    ps.FailureText,
		CompletionText: ps.CompletionText,
	}

	if room := e.world.Rooms[ps.Room]; room != nil {
		room.Puzzles = append(room.Puzzles, ps.ID)
	}

	return nil
}

func (e *expander) expandWinCondition(ps *PuzzleSpec) error {
	winVerb := or(ps.WinVerb, "take")

	item := e.world.Items[ps.WinItem]
	if item == nil {
		return fmt.Errorf("win_item %q not found", ps.WinItem)
	}

	if len(ps.Endings) > 0 {
		// Conditional endings: create one interaction per ending.
		// Specific endings (with conditions) first, fallback (no conditions) last.
		for _, ending := range ps.Endings {
			var conditions []engine.Condition
			for k, v := range ending.Conditions {
				conditions = append(conditions, engine.Condition{
					Type: "var_equals", Key: k, Value: parseConditionValue(v),
				})
			}

			effects := []engine.Effect{
				{Type: "set_var", Key: "game_won", Value: true},
				{Type: "set_status", Value: "completed"},
			}
			if ending.ID != "" {
				effects = append(effects, engine.Effect{Type: "set_ending_id", Value: ending.ID})
			}
			if ending.Title != "" {
				effects = append(effects, engine.Effect{Type: "set_ending_title", Value: ending.Title})
			}

			item.Interactions = append(item.Interactions, engine.Interaction{
				Verb:       winVerb,
				Conditions: conditions,
				Effects:    effects,
				Response:   ending.Text,
			})
		}
	} else {
		// Single ending (backward compatible)
		effects := []engine.Effect{
			{Type: "set_var", Key: "game_won", Value: true},
			{Type: "set_status", Value: "completed"},
		}
		if ps.EndingID != "" {
			effects = append(effects, engine.Effect{Type: "set_ending_id", Value: ps.EndingID})
		}
		if ps.EndingTitle != "" {
			effects = append(effects, engine.Effect{Type: "set_ending_title", Value: ps.EndingTitle})
		}

		item.Interactions = append(item.Interactions, engine.Interaction{
			Verb:    winVerb,
			Effects: effects,
			Response: ps.WinText,
		})
	}

	return nil
}

// parseConditionValue converts string values from EndingSpec.Conditions
// to typed values the engine can match (bool, int, or string).
func parseConditionValue(s string) interface{} {
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}
	return s
}

func (e *expander) expandCombinationLock(ps *PuzzleSpec) error {
	verb := or(ps.CombinationVerb, "turn")
	startedVar := ps.ID + "_started"
	stepVar := ps.ID + "_step"
	solvedVar := ps.ID + "_solved"
	n := ps.CombinationSteps

	item := e.world.Items[ps.CombinationTarget]
	if item == nil {
		return fmt.Errorf("combination_target item %q not found", ps.CombinationTarget)
	}

	stepText := func(i int) string {
		if i < len(ps.CombinationTexts) {
			return ps.CombinationTexts[i]
		}
		if i == n-1 {
			return or(ps.CompletionText, "The mechanism clicks into place!")
		}
		return fmt.Sprintf("Click. Step %d of %d.", i+1, n)
	}

	// Already solved guard
	item.Interactions = append(item.Interactions, engine.Interaction{
		Verb: verb,
		Conditions: []engine.Condition{
			{Type: "var_equals", Key: solvedVar, Value: true},
		},
		Response: "It's already been solved.",
	})

	if n == 1 {
		// Single step: goes straight to solved
		item.Interactions = append(item.Interactions, engine.Interaction{
			Verb: verb,
			Conditions: []engine.Condition{
				{Type: "var_equals", Key: startedVar, Value: true, Negate: true},
			},
			Effects: []engine.Effect{
				{Type: "set_var", Key: startedVar, Value: true},
				{Type: "set_var", Key: solvedVar, Value: true},
			},
			Response: stepText(0),
		})
	} else {
		// Step 0: not yet started
		item.Interactions = append(item.Interactions, engine.Interaction{
			Verb: verb,
			Conditions: []engine.Condition{
				{Type: "var_equals", Key: startedVar, Value: true, Negate: true},
			},
			Effects: []engine.Effect{
				{Type: "set_var", Key: startedVar, Value: true},
				{Type: "set_var", Key: stepVar, Value: 1},
			},
			Response: stepText(0),
		})

		// Intermediate steps 1..N-2
		for i := 1; i < n-1; i++ {
			item.Interactions = append(item.Interactions, engine.Interaction{
				Verb: verb,
				Conditions: []engine.Condition{
					{Type: "var_equals", Key: stepVar, Value: i},
				},
				Effects: []engine.Effect{
					{Type: "set_var", Key: stepVar, Value: i + 1},
				},
				Response: stepText(i),
			})
		}

		// Final step
		item.Interactions = append(item.Interactions, engine.Interaction{
			Verb: verb,
			Conditions: []engine.Condition{
				{Type: "var_equals", Key: stepVar, Value: n - 1},
			},
			Effects: []engine.Effect{
				{Type: "set_var", Key: solvedVar, Value: true},
			},
			Response: stepText(n - 1),
		})
	}

	// Puzzle definition
	var stepEffects []engine.Effect
	if ps.UnlockDirection != "" && ps.UnlockRoom != "" {
		stepEffects = append(stepEffects, engine.Effect{
			Type: "unlock_connection", Key: ps.Room + "." + ps.UnlockDirection, Value: ps.UnlockRoom,
		})
		e.removeLockAndRegister(ps.Room, ps.UnlockDirection, ps.UnlockRoom, ps.ID)
	} else {
		if room := e.world.Rooms[ps.Room]; room != nil {
			room.Puzzles = append(room.Puzzles, ps.ID)
		}
	}

	e.world.Puzzles[ps.ID] = &engine.PuzzleDef{
		ID:          ps.ID,
		Name:        ps.Name,
		Description: ps.Description,
		Steps: []engine.PuzzleStep{
			{
				StepID:     "solve",
				Prompt:     ps.Description,
				Conditions: []engine.Condition{{Type: "var_equals", Key: solvedVar, Value: true}},
				Effects:    stepEffects,
			},
		},
		CompletionText: ps.CompletionText,
	}

	e.addConditionalDesc(ps.Room, solvedVar)

	return nil
}

func (e *expander) expandItemCombine(ps *PuzzleSpec) error {
	verb := or(ps.CombineVerb, "use")
	combinedVar := ps.ID + "_combined"

	itemA := e.world.Items[ps.CombineItemA]
	if itemA == nil {
		return fmt.Errorf("combine_item_a %q not found", ps.CombineItemA)
	}
	if e.world.Items[ps.CombineItemB] == nil {
		return fmt.Errorf("combine_item_b %q not found", ps.CombineItemB)
	}
	if e.world.Items[ps.CombineResult] == nil {
		return fmt.Errorf("combine_result %q not found", ps.CombineResult)
	}

	// Success interaction: has both items
	effects := []engine.Effect{
		{Type: "add_item", Key: ps.CombineResult},
		{Type: "set_var", Key: combinedVar, Value: true},
	}
	if ps.CombineConsumeA {
		effects = append(effects, engine.Effect{Type: "remove_item", Key: ps.CombineItemA})
	}
	if ps.CombineConsumeB {
		effects = append(effects, engine.Effect{Type: "remove_item", Key: ps.CombineItemB})
	}

	itemA.Interactions = append(itemA.Interactions, engine.Interaction{
		Verb: verb,
		Conditions: []engine.Condition{
			{Type: "has_item", Key: ps.CombineItemA},
			{Type: "has_item", Key: ps.CombineItemB},
		},
		Effects:  effects,
		Response: ps.CombineText,
	})

	// Fail interaction: only has item A
	if ps.CombineFailText != "" {
		itemA.Interactions = append(itemA.Interactions, engine.Interaction{
			Verb: verb,
			Conditions: []engine.Condition{
				{Type: "has_item", Key: ps.CombineItemA},
			},
			FailResponse: ps.CombineFailText,
		})
	}

	// Puzzle definition
	e.world.Puzzles[ps.ID] = &engine.PuzzleDef{
		ID:          ps.ID,
		Name:        ps.Name,
		Description: ps.Description,
		Steps: []engine.PuzzleStep{
			{
				StepID:     "combine",
				Prompt:     ps.Description,
				Conditions: []engine.Condition{{Type: "var_equals", Key: combinedVar, Value: true}},
			},
		},
		CompletionText: ps.CompletionText,
	}

	if room := e.world.Rooms[ps.Room]; room != nil {
		room.Puzzles = append(room.Puzzles, ps.ID)
	}

	e.addConditionalDesc(ps.Room, combinedVar)

	return nil
}

func (e *expander) expandCounterPuzzle(ps *PuzzleSpec) error {
	verb := or(ps.CounterVerb, "use")
	countVar := ps.ID + "_count"
	reachedVar := ps.ID + "_reached"

	for _, itemID := range ps.CounterItems {
		item := e.world.Items[itemID]
		if item == nil {
			return fmt.Errorf("counter_items references unknown item %q", itemID)
		}

		doneVar := ps.ID + "_" + itemID + "_done"

		// Response text for this item
		responseText := ps.CounterDefaultText
		if ps.CounterItemTexts != nil {
			if text, ok := ps.CounterItemTexts[itemID]; ok {
				responseText = text
			}
		}
		if responseText == "" {
			responseText = "Done."
		}

		// Already done guard
		item.Interactions = append(item.Interactions, engine.Interaction{
			Verb: verb,
			Conditions: []engine.Condition{
				{Type: "has_item", Key: itemID},
				{Type: "var_equals", Key: doneVar, Value: true},
			},
			Response: "You've already used this.",
		})

		// Success interaction
		effects := []engine.Effect{
			{Type: "increment_var", Key: countVar, Value: 1},
			{Type: "set_var", Key: doneVar, Value: true},
		}
		if ps.CounterConsumeItems {
			effects = append(effects, engine.Effect{Type: "remove_item", Key: itemID})
		}

		item.Interactions = append(item.Interactions, engine.Interaction{
			Verb: verb,
			Conditions: []engine.Condition{
				{Type: "has_item", Key: itemID},
				{Type: "var_equals", Key: doneVar, Value: true, Negate: true},
			},
			Effects:  effects,
			Response: responseText,
		})
	}

	// Puzzle definition
	stepEffects := []engine.Effect{
		{Type: "set_var", Key: reachedVar, Value: true},
	}
	if ps.UnlockDirection != "" && ps.UnlockRoom != "" {
		stepEffects = append(stepEffects, engine.Effect{
			Type: "unlock_connection", Key: ps.Room + "." + ps.UnlockDirection, Value: ps.UnlockRoom,
		})
		e.removeLockAndRegister(ps.Room, ps.UnlockDirection, ps.UnlockRoom, ps.ID)
	} else {
		if room := e.world.Rooms[ps.Room]; room != nil {
			room.Puzzles = append(room.Puzzles, ps.ID)
		}
	}

	e.world.Puzzles[ps.ID] = &engine.PuzzleDef{
		ID:          ps.ID,
		Name:        ps.Name,
		Description: ps.Description,
		Steps: []engine.PuzzleStep{
			{
				StepID:     "collect",
				Prompt:     ps.Description,
				Conditions: []engine.Condition{{Type: "var_gte", Key: countVar, Value: ps.CounterTarget}},
				Effects:    stepEffects,
			},
		},
		CompletionText: ps.CompletionText,
	}

	e.addConditionalDesc(ps.Room, reachedVar)

	return nil
}

// removeLockAndRegister removes a connection from the base room (it starts locked),
// adds the reverse connection, and registers the puzzle on the room.
func (e *expander) removeLockAndRegister(roomID, direction, targetRoomID, puzzleID string) {
	if room := e.world.Rooms[roomID]; room != nil {
		delete(room.Connections, direction)
		room.Puzzles = append(room.Puzzles, puzzleID)
	}
	addReverseConnection(e.world, targetRoomID, direction, roomID)
}

// addConditionalDesc adds a replace:true conditional description to a room
// when the given variable becomes true, using the room's DescriptionAfterPuzzle.
func (e *expander) addConditionalDesc(roomID, condVar string) {
	roomSpec, ok := e.spec.Rooms[roomID]
	if !ok || roomSpec.DescriptionAfterPuzzle == "" {
		return
	}
	room := e.world.Rooms[roomID]
	if room == nil {
		return
	}
	// Prepend so more specific (later puzzle) descriptions come first
	room.ConditionalDescriptions = append([]engine.ConditionalText{
		{
			Condition: engine.Condition{Type: "var_equals", Key: condVar, Value: true},
			Text:      roomSpec.DescriptionAfterPuzzle,
			Replace:   true,
		},
	}, room.ConditionalDescriptions...)
}

// addReverseConnection ensures the target room has a connection back.
func addReverseConnection(world *engine.WorldDefinition, targetRoomID, direction, sourceRoomID string) {
	rev, ok := reverseDirection[direction]
	if !ok {
		return
	}
	targetRoom := world.Rooms[targetRoomID]
	if targetRoom == nil {
		return
	}
	if _, exists := targetRoom.Connections[rev]; !exists {
		if targetRoom.Connections == nil {
			targetRoom.Connections = make(map[string]string)
		}
		targetRoom.Connections[rev] = sourceRoomID
	}
}

func or(val, fallback string) string {
	if val != "" {
		return val
	}
	return fallback
}
