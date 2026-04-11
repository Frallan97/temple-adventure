package storygen

import (
	"fmt"
	"strings"

	"temple-adventure/engine"
)

// ValidateSpec checks a StorySpec for structural errors before expansion.
func ValidateSpec(spec *StorySpec) []string {
	var errs []string

	if spec.Title == "" {
		errs = append(errs, "title is required")
	}
	if spec.Slug == "" {
		errs = append(errs, "slug is required")
	}
	if len(spec.Rooms) == 0 {
		errs = append(errs, "at least one room is required")
	}
	if spec.StartRoom == "" {
		errs = append(errs, "start_room is required")
	} else if _, ok := spec.Rooms[spec.StartRoom]; !ok {
		errs = append(errs, fmt.Sprintf("start_room %q not found in rooms", spec.StartRoom))
	}

	// Check item references in rooms
	for roomID, room := range spec.Rooms {
		for _, itemID := range room.Items {
			if _, ok := spec.Items[itemID]; !ok {
				errs = append(errs, fmt.Sprintf("room %q references unknown item %q", roomID, itemID))
			}
		}
		for dir, targetID := range room.Connections {
			if _, ok := spec.Rooms[targetID]; !ok {
				errs = append(errs, fmt.Sprintf("room %q connection %q references unknown room %q", roomID, dir, targetID))
			}
		}
	}

	// Check puzzles
	winCount := 0
	validTypes := map[string]bool{
		"key_lock": true, "examine_learn": true, "fetch_quest": true,
		"timed_challenge": true, "win_condition": true,
		"combination_lock": true, "item_combine": true, "counter_puzzle": true,
	}
	seenIDs := map[string]bool{}

	for _, ps := range spec.Puzzles {
		if ps.ID == "" {
			errs = append(errs, "puzzle has empty id")
			continue
		}
		if seenIDs[ps.ID] {
			errs = append(errs, fmt.Sprintf("duplicate puzzle id %q", ps.ID))
		}
		seenIDs[ps.ID] = true

		if !validTypes[ps.Type] {
			errs = append(errs, fmt.Sprintf("puzzle %q has invalid type %q", ps.ID, ps.Type))
			continue
		}

		if ps.Room != "" {
			if _, ok := spec.Rooms[ps.Room]; !ok {
				errs = append(errs, fmt.Sprintf("puzzle %q references unknown room %q", ps.ID, ps.Room))
			}
		}

		// Type-specific checks
		switch ps.Type {
		case "key_lock":
			errs = append(errs, checkItemRef(spec, ps.ID, "key_item", ps.KeyItem)...)
			errs = append(errs, checkItemRef(spec, ps.ID, "lock_target", ps.LockTarget)...)
			errs = append(errs, checkRoomRef(spec, ps.ID, "unlock_room", ps.UnlockRoom)...)
			if ps.UnlockDirection == "" {
				errs = append(errs, fmt.Sprintf("puzzle %q: unlock_direction is required", ps.ID))
			}
		case "examine_learn":
			errs = append(errs, checkItemRef(spec, ps.ID, "source_item", ps.SourceItem)...)
			errs = append(errs, checkItemRef(spec, ps.ID, "target_item", ps.TargetItem)...)
			if ps.UnlockRoom != "" {
				errs = append(errs, checkRoomRef(spec, ps.ID, "unlock_room", ps.UnlockRoom)...)
			}
		case "fetch_quest":
			errs = append(errs, checkItemRef(spec, ps.ID, "fetch_item", ps.FetchItem)...)
			if ps.FetchRoom != "" {
				errs = append(errs, checkRoomRef(spec, ps.ID, "fetch_room", ps.FetchRoom)...)
			}
			if ps.UnlockRoom != "" {
				errs = append(errs, checkRoomRef(spec, ps.ID, "unlock_room", ps.UnlockRoom)...)
			}
		case "timed_challenge":
			errs = append(errs, checkItemRef(spec, ps.ID, "trigger_item", ps.TriggerItem)...)
			if ps.TurnLimit <= 0 {
				errs = append(errs, fmt.Sprintf("puzzle %q: turn_limit must be > 0", ps.ID))
			}
			if ps.FetchItem != "" {
				errs = append(errs, checkItemRef(spec, ps.ID, "fetch_item", ps.FetchItem)...)
			}
			if ps.UnlockRoom != "" {
				errs = append(errs, checkRoomRef(spec, ps.ID, "unlock_room", ps.UnlockRoom)...)
			}
			for _, fe := range ps.FailureEffects {
				if fe.Type != "move_player" && fe.Type != "lock_connection" {
					errs = append(errs, fmt.Sprintf("puzzle %q: invalid failure effect type %q", ps.ID, fe.Type))
				}
			}
		case "win_condition":
			errs = append(errs, checkItemRef(spec, ps.ID, "win_item", ps.WinItem)...)
			winCount++
		case "combination_lock":
			errs = append(errs, checkItemRef(spec, ps.ID, "combination_target", ps.CombinationTarget)...)
			if ps.CombinationSteps <= 0 {
				errs = append(errs, fmt.Sprintf("puzzle %q: combination_steps must be > 0", ps.ID))
			}
			if len(ps.CombinationTexts) > 0 && len(ps.CombinationTexts) != ps.CombinationSteps {
				errs = append(errs, fmt.Sprintf("puzzle %q: combination_texts length (%d) must match combination_steps (%d)", ps.ID, len(ps.CombinationTexts), ps.CombinationSteps))
			}
			if ps.UnlockRoom != "" {
				errs = append(errs, checkRoomRef(spec, ps.ID, "unlock_room", ps.UnlockRoom)...)
			}
		case "item_combine":
			errs = append(errs, checkItemRef(spec, ps.ID, "combine_item_a", ps.CombineItemA)...)
			errs = append(errs, checkItemRef(spec, ps.ID, "combine_item_b", ps.CombineItemB)...)
			errs = append(errs, checkItemRef(spec, ps.ID, "combine_result", ps.CombineResult)...)
			if ps.CombineResult != "" {
				if item, ok := spec.Items[ps.CombineResult]; ok && !item.Portable {
					errs = append(errs, fmt.Sprintf("puzzle %q: combine_result %q should be portable", ps.ID, ps.CombineResult))
				}
				for roomID, room := range spec.Rooms {
					for _, itemID := range room.Items {
						if itemID == ps.CombineResult {
							errs = append(errs, fmt.Sprintf("puzzle %q: combine_result %q should not be placed in room %q", ps.ID, ps.CombineResult, roomID))
						}
					}
				}
			}
		case "counter_puzzle":
			if len(ps.CounterItems) == 0 {
				errs = append(errs, fmt.Sprintf("puzzle %q: counter_items is required", ps.ID))
			}
			for _, itemID := range ps.CounterItems {
				errs = append(errs, checkItemRef(spec, ps.ID, "counter_items", itemID)...)
				if item, ok := spec.Items[itemID]; ok && !item.Portable {
					errs = append(errs, fmt.Sprintf("puzzle %q: counter item %q must be portable", ps.ID, itemID))
				}
			}
			if ps.CounterTarget <= 0 {
				errs = append(errs, fmt.Sprintf("puzzle %q: counter_target must be > 0", ps.ID))
			}
			if ps.CounterTarget > len(ps.CounterItems) {
				errs = append(errs, fmt.Sprintf("puzzle %q: counter_target (%d) exceeds counter_items count (%d)", ps.ID, ps.CounterTarget, len(ps.CounterItems)))
			}
			if ps.UnlockRoom != "" {
				errs = append(errs, checkRoomRef(spec, ps.ID, "unlock_room", ps.UnlockRoom)...)
			}
		}
	}

	if winCount == 0 {
		errs = append(errs, "at least one win_condition puzzle is required")
	}

	// Check NPCs
	for npcID, npc := range spec.Npcs {
		if npc.Name == "" {
			errs = append(errs, fmt.Sprintf("npc %q: name is required", npcID))
		}
		if npc.Room == "" {
			errs = append(errs, fmt.Sprintf("npc %q: room is required", npcID))
		} else if _, ok := spec.Rooms[npc.Room]; !ok {
			errs = append(errs, fmt.Sprintf("npc %q: references unknown room %q", npcID, npc.Room))
		}
	}

	return errs
}

func checkItemRef(spec *StorySpec, puzzleID, field, itemID string) []string {
	if itemID == "" {
		return []string{fmt.Sprintf("puzzle %q: %s is required", puzzleID, field)}
	}
	if _, ok := spec.Items[itemID]; !ok {
		return []string{fmt.Sprintf("puzzle %q: %s references unknown item %q", puzzleID, field, itemID)}
	}
	return nil
}

func checkRoomRef(spec *StorySpec, puzzleID, field, roomID string) []string {
	if roomID == "" {
		return []string{fmt.Sprintf("puzzle %q: %s is required", puzzleID, field)}
	}
	if _, ok := spec.Rooms[roomID]; !ok {
		return []string{fmt.Sprintf("puzzle %q: %s references unknown room %q", puzzleID, field, roomID)}
	}
	return nil
}

// ValidateWorldDeep performs semantic validation on an expanded WorldDefinition.
func ValidateWorldDeep(world *engine.WorldDefinition, startRoom string) []string {
	var errs []string

	// Basic structural checks
	if err := world.ValidateWithStartRoom(startRoom); err != nil {
		errs = append(errs, err.Error())
		return errs // stop early on basic failures
	}

	// Collect all variables set by effects
	setVars := collectSetVars(world)

	// Add engine-implicit variables
	for puzzleID := range world.Puzzles {
		setVars["puzzle."+puzzleID+".step"] = true
		setVars["puzzle."+puzzleID+".complete"] = true
		setVars["puzzle."+puzzleID+".failed"] = true
		setVars["puzzle."+puzzleID+".started"] = true
		setVars["puzzle."+puzzleID+".start_turn"] = true
	}

	// Check variables in conditions that are never set
	checkedVars := collectCheckedVars(world)
	for v := range checkedVars {
		if !setVars[v] {
			errs = append(errs, fmt.Sprintf("variable %q is checked in conditions but never set by any effect", v))
		}
	}

	// Check items referenced in effects
	errs = append(errs, checkEffectItemRefs(world)...)

	// Check puzzles with no steps
	for id, puzzle := range world.Puzzles {
		if len(puzzle.Steps) == 0 {
			errs = append(errs, fmt.Sprintf("puzzle %q has no steps", id))
		}
	}

	// Check unlock_connection targets
	errs = append(errs, checkUnlockTargets(world)...)

	// Check win condition exists
	if !hasWinCondition(world) {
		errs = append(errs, "no effect sets game status to 'completed' — story has no win condition")
	}

	return errs
}

func collectSetVars(world *engine.WorldDefinition) map[string]bool {
	vars := make(map[string]bool)
	walkEffects(world, func(e engine.Effect) {
		if e.Type == "set_var" || e.Type == "increment_var" {
			vars[e.Key] = true
		}
	})
	return vars
}

func collectCheckedVars(world *engine.WorldDefinition) map[string]bool {
	vars := make(map[string]bool)
	walkConditions(world, func(c engine.Condition) {
		if c.Type == "var_equals" || c.Type == "var_gte" || c.Type == "var_lte" {
			vars[c.Key] = true
		}
	})
	return vars
}

func checkEffectItemRefs(world *engine.WorldDefinition) []string {
	var errs []string
	walkEffects(world, func(e engine.Effect) {
		switch e.Type {
		case "add_item", "remove_item":
			if _, ok := world.Items[e.Key]; !ok {
				errs = append(errs, fmt.Sprintf("effect %s references unknown item %q", e.Type, e.Key))
			}
		case "add_room_item", "remove_room_item":
			if _, ok := world.Items[e.Key]; !ok {
				errs = append(errs, fmt.Sprintf("effect %s references unknown item %q", e.Type, e.Key))
			}
		}
	})
	return errs
}

func checkUnlockTargets(world *engine.WorldDefinition) []string {
	var errs []string
	walkEffects(world, func(e engine.Effect) {
		if e.Type != "unlock_connection" {
			return
		}
		parts := strings.SplitN(e.Key, ".", 2)
		if len(parts) != 2 {
			errs = append(errs, fmt.Sprintf("unlock_connection has malformed key %q", e.Key))
			return
		}
		roomID := parts[0]
		if _, ok := world.Rooms[roomID]; !ok {
			errs = append(errs, fmt.Sprintf("unlock_connection key references unknown room %q", roomID))
		}
		if targetRoom, ok := e.Value.(string); ok && targetRoom != "" {
			if _, ok := world.Rooms[targetRoom]; !ok {
				errs = append(errs, fmt.Sprintf("unlock_connection value references unknown room %q", targetRoom))
			}
		}
	})
	return errs
}

func hasWinCondition(world *engine.WorldDefinition) bool {
	found := false
	walkEffects(world, func(e engine.Effect) {
		if e.Type == "set_status" {
			if v, ok := e.Value.(string); ok && v == "completed" {
				found = true
			}
		}
	})
	return found
}

// walkEffects calls fn for every effect in the world (items + puzzles + npcs).
func walkEffects(world *engine.WorldDefinition, fn func(engine.Effect)) {
	for _, item := range world.Items {
		for _, inter := range item.Interactions {
			for _, eff := range inter.Effects {
				fn(eff)
			}
		}
	}
	for _, puzzle := range world.Puzzles {
		for _, step := range puzzle.Steps {
			for _, eff := range step.Effects {
				fn(eff)
			}
		}
		for _, eff := range puzzle.FailureEffects {
			fn(eff)
		}
	}
	for _, npc := range world.Npcs {
		for _, dl := range npc.Dialogue {
			for _, eff := range dl.Effects {
				fn(eff)
			}
		}
	}
}

// walkConditions calls fn for every condition in the world.
func walkConditions(world *engine.WorldDefinition, fn func(engine.Condition)) {
	for _, item := range world.Items {
		for _, inter := range item.Interactions {
			for _, cond := range inter.Conditions {
				fn(cond)
			}
		}
		for _, cd := range item.ConditionalDescriptions {
			fn(cd.Condition)
		}
	}
	for _, room := range world.Rooms {
		for _, cd := range room.ConditionalDescriptions {
			fn(cd.Condition)
		}
		for _, h := range room.Hints {
			if h.Condition != nil {
				fn(*h.Condition)
			}
		}
	}
	for _, puzzle := range world.Puzzles {
		for _, step := range puzzle.Steps {
			for _, cond := range step.Conditions {
				fn(cond)
			}
		}
	}
	for _, npc := range world.Npcs {
		for _, dl := range npc.Dialogue {
			for _, cond := range dl.Conditions {
				fn(cond)
			}
		}
		for _, cd := range npc.ConditionalDescriptions {
			fn(cd.Condition)
		}
	}
}
