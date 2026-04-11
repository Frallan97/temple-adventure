package engine

// --- Content definitions (loaded from YAML, immutable templates) ---

type WorldDefinition struct {
	Rooms   map[string]*RoomDef   `yaml:"rooms" json:"rooms"`
	Items   map[string]*ItemDef   `yaml:"items" json:"items"`
	Puzzles map[string]*PuzzleDef `yaml:"puzzles" json:"puzzles"`
	Npcs    map[string]*NpcDef    `yaml:"npcs" json:"npcs"`
}

type RoomDef struct {
	ID                      string            `yaml:"id" json:"id"`
	Name                    string            `yaml:"name" json:"name"`
	Description             string            `yaml:"description" json:"description"`
	Connections             map[string]string  `yaml:"connections" json:"connections"`
	Items                   []string           `yaml:"items" json:"items"`
	Puzzles                 []string           `yaml:"puzzles" json:"puzzles"`
	ConditionalDescriptions []ConditionalText  `yaml:"conditional_descriptions" json:"conditional_descriptions"`
	Hints                   []ConditionalHint  `yaml:"hints" json:"hints"`
}

type ConditionalHint struct {
	Condition *Condition `yaml:"condition" json:"condition"`
	Text      string     `yaml:"text" json:"text"`
}

type ConditionalText struct {
	Condition Condition `yaml:"condition" json:"condition"`
	Text      string    `yaml:"text" json:"text"`
	Replace   bool      `yaml:"replace" json:"replace"`
}

type ItemDef struct {
	ID                      string            `yaml:"id" json:"id"`
	Name                    string            `yaml:"name" json:"name"`
	Aliases                 []string          `yaml:"aliases" json:"aliases"`
	Description             string            `yaml:"description" json:"description"`
	Portable                bool              `yaml:"portable" json:"portable"`
	Interactions            []Interaction     `yaml:"interactions" json:"interactions"`
	ConditionalDescriptions []ConditionalText `yaml:"conditional_descriptions" json:"conditional_descriptions"`
}

type Interaction struct {
	Verb         string      `yaml:"verb" json:"verb"`
	Conditions   []Condition `yaml:"conditions" json:"conditions"`
	Effects      []Effect    `yaml:"effects" json:"effects"`
	Response     string      `yaml:"response" json:"response"`
	FailResponse string      `yaml:"fail_response" json:"fail_response"`
}

type PuzzleDef struct {
	ID             string       `yaml:"id" json:"id"`
	Name           string       `yaml:"name" json:"name"`
	Description    string       `yaml:"description" json:"description"`
	Steps          []PuzzleStep `yaml:"steps" json:"steps"`
	TimedWindow    *TimedWindow `yaml:"timed_window" json:"timed_window"`
	FailureEffects []Effect     `yaml:"failure_effects" json:"failure_effects"`
	FailureText    string       `yaml:"failure_text" json:"failure_text"`
	CompletionText string       `yaml:"completion_text" json:"completion_text"`
}

type PuzzleStep struct {
	StepID     string      `yaml:"step_id" json:"step_id"`
	Prompt     string      `yaml:"prompt" json:"prompt"`
	Conditions []Condition `yaml:"conditions" json:"conditions"`
	Effects    []Effect    `yaml:"effects" json:"effects"`
}

type TimedWindow struct {
	StartTrigger string `yaml:"start_trigger" json:"start_trigger"`
	TurnLimit    int    `yaml:"turn_limit" json:"turn_limit"`
}

// --- Condition system ---

type Condition struct {
	Type   string      `yaml:"type" json:"type"`
	Key    string      `yaml:"key" json:"key"`
	Value  interface{} `yaml:"value" json:"value"`
	Negate bool        `yaml:"negate" json:"negate"`
}

// --- Effect system ---

type Effect struct {
	Type  string      `yaml:"type" json:"type"`
	Key   string      `yaml:"key" json:"key"`
	Value interface{} `yaml:"value" json:"value"`
}

// --- NPC definitions ---

type NpcDef struct {
	ID                      string            `yaml:"id" json:"id"`
	Name                    string            `yaml:"name" json:"name"`
	Description             string            `yaml:"description" json:"description"`
	Aliases                 []string          `yaml:"aliases" json:"aliases"`
	Room                    string            `yaml:"room" json:"room"`
	Dialogue                []DialogueLine    `yaml:"dialogue" json:"dialogue"`
	Movement                []NpcMovement     `yaml:"movement" json:"movement"`
	ConditionalDescriptions []ConditionalText `yaml:"conditional_descriptions" json:"conditional_descriptions"`
}

type DialogueLine struct {
	Topic      string      `yaml:"topic" json:"topic"`
	Conditions []Condition `yaml:"conditions" json:"conditions"`
	Response   string      `yaml:"response" json:"response"`
	Effects    []Effect    `yaml:"effects" json:"effects"`
}

type NpcMovement struct {
	Conditions []Condition `yaml:"conditions" json:"conditions"`
	TargetRoom string      `yaml:"target_room" json:"target_room"`
}

// --- Runtime state (mutable, per-session) ---

type WorldState struct {
	SessionID   string
	CurrentRoom string
	TurnNumber  int
	Status      string
	Inventory   map[string]bool
	Variables   map[string]Variable
	RoomStates  map[string]*RoomState
	NpcStates   map[string]*NpcState
}

type NpcState struct {
	CurrentRoom string
}

type Variable struct {
	Type    string
	BoolVal bool
	IntVal  int
	StrVal  string
}

type RoomState struct {
	AddedItems         map[string]bool
	RemovedItems       map[string]bool
	BlockedConnections map[string]bool
	AddedConnections   map[string]string
}

// --- Command types ---

type ParsedCommand struct {
	Raw    string
	Verb   string
	Target string
}

type CommandResult struct {
	Text        string `json:"text"`
	RoomChanged bool   `json:"room_changed"`
	GameOver    bool   `json:"game_over"`
	GameStatus  string `json:"game_status"`
	TurnNumber  int    `json:"turn_number"`
}
