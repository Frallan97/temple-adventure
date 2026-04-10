package engine

import "fmt"

type PuzzleSystem struct {
	world *WorldDefinition
}

func NewPuzzleSystem(world *WorldDefinition) *PuzzleSystem {
	return &PuzzleSystem{world: world}
}

// CheckTimedWindows checks if any active timed puzzle has expired.
// Returns narrative text if a puzzle failed, empty string otherwise.
func (ps *PuzzleSystem) CheckTimedWindows(state *WorldState) string {
	for _, puzzle := range ps.world.Puzzles {
		if puzzle.TimedWindow == nil {
			continue
		}

		// Check if puzzle is already complete or failed
		if ps.IsPuzzleComplete(state, puzzle.ID) || ps.IsPuzzleFailed(state, puzzle.ID) {
			continue
		}

		// Check if the timed window has started
		startedVar, started := state.Variables[puzzle.TimedWindow.StartTrigger]
		if !started || !startedVar.BoolVal {
			continue
		}

		// Get the turn the timer started
		startTurnVar, ok := state.Variables["puzzle."+puzzle.ID+".start_turn"]
		if !ok {
			continue
		}

		elapsed := state.TurnNumber - startTurnVar.IntVal
		if elapsed > puzzle.TimedWindow.TurnLimit {
			// Timer expired — apply failure effects permanently
			ApplyEffects(state, puzzle.FailureEffects)
			state.Variables["puzzle."+puzzle.ID+".failed"] = Variable{Type: "bool", BoolVal: true}
			return puzzle.FailureText
		}
	}

	return ""
}

// CheckPuzzleProgress evaluates all puzzles in the current room.
// Returns narrative text if a puzzle step was completed or the puzzle was solved.
func (ps *PuzzleSystem) CheckPuzzleProgress(state *WorldState) string {
	room, ok := ps.world.Rooms[state.CurrentRoom]
	if !ok {
		return ""
	}

	var result string

	for _, puzzleID := range room.Puzzles {
		puzzle, ok := ps.world.Puzzles[puzzleID]
		if !ok {
			continue
		}

		if ps.IsPuzzleComplete(state, puzzleID) || ps.IsPuzzleFailed(state, puzzleID) {
			continue
		}

		currentStep := ps.getCurrentStep(state, puzzleID)
		if currentStep >= len(puzzle.Steps) {
			continue
		}

		step := puzzle.Steps[currentStep]
		if EvaluateConditions(state, step.Conditions) {
			// Step completed
			ApplyEffects(state, step.Effects)

			nextStep := currentStep + 1
			state.Variables[fmt.Sprintf("puzzle.%s.step", puzzleID)] = Variable{Type: "int", IntVal: nextStep}

			if nextStep >= len(puzzle.Steps) {
				// Puzzle fully solved
				state.Variables["puzzle."+puzzleID+".complete"] = Variable{Type: "bool", BoolVal: true}
				result += puzzle.CompletionText
			}
		}
	}

	return result
}

func (ps *PuzzleSystem) IsPuzzleComplete(state *WorldState, puzzleID string) bool {
	v, ok := state.Variables["puzzle."+puzzleID+".complete"]
	return ok && v.BoolVal
}

func (ps *PuzzleSystem) IsPuzzleFailed(state *WorldState, puzzleID string) bool {
	v, ok := state.Variables["puzzle."+puzzleID+".failed"]
	return ok && v.BoolVal
}

func (ps *PuzzleSystem) getCurrentStep(state *WorldState, puzzleID string) int {
	v, ok := state.Variables[fmt.Sprintf("puzzle.%s.step", puzzleID)]
	if !ok {
		return 0
	}
	return v.IntVal
}
