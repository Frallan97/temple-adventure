package engine

import "sort"

// CommandContext exposes only entities the player can currently interact with.
type CommandContext struct {
	Actions  []string        `json:"actions"`
	Objects  []CommandEntity `json:"objects"`
	Npcs     []CommandEntity `json:"npcs"`
	Commands []string        `json:"commands"`
}
type CommandEntity struct {
	Name    string   `json:"name"`
	Aliases []string `json:"aliases"`
}

func (e *Engine) GetCommandContext(state *WorldState) CommandContext {
	ctx := CommandContext{Objects: []CommandEntity{}, Npcs: []CommandEntity{}}
	actions := map[string]bool{}
	commands := map[string]bool{}
	for verb := range e.registry.actions {
		actions[verb] = true
	}
	for alias := range verbAliases {
		actions[alias] = true
	}
	for _, verb := range []string{"look", "inventory", "help", "hint"} {
		commands[verb] = true
	}
	for dir := range GetRoomConnections(state, e.World, state.CurrentRoom) {
		commands["move "+dir] = true
	}
	roomItems := map[string]bool{}
	for _, id := range GetRoomItems(state, e.World, state.CurrentRoom) {
		roomItems[id] = true
	}
	for id, item := range e.World.Items {
		if !roomItems[id] && !state.Inventory[id] {
			continue
		}
		ctx.Objects = append(ctx.Objects, CommandEntity{Name: item.Name, Aliases: append([]string{id}, item.Aliases...)})
		commands["look "+item.Name] = true
		commands["use "+item.Name] = true
		if state.Inventory[id] {
			commands["drop "+item.Name] = true
		} else if item.Portable {
			commands["take "+item.Name] = true
		}
		for _, interaction := range item.Interactions {
			actions[interaction.Verb] = true
			commands[interaction.Verb+" "+item.Name] = true
		}
	}
	for _, id := range GetRoomNpcs(state, e.World, state.CurrentRoom) {
		npc := e.World.Npcs[id]
		ctx.Npcs = append(ctx.Npcs, CommandEntity{Name: npc.Name, Aliases: append([]string{id}, npc.Aliases...)})
		commands["look "+npc.Name] = true
		commands["talk "+npc.Name] = true
	}
	for action := range actions {
		ctx.Actions = append(ctx.Actions, action)
	}
	for command := range commands {
		ctx.Commands = append(ctx.Commands, command)
	}
	sort.Strings(ctx.Actions)
	sort.Strings(ctx.Commands)
	sort.Slice(ctx.Objects, func(i, j int) bool { return ctx.Objects[i].Name < ctx.Objects[j].Name })
	sort.Slice(ctx.Npcs, func(i, j int) bool { return ctx.Npcs[i].Name < ctx.Npcs[j].Name })
	return ctx
}
