package engine

import (
	"fmt"
	"strings"
)

func RegisterBuiltinActions(registry *CommandRegistry) {
	registry.Register("look", actionLook)
	registry.Register("move", actionMove)
	registry.Register("take", actionTake)
	registry.Register("drop", actionDrop)
	registry.Register("use", actionUse)
	registry.Register("inventory", actionInventory)
	registry.Register("help", actionHelp)
	registry.Register("hint", actionHint)
}

func actionLook(state *WorldState, world *WorldDefinition, cmd *ParsedCommand) *CommandResult {
	// If target specified, look at a specific item
	if cmd.Target != "" {
		return actionExamine(state, world, cmd)
	}

	return &CommandResult{
		Text:       describeRoom(state, world),
		GameStatus: state.Status,
		TurnNumber: state.TurnNumber,
	}
}

func actionExamine(state *WorldState, world *WorldDefinition, cmd *ParsedCommand) *CommandResult {
	itemID := resolveItemID(state, world, cmd.Target)
	if itemID == "" {
		return &CommandResult{
			Text:       fmt.Sprintf("You don't see '%s' here.", cmd.Target),
			GameStatus: state.Status,
			TurnNumber: state.TurnNumber,
		}
	}

	itemDef := world.Items[itemID]
	desc := itemDef.Description

	// Check for conditional descriptions
	for _, cd := range itemDef.ConditionalDescriptions {
		if EvaluateCondition(state, cd.Condition) {
			if cd.Replace {
				desc = cd.Text
			} else {
				desc = desc + "\n" + cd.Text
			}
		}
	}

	// Also try to trigger an "examine" interaction
	for _, interaction := range itemDef.Interactions {
		if interaction.Verb == "examine" || interaction.Verb == "look" {
			if EvaluateConditions(state, interaction.Conditions) {
				ApplyEffects(state, interaction.Effects)
				if interaction.Response != "" {
					desc = interaction.Response
				}
			}
		}
	}

	return &CommandResult{
		Text:       desc,
		GameStatus: state.Status,
		TurnNumber: state.TurnNumber,
	}
}

func actionMove(state *WorldState, world *WorldDefinition, cmd *ParsedCommand) *CommandResult {
	if cmd.Target == "" {
		return &CommandResult{
			Text:       "Move where? Specify a direction (north, south, east, west, up, down).",
			GameStatus: state.Status,
			TurnNumber: state.TurnNumber,
		}
	}

	connections := GetRoomConnections(state, world, state.CurrentRoom)
	targetRoom, ok := connections[cmd.Target]
	if !ok {
		return &CommandResult{
			Text:       fmt.Sprintf("You can't go %s from here.", cmd.Target),
			GameStatus: state.Status,
			TurnNumber: state.TurnNumber,
		}
	}

	state.CurrentRoom = targetRoom
	return &CommandResult{
		Text:        describeRoom(state, world),
		RoomChanged: true,
		GameStatus:  state.Status,
		TurnNumber:  state.TurnNumber,
	}
}

func actionTake(state *WorldState, world *WorldDefinition, cmd *ParsedCommand) *CommandResult {
	if cmd.Target == "" {
		return &CommandResult{
			Text:       "Take what?",
			GameStatus: state.Status,
			TurnNumber: state.TurnNumber,
		}
	}

	// Find the item in the room
	roomItems := GetRoomItems(state, world, state.CurrentRoom)
	var foundID string
	for _, itemID := range roomItems {
		if matchesItem(world, itemID, cmd.Target) {
			foundID = itemID
			break
		}
	}

	if foundID == "" {
		return &CommandResult{
			Text:       fmt.Sprintf("You don't see '%s' here.", cmd.Target),
			GameStatus: state.Status,
			TurnNumber: state.TurnNumber,
		}
	}

	itemDef := world.Items[foundID]
	if !itemDef.Portable {
		return &CommandResult{
			Text:       fmt.Sprintf("You can't pick up the %s.", itemDef.Name),
			GameStatus: state.Status,
			TurnNumber: state.TurnNumber,
		}
	}

	// Check if item has a "take" interaction with custom effects
	for _, interaction := range itemDef.Interactions {
		if interaction.Verb == "take" {
			if EvaluateConditions(state, interaction.Conditions) {
				// Remove from room, add to inventory
				rs := state.RoomStates[state.CurrentRoom]
				rs.RemovedItems[foundID] = true
				delete(rs.AddedItems, foundID)
				state.Inventory[foundID] = true
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
	}

	// Default take behavior
	rs := state.RoomStates[state.CurrentRoom]
	rs.RemovedItems[foundID] = true
	delete(rs.AddedItems, foundID)
	state.Inventory[foundID] = true

	return &CommandResult{
		Text:       fmt.Sprintf("You pick up the %s.", itemDef.Name),
		GameStatus: state.Status,
		TurnNumber: state.TurnNumber,
	}
}

func actionDrop(state *WorldState, world *WorldDefinition, cmd *ParsedCommand) *CommandResult {
	if cmd.Target == "" {
		return &CommandResult{
			Text:       "Drop what?",
			GameStatus: state.Status,
			TurnNumber: state.TurnNumber,
		}
	}

	var foundID string
	for itemID := range state.Inventory {
		if matchesItem(world, itemID, cmd.Target) {
			foundID = itemID
			break
		}
	}

	if foundID == "" {
		return &CommandResult{
			Text:       fmt.Sprintf("You don't have '%s'.", cmd.Target),
			GameStatus: state.Status,
			TurnNumber: state.TurnNumber,
		}
	}

	itemDef := world.Items[foundID]
	delete(state.Inventory, foundID)
	rs := state.RoomStates[state.CurrentRoom]
	rs.AddedItems[foundID] = true
	delete(rs.RemovedItems, foundID)

	return &CommandResult{
		Text:       fmt.Sprintf("You drop the %s.", itemDef.Name),
		GameStatus: state.Status,
		TurnNumber: state.TurnNumber,
	}
}

func actionUse(state *WorldState, world *WorldDefinition, cmd *ParsedCommand) *CommandResult {
	if cmd.Target == "" {
		return &CommandResult{
			Text:       "Use what?",
			GameStatus: state.Status,
			TurnNumber: state.TurnNumber,
		}
	}

	itemID := resolveItemID(state, world, cmd.Target)
	if itemID == "" {
		return &CommandResult{
			Text:       fmt.Sprintf("You don't see '%s' here.", cmd.Target),
			GameStatus: state.Status,
			TurnNumber: state.TurnNumber,
		}
	}

	itemDef := world.Items[itemID]
	for _, interaction := range itemDef.Interactions {
		if interaction.Verb != "use" {
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

	return &CommandResult{
		Text:       fmt.Sprintf("You can't figure out how to use the %s right now.", itemDef.Name),
		GameStatus: state.Status,
		TurnNumber: state.TurnNumber,
	}
}

func actionInventory(state *WorldState, world *WorldDefinition, cmd *ParsedCommand) *CommandResult {
	if len(state.Inventory) == 0 {
		return &CommandResult{
			Text:       "You are carrying nothing.",
			GameStatus: state.Status,
			TurnNumber: state.TurnNumber,
		}
	}

	var items []string
	for itemID := range state.Inventory {
		if def, ok := world.Items[itemID]; ok {
			items = append(items, "  - "+def.Name)
		}
	}

	return &CommandResult{
		Text:       "You are carrying:\n" + strings.Join(items, "\n"),
		GameStatus: state.Status,
		TurnNumber: state.TurnNumber,
	}
}

func actionHint(state *WorldState, world *WorldDefinition, cmd *ParsedCommand) *CommandResult {
	room := world.Rooms[state.CurrentRoom]
	for _, h := range room.Hints {
		if h.Condition == nil || EvaluateCondition(state, *h.Condition) {
			return &CommandResult{
				Text:       "Hint: " + h.Text,
				GameStatus: state.Status,
				TurnNumber: state.TurnNumber,
			}
		}
	}
	return &CommandResult{
		Text:       "You can't think of anything useful right now. Try looking around.",
		GameStatus: state.Status,
		TurnNumber: state.TurnNumber,
	}
}

func actionHelp(state *WorldState, world *WorldDefinition, cmd *ParsedCommand) *CommandResult {
	return &CommandResult{
		Text: `Available commands:
  look          - Describe your surroundings
  look <item>   - Examine an item
  move <dir>    - Move in a direction (north, south, east, west, up, down)
  take <item>   - Pick up an item
  drop <item>   - Drop an item
  use <item>    - Use an item
  inventory     - List what you're carrying
  hint          - Get a hint for what to do next
  help          - Show this help

Shortcuts: n/s/e/w (directions), i (inventory), l (look)
You can also try: push, pull, turn, open on objects in the room.`,
		GameStatus: state.Status,
		TurnNumber: state.TurnNumber,
	}
}

// describeRoom builds the full room description including conditional text and visible items.
func describeRoom(state *WorldState, world *WorldDefinition) string {
	room := world.Rooms[state.CurrentRoom]
	desc := room.Description

	// Apply conditional descriptions
	for _, cd := range room.ConditionalDescriptions {
		if EvaluateCondition(state, cd.Condition) {
			if cd.Replace {
				desc = cd.Text
			} else {
				desc = desc + "\n" + cd.Text
			}
		}
	}

	// List visible items
	roomItems := GetRoomItems(state, world, state.CurrentRoom)
	if len(roomItems) > 0 {
		var itemNames []string
		for _, itemID := range roomItems {
			if def, ok := world.Items[itemID]; ok {
				itemNames = append(itemNames, def.Name)
			}
		}
		if len(itemNames) > 0 {
			desc += "\n\nYou can see: " + strings.Join(itemNames, ", ") + "."
		}
	}

	// List exits
	connections := GetRoomConnections(state, world, state.CurrentRoom)
	if len(connections) > 0 {
		var dirs []string
		for dir := range connections {
			dirs = append(dirs, dir)
		}
		desc += "\nExits: " + strings.Join(dirs, ", ") + "."
	}

	return desc
}
