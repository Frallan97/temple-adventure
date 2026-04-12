package storygen

// StorySpec is the simplified format for authoring stories.
// High-level puzzle templates replace raw conditions/effects/interactions.
type StorySpec struct {
	Title       string              `json:"title"`
	Slug        string              `json:"slug"`
	Description string              `json:"description"`
	Author      string              `json:"author"`
	StartRoom   string              `json:"start_room"`
	Rooms       map[string]RoomSpec `json:"rooms"`
	Items       map[string]ItemSpec `json:"items"`
	Puzzles     []PuzzleSpec        `json:"puzzles"`
	Npcs        map[string]NpcSpec  `json:"npcs,omitempty"`
}

// RoomSpec defines a room in simplified form.
type RoomSpec struct {
	Name                   string            `json:"name"`
	Description            string            `json:"description"`
	Connections            map[string]string  `json:"connections"`
	Items                  []string           `json:"items"`
	DescriptionAfterPuzzle string            `json:"description_after_puzzle,omitempty"`
}

// ItemSpec defines an item in simplified form.
type ItemSpec struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Aliases     []string `json:"aliases"`
	Portable    bool     `json:"portable"`
	ExamineText string   `json:"examine_text,omitempty"`
}

// PuzzleSpec uses a discriminated union via the Type field.
// Only fields relevant to the chosen Type need to be populated.
type PuzzleSpec struct {
	ID             string `json:"id"`
	Type           string `json:"type"` // "key_lock", "examine_learn", "fetch_quest", "timed_challenge", "win_condition", "combination_lock", "item_combine", "counter_puzzle"
	Name           string `json:"name"`
	Description    string `json:"description"`
	Room           string `json:"room"`
	CompletionText string `json:"completion_text,omitempty"`

	// key_lock: use KeyItem on LockTarget to unlock a direction
	KeyItem         string `json:"key_item,omitempty"`
	LockTarget      string `json:"lock_target,omitempty"`
	UnlockDirection string `json:"unlock_direction,omitempty"`
	UnlockRoom      string `json:"unlock_room,omitempty"`
	LockVerb        string `json:"lock_verb,omitempty"`
	LockFailText    string `json:"lock_fail_text,omitempty"`

	// examine_learn: examine SourceItem to learn clue, then interact with TargetItem
	SourceItem        string `json:"source_item,omitempty"`
	SourceLearnText   string `json:"source_learn_text,omitempty"`
	TargetItem        string `json:"target_item,omitempty"`
	TargetVerb        string `json:"target_verb,omitempty"`
	TargetSuccessText string `json:"target_success_text,omitempty"`
	TargetFailText    string `json:"target_fail_text,omitempty"`

	// fetch_quest: bring FetchItem to FetchRoom, use on FetchTarget
	FetchItem        string `json:"fetch_item,omitempty"`
	FetchRoom        string `json:"fetch_room,omitempty"`
	FetchTarget      string `json:"fetch_target,omitempty"`
	FetchVerb        string `json:"fetch_verb,omitempty"`
	FetchSuccessText string `json:"fetch_success_text,omitempty"`
	FetchConsumeItem bool   `json:"fetch_consume_item,omitempty"`

	// timed_challenge: wraps another mechanic with a turn limit
	TriggerItem    string              `json:"trigger_item,omitempty"`
	TriggerVerb    string              `json:"trigger_verb,omitempty"`
	TriggerText    string              `json:"trigger_text,omitempty"`
	TurnLimit      int                 `json:"turn_limit,omitempty"`
	FailureText    string              `json:"failure_text,omitempty"`
	FailureEffects []FailureEffectSpec `json:"failure_effects,omitempty"`

	// win_condition: taking/using WinItem ends the game
	WinItem     string       `json:"win_item,omitempty"`
	WinVerb     string       `json:"win_verb,omitempty"`
	WinText     string       `json:"win_text,omitempty"`      // simple single ending text
	Endings     []EndingSpec `json:"endings,omitempty"`        // conditional multiple endings (overrides WinText)
	EndingID    string       `json:"ending_id,omitempty"`      // identifier for this ending (when using multiple win_conditions)
	EndingTitle string       `json:"ending_title,omitempty"`   // display title for this ending

	// combination_lock: interact with CombinationTarget N times to solve
	CombinationTarget string   `json:"combination_target,omitempty"`
	CombinationVerb   string   `json:"combination_verb,omitempty"`
	CombinationSteps  int      `json:"combination_steps,omitempty"`
	CombinationTexts  []string `json:"combination_texts,omitempty"`

	// item_combine: combine two inventory items into a new item
	CombineItemA    string `json:"combine_item_a,omitempty"`
	CombineItemB    string `json:"combine_item_b,omitempty"`
	CombineResult   string `json:"combine_result,omitempty"`
	CombineVerb     string `json:"combine_verb,omitempty"`
	CombineConsumeA bool   `json:"combine_consume_a,omitempty"`
	CombineConsumeB bool   `json:"combine_consume_b,omitempty"`
	CombineText     string `json:"combine_text,omitempty"`
	CombineFailText string `json:"combine_fail_text,omitempty"`

	// counter_puzzle: accumulate interactions across multiple items
	CounterItems        []string          `json:"counter_items,omitempty"`
	CounterVerb         string            `json:"counter_verb,omitempty"`
	CounterTarget       int               `json:"counter_target,omitempty"`
	CounterItemTexts    map[string]string `json:"counter_item_texts,omitempty"`
	CounterDefaultText  string            `json:"counter_default_text,omitempty"`
	CounterConsumeItems bool              `json:"counter_consume_items,omitempty"`
}

// EndingSpec defines a conditional ending for a win_condition puzzle.
type EndingSpec struct {
	ID         string            `json:"id"`
	Title      string            `json:"title"`
	Conditions map[string]string `json:"conditions,omitempty"` // variable conditions: key → expected value
	Text       string            `json:"text"`
}

// FailureEffectSpec is a simplified failure effect for timed challenges.
type FailureEffectSpec struct {
	Type      string `json:"type"`                // "move_player" or "lock_connection"
	Room      string `json:"room,omitempty"`       // target room for move_player
	Direction string `json:"direction,omitempty"`  // direction for lock_connection
}

// NpcSpec defines an NPC in simplified form.
type NpcSpec struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Aliases     []string            `json:"aliases"`
	Room        string              `json:"room"`
	Greeting    string              `json:"greeting,omitempty"`
	Topics      map[string]string   `json:"topics,omitempty"`
	Dialogue    []DialogueNodeSpec  `json:"dialogue,omitempty"` // full dialogue tree (overrides Greeting if present)
}

// DialogueNodeSpec defines a node in a dialogue tree.
type DialogueNodeSpec struct {
	NodeID  string               `json:"node_id"`
	Text    string               `json:"text"`
	Topic   string               `json:"topic,omitempty"`
	Choices []DialogueChoiceSpec `json:"choices,omitempty"`
}

// DialogueChoiceSpec defines a player choice in a dialogue node.
type DialogueChoiceSpec struct {
	Text     string `json:"text"`
	NextNode string `json:"next_node"`
	SetVar   string `json:"set_var,omitempty"`   // shorthand: "key=value" → set_var effect
	GiveItem string `json:"give_item,omitempty"` // shorthand: add_item effect
	NeedItem string `json:"need_item,omitempty"` // shorthand: has_item condition
}

// Reverse direction mapping for bidirectional connections.
var reverseDirection = map[string]string{
	"north": "south",
	"south": "north",
	"east":  "west",
	"west":  "east",
	"up":    "down",
	"down":  "up",
}
