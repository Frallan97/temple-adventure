package engine

import "fmt"

func EvaluateCondition(state *WorldState, cond Condition) bool {
	result := evaluateConditionInner(state, cond)
	if cond.Negate {
		return !result
	}
	return result
}

func evaluateConditionInner(state *WorldState, cond Condition) bool {
	switch cond.Type {
	case "has_item":
		return state.Inventory[cond.Key]

	case "var_equals":
		v, ok := state.Variables[cond.Key]
		if !ok {
			return false
		}
		return variableEquals(v, cond.Value)

	case "var_gte":
		v, ok := state.Variables[cond.Key]
		if !ok {
			return false
		}
		return v.IntVal >= toInt(cond.Value)

	case "var_lte":
		v, ok := state.Variables[cond.Key]
		if !ok {
			return false
		}
		return v.IntVal <= toInt(cond.Value)

	case "in_room":
		return state.CurrentRoom == fmt.Sprintf("%v", cond.Key)

	case "item_in_room":
		return IsItemInRoom(state, cond.Key, state.CurrentRoom)

	case "puzzle_complete":
		v, ok := state.Variables["puzzle."+cond.Key+".complete"]
		return ok && v.BoolVal

	default:
		return false
	}
}

func EvaluateConditions(state *WorldState, conds []Condition) bool {
	for _, c := range conds {
		if !EvaluateCondition(state, c) {
			return false
		}
	}
	return true
}

func ApplyEffect(state *WorldState, effect Effect) {
	switch effect.Type {
	case "set_var":
		// Handle special placeholder for current turn
		if str, ok := effect.Value.(string); ok && str == "__CURRENT_TURN__" {
			state.Variables[effect.Key] = Variable{Type: "int", IntVal: state.TurnNumber}
		} else {
			state.Variables[effect.Key] = toVariable(effect.Value)
		}

	case "increment_var":
		v := state.Variables[effect.Key]
		v.Type = "int"
		v.IntVal += toInt(effect.Value)
		state.Variables[effect.Key] = v

	case "add_item":
		state.Inventory[effect.Key] = true

	case "remove_item":
		delete(state.Inventory, effect.Key)

	case "add_room_item":
		roomID := fmt.Sprintf("%v", effect.Value)
		if rs, ok := state.RoomStates[roomID]; ok {
			rs.AddedItems[effect.Key] = true
			delete(rs.RemovedItems, effect.Key)
		}

	case "remove_room_item":
		roomID := state.CurrentRoom
		if effect.Value != nil {
			roomID = fmt.Sprintf("%v", effect.Value)
		}
		if rs, ok := state.RoomStates[roomID]; ok {
			rs.RemovedItems[effect.Key] = true
			delete(rs.AddedItems, effect.Key)
		}

	case "move_player":
		state.CurrentRoom = fmt.Sprintf("%v", effect.Value)

	case "unlock_connection":
		roomAndDir := effect.Key
		// Key format: "room_id.direction"
		roomID, dir := splitRoomDir(roomAndDir)
		if rs, ok := state.RoomStates[roomID]; ok {
			delete(rs.BlockedConnections, dir)
			if targetRoom, ok := effect.Value.(string); ok && targetRoom != "" {
				rs.AddedConnections[dir] = targetRoom
			}
		}

	case "lock_connection":
		roomID, dir := splitRoomDir(effect.Key)
		if rs, ok := state.RoomStates[roomID]; ok {
			rs.BlockedConnections[dir] = true
		}

	case "set_status":
		state.Status = fmt.Sprintf("%v", effect.Value)
	}
}

func ApplyEffects(state *WorldState, effects []Effect) {
	for _, e := range effects {
		ApplyEffect(state, e)
	}
}

// IsItemInRoom checks if an item is present in a room (considering mutations).
func IsItemInRoom(state *WorldState, itemID, roomID string) bool {
	rs := state.RoomStates[roomID]
	if rs == nil {
		return false
	}

	if rs.RemovedItems[itemID] {
		return false
	}
	if rs.AddedItems[itemID] {
		return true
	}
	return false
}

// IsItemInRoomOrDefault checks base room items + mutations.
func IsItemInRoomOrDefault(state *WorldState, world *WorldDefinition, itemID, roomID string) bool {
	rs := state.RoomStates[roomID]

	if rs != nil && rs.RemovedItems[itemID] {
		return false
	}
	if rs != nil && rs.AddedItems[itemID] {
		return true
	}

	room, ok := world.Rooms[roomID]
	if !ok {
		return false
	}
	for _, id := range room.Items {
		if id == itemID {
			return true
		}
	}
	return false
}

// GetRoomItems returns all items currently in a room.
func GetRoomItems(state *WorldState, world *WorldDefinition, roomID string) []string {
	room, ok := world.Rooms[roomID]
	if !ok {
		return nil
	}

	rs := state.RoomStates[roomID]
	items := make([]string, 0)

	for _, id := range room.Items {
		if rs != nil && rs.RemovedItems[id] {
			continue
		}
		items = append(items, id)
	}

	if rs != nil {
		for id := range rs.AddedItems {
			items = append(items, id)
		}
	}

	return items
}

// GetRoomConnections returns available connections from a room.
func GetRoomConnections(state *WorldState, world *WorldDefinition, roomID string) map[string]string {
	room, ok := world.Rooms[roomID]
	if !ok {
		return nil
	}

	rs := state.RoomStates[roomID]
	connections := make(map[string]string)

	for dir, target := range room.Connections {
		if rs != nil && rs.BlockedConnections[dir] {
			continue
		}
		connections[dir] = target
	}

	if rs != nil {
		for dir, target := range rs.AddedConnections {
			if !rs.BlockedConnections[dir] {
				connections[dir] = target
			}
		}
	}

	return connections
}

// --- Helpers ---

func variableEquals(v Variable, value interface{}) bool {
	switch v.Type {
	case "bool":
		return v.BoolVal == toBool(value)
	case "int":
		return v.IntVal == toInt(value)
	case "string":
		return v.StrVal == fmt.Sprintf("%v", value)
	}
	return false
}

func toVariable(value interface{}) Variable {
	switch v := value.(type) {
	case bool:
		return Variable{Type: "bool", BoolVal: v}
	case int:
		return Variable{Type: "int", IntVal: v}
	case float64:
		return Variable{Type: "int", IntVal: int(v)}
	case string:
		return Variable{Type: "string", StrVal: v}
	default:
		return Variable{Type: "string", StrVal: fmt.Sprintf("%v", v)}
	}
}

func toInt(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case string:
		return 0
	default:
		return 0
	}
}

func toBool(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return v
	default:
		return false
	}
}

func splitRoomDir(key string) (string, string) {
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == '.' {
			return key[:i], key[i+1:]
		}
	}
	return key, ""
}
