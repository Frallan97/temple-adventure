package engine

import "strings"

var verbAliases = map[string]string{
	"l":       "look",
	"examine": "look",
	"inspect": "look",
	"i":       "inventory",
	"get":     "take",
	"grab":    "take",
	"go":      "move",
	"walk":    "move",
	"n":       "move north",
	"s":       "move south",
	"e":       "move east",
	"w":       "move west",
	"u":       "move up",
	"d":       "move down",
	"north":   "move north",
	"south":   "move south",
	"east":    "move east",
	"west":    "move west",
	"up":      "move up",
	"down":    "move down",
	"?":       "help",
	"h":       "hint",
	"clue":    "hint",
}

type CommandParser struct{}

func NewCommandParser() *CommandParser {
	return &CommandParser{}
}

func (p *CommandParser) Parse(rawInput string) *ParsedCommand {
	input := strings.TrimSpace(strings.ToLower(rawInput))
	if input == "" {
		return &ParsedCommand{Raw: rawInput, Verb: "", Target: ""}
	}

	parts := strings.SplitN(input, " ", 2)
	verb := parts[0]
	target := ""
	if len(parts) > 1 {
		target = strings.TrimSpace(parts[1])
	}

	// Check if the full input matches an alias (e.g., "n" -> "move north")
	if alias, ok := verbAliases[input]; ok {
		aliasParts := strings.SplitN(alias, " ", 2)
		verb = aliasParts[0]
		if len(aliasParts) > 1 {
			target = aliasParts[1]
		}
		return &ParsedCommand{Raw: rawInput, Verb: verb, Target: target}
	}

	// Check if just the verb matches an alias (e.g., "go north" -> "move north")
	if alias, ok := verbAliases[verb]; ok {
		aliasParts := strings.SplitN(alias, " ", 2)
		verb = aliasParts[0]
		if len(aliasParts) > 1 && target == "" {
			target = aliasParts[1]
		}
	}

	return &ParsedCommand{Raw: rawInput, Verb: verb, Target: target}
}
