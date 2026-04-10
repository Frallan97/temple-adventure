package engine

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type roomFile struct {
	ID                      string              `yaml:"id"`
	Name                    string              `yaml:"name"`
	Description             string              `yaml:"description"`
	Connections             map[string]string    `yaml:"connections"`
	Items                   []string             `yaml:"items"`
	Puzzles                 []string             `yaml:"puzzles"`
	ConditionalDescriptions []ConditionalText    `yaml:"conditional_descriptions"`
	Hints                   []ConditionalHint    `yaml:"hints"`
}

type itemsFile struct {
	Items map[string]*ItemDef `yaml:"items"`
}

type puzzleFile struct {
	ID             string       `yaml:"id"`
	Name           string       `yaml:"name"`
	Description    string       `yaml:"description"`
	Steps          []PuzzleStep `yaml:"steps"`
	TimedWindow    *TimedWindow `yaml:"timed_window"`
	FailureEffects []Effect     `yaml:"failure_effects"`
	FailureText    string       `yaml:"failure_text"`
	CompletionText string       `yaml:"completion_text"`
}

func LoadWorldDefinition(contentDir string) (*WorldDefinition, error) {
	world := &WorldDefinition{
		Rooms:   make(map[string]*RoomDef),
		Items:   make(map[string]*ItemDef),
		Puzzles: make(map[string]*PuzzleDef),
	}

	if err := loadRooms(filepath.Join(contentDir, "rooms"), world); err != nil {
		return nil, fmt.Errorf("loading rooms: %w", err)
	}

	if err := loadItems(filepath.Join(contentDir, "items"), world); err != nil {
		return nil, fmt.Errorf("loading items: %w", err)
	}

	if err := loadPuzzles(filepath.Join(contentDir, "puzzles"), world); err != nil {
		return nil, fmt.Errorf("loading puzzles: %w", err)
	}

	if err := world.Validate(); err != nil {
		return nil, fmt.Errorf("validating world: %w", err)
	}

	return world, nil
}

func loadRooms(dir string, world *WorldDefinition) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return fmt.Errorf("globbing room files: %w", err)
	}

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("reading %s: %w", f, err)
		}

		var rf roomFile
		if err := yaml.Unmarshal(data, &rf); err != nil {
			return fmt.Errorf("parsing %s: %w", f, err)
		}

		world.Rooms[rf.ID] = &RoomDef{
			ID:                      rf.ID,
			Name:                    rf.Name,
			Description:             rf.Description,
			Connections:             rf.Connections,
			Items:                   rf.Items,
			Puzzles:                 rf.Puzzles,
			ConditionalDescriptions: rf.ConditionalDescriptions,
			Hints:                   rf.Hints,
		}
	}

	return nil
}

func loadItems(dir string, world *WorldDefinition) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return fmt.Errorf("globbing item files: %w", err)
	}

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("reading %s: %w", f, err)
		}

		var iff itemsFile
		if err := yaml.Unmarshal(data, &iff); err != nil {
			return fmt.Errorf("parsing %s: %w", f, err)
		}

		for id, item := range iff.Items {
			item.ID = id
			world.Items[id] = item
		}
	}

	return nil
}

func loadPuzzles(dir string, world *WorldDefinition) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return fmt.Errorf("globbing puzzle files: %w", err)
	}

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("reading %s: %w", f, err)
		}

		var pf puzzleFile
		if err := yaml.Unmarshal(data, &pf); err != nil {
			return fmt.Errorf("parsing %s: %w", f, err)
		}

		world.Puzzles[pf.ID] = &PuzzleDef{
			ID:             pf.ID,
			Name:           pf.Name,
			Description:    pf.Description,
			Steps:          pf.Steps,
			TimedWindow:    pf.TimedWindow,
			FailureEffects: pf.FailureEffects,
			FailureText:    pf.FailureText,
			CompletionText: pf.CompletionText,
		}
	}

	return nil
}

func (wd *WorldDefinition) Validate() error {
	return wd.ValidateWithStartRoom("entrance")
}

func (wd *WorldDefinition) ValidateWithStartRoom(startRoom string) error {
	if len(wd.Rooms) == 0 {
		return fmt.Errorf("no rooms defined")
	}

	if _, ok := wd.Rooms[startRoom]; !ok {
		return fmt.Errorf("start room %q not found", startRoom)
	}

	for id, room := range wd.Rooms {
		for dir, targetID := range room.Connections {
			if _, ok := wd.Rooms[targetID]; !ok {
				return fmt.Errorf("room %q connection %q references unknown room %q", id, dir, targetID)
			}
		}
		for _, itemID := range room.Items {
			if _, ok := wd.Items[itemID]; !ok {
				return fmt.Errorf("room %q references unknown item %q", id, itemID)
			}
		}
		for _, puzzleID := range room.Puzzles {
			if _, ok := wd.Puzzles[puzzleID]; !ok {
				return fmt.Errorf("room %q references unknown puzzle %q", id, puzzleID)
			}
		}
	}

	return nil
}

func (wd *WorldDefinition) NewWorldState(sessionID string, startRoom ...string) *WorldState {
	room := "entrance"
	if len(startRoom) > 0 && startRoom[0] != "" {
		room = startRoom[0]
	}
	state := &WorldState{
		SessionID:   sessionID,
		CurrentRoom: room,
		TurnNumber:  0,
		Status:      "active",
		Inventory:   make(map[string]bool),
		Variables:   make(map[string]Variable),
		RoomStates:  make(map[string]*RoomState),
	}

	for roomID := range wd.Rooms {
		state.RoomStates[roomID] = &RoomState{
			AddedItems:         make(map[string]bool),
			RemovedItems:       make(map[string]bool),
			BlockedConnections: make(map[string]bool),
			AddedConnections:   make(map[string]string),
		}
	}

	return state
}
