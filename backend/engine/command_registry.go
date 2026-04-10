package engine

import "fmt"

type ActionFunc func(state *WorldState, world *WorldDefinition, cmd *ParsedCommand) *CommandResult

type CommandRegistry struct {
	actions map[string]ActionFunc
}

func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		actions: make(map[string]ActionFunc),
	}
}

func (r *CommandRegistry) Register(verb string, action ActionFunc) {
	r.actions[verb] = action
}

func (r *CommandRegistry) Execute(state *WorldState, world *WorldDefinition, cmd *ParsedCommand) *CommandResult {
	action, ok := r.actions[cmd.Verb]
	if !ok {
		// Try to resolve as an item interaction
		result := tryItemInteraction(state, world, cmd)
		if result != nil {
			return result
		}
		return &CommandResult{
			Text:       fmt.Sprintf("I don't understand '%s'. Type 'help' for available commands.", cmd.Raw),
			GameStatus: state.Status,
			TurnNumber: state.TurnNumber,
		}
	}
	return action(state, world, cmd)
}

func tryItemInteraction(state *WorldState, world *WorldDefinition, cmd *ParsedCommand) *CommandResult {
	// Look for an item in the room or inventory that has a matching interaction
	itemID := resolveItemID(state, world, cmd.Target)
	if itemID == "" {
		return nil
	}

	itemDef, ok := world.Items[itemID]
	if !ok {
		return nil
	}

	for _, interaction := range itemDef.Interactions {
		if interaction.Verb != cmd.Verb {
			continue
		}

		if EvaluateConditions(state, interaction.Conditions) {
			ApplyEffects(state, interaction.Effects)
			return &CommandResult{
				Text:       interaction.Response,
				GameStatus: state.Status,
				TurnNumber: state.TurnNumber,
			}
		}

		if interaction.FailResponse != "" {
			return &CommandResult{
				Text:       interaction.FailResponse,
				GameStatus: state.Status,
				TurnNumber: state.TurnNumber,
			}
		}
	}

	return nil
}

// resolveItemID finds an item by ID or alias in inventory or current room.
func resolveItemID(state *WorldState, world *WorldDefinition, target string) string {
	if target == "" {
		return ""
	}

	// Check inventory first
	for itemID := range state.Inventory {
		if matchesItem(world, itemID, target) {
			return itemID
		}
	}

	// Check room items
	roomItems := GetRoomItems(state, world, state.CurrentRoom)
	for _, itemID := range roomItems {
		if matchesItem(world, itemID, target) {
			return itemID
		}
	}

	return ""
}

func matchesItem(world *WorldDefinition, itemID, target string) bool {
	if itemID == target {
		return true
	}
	itemDef, ok := world.Items[itemID]
	if !ok {
		return false
	}
	if itemDef.Name == target {
		return true
	}
	for _, alias := range itemDef.Aliases {
		if alias == target {
			return true
		}
	}
	return false
}
