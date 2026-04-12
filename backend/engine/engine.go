package engine

import "log"

type Engine struct {
	World    *WorldDefinition
	parser   *CommandParser
	registry *CommandRegistry
	puzzles  *PuzzleSystem
}

func NewEngine(contentDir string) (*Engine, error) {
	world, err := LoadWorldDefinition(contentDir)
	if err != nil {
		return nil, err
	}
	return NewEngineFromWorld(world), nil
}

func NewEngineFromWorld(world *WorldDefinition) *Engine {
	registry := NewCommandRegistry()
	RegisterBuiltinActions(registry)

	return &Engine{
		World:    world,
		parser:   NewCommandParser(),
		registry: registry,
		puzzles:  NewPuzzleSystem(world),
	}
}

func (e *Engine) ProcessCommand(state *WorldState, rawInput string) *CommandResult {
	if state.Status != "active" {
		result := &CommandResult{
			Text:       "The game is over.",
			GameOver:   true,
			GameStatus: state.Status,
			TurnNumber: state.TurnNumber,
		}
		if v, ok := state.Variables["__ending_id__"]; ok {
			result.EndingID = v.StrVal
		}
		if v, ok := state.Variables["__ending_title__"]; ok {
			result.EndingTitle = v.StrVal
		}
		return result
	}

	cmd := e.parser.Parse(rawInput)
	if cmd.Verb == "" {
		return &CommandResult{
			Text:       "What would you like to do? Type 'help' for available commands.",
			GameStatus: state.Status,
			TurnNumber: state.TurnNumber,
		}
	}

	// Increment turn counter
	state.TurnNumber++

	// Check timed puzzle windows before processing command
	timedText := e.puzzles.CheckTimedWindows(state)
	if timedText != "" {
		// A timed puzzle failed — return failure text, skip the player's command
		return &CommandResult{
			Text:       timedText,
			RoomChanged: state.CurrentRoom != state.CurrentRoom, // will be false, but included for consistency
			GameOver:   state.Status != "active",
			GameStatus: state.Status,
			TurnNumber: state.TurnNumber,
		}
	}

	// Execute the command
	result := e.registry.Execute(state, e.World, cmd)

	// Evaluate NPC movement rules
	for _, npc := range e.World.Npcs {
		ns := state.NpcStates[npc.ID]
		if ns == nil {
			continue
		}
		for _, mv := range npc.Movement {
			if EvaluateConditions(state, mv.Conditions) {
				ns.CurrentRoom = mv.TargetRoom
				break
			}
		}
	}

	// Check puzzle progress after the command
	puzzleText := e.puzzles.CheckPuzzleProgress(state)
	if puzzleText != "" {
		result.Text += "\n\n" + puzzleText
	}

	// Update game over status
	result.GameOver = state.Status != "active"
	result.GameStatus = state.Status
	result.TurnNumber = state.TurnNumber

	// Propagate ending metadata if game is over
	if result.GameOver {
		if v, ok := state.Variables["__ending_id__"]; ok {
			result.EndingID = v.StrVal
		}
		if v, ok := state.Variables["__ending_title__"]; ok {
			result.EndingTitle = v.StrVal
		}
	}

	log.Printf("[Turn %d] %s -> %s (room: %s)", state.TurnNumber, rawInput, cmd.Verb, state.CurrentRoom)

	return result
}

func (e *Engine) ParseCommand(rawInput string) *ParsedCommand {
	return e.parser.Parse(rawInput)
}

func (e *Engine) GetRoomDescription(state *WorldState) string {
	return describeRoom(state, e.World)
}

type ItemInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (e *Engine) GetInventory(state *WorldState) []ItemInfo {
	items := make([]ItemInfo, 0, len(state.Inventory))
	for itemID := range state.Inventory {
		if def, ok := e.World.Items[itemID]; ok {
			items = append(items, ItemInfo{
				ID:          itemID,
				Name:        def.Name,
				Description: def.Description,
			})
		}
	}
	return items
}
