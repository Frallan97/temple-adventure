package engine

import (
	"slices"
	"testing"
)

func TestCommandContextTracksVisibleEntities(t *testing.T) {
	world := &WorldDefinition{
		Rooms: map[string]*RoomDef{"room": {ID: "room", Items: []string{"key"}, Connections: map[string]string{"north": "other"}}, "other": {ID: "other"}},
		Items: map[string]*ItemDef{"key": {ID: "key", Name: "Brass key", Portable: true, Aliases: []string{"key"}, Interactions: []Interaction{{Verb: "polish"}}}, "secret": {ID: "secret", Name: "Secret gem"}},
		Npcs:  map[string]*NpcDef{"guard": {ID: "guard", Name: "Guard", Room: "room"}, "hidden": {ID: "hidden", Name: "Hidden NPC", Room: "other"}},
	}
	eng := NewEngineFromWorld(world)
	state := world.NewWorldState("test", "room")
	ctx := eng.GetCommandContext(state)
	if len(ctx.Objects) != 1 || len(ctx.Npcs) != 1 {
		t.Fatalf("revealed hidden entities: %+v", ctx)
	}
	for _, command := range []string{"take Brass key", "polish Brass key", "talk Guard", "move north"} {
		if !slices.Contains(ctx.Commands, command) {
			t.Errorf("missing %q", command)
		}
	}
	state.Inventory["key"] = true
	state.RoomStates["room"].RemovedItems["key"] = true
	state.RoomStates["room"].BlockedConnections["north"] = true
	state.NpcStates["guard"].CurrentRoom = "other"
	ctx = eng.GetCommandContext(state)
	if !slices.Contains(ctx.Commands, "drop Brass key") || slices.Contains(ctx.Commands, "take Brass key") || slices.Contains(ctx.Commands, "move north") || len(ctx.Npcs) != 0 {
		t.Fatalf("stale context: %+v", ctx)
	}
}
