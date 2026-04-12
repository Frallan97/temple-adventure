export interface CreateGameResponse {
  id: string;
  story_id: string;
  room_name: string;
  description: string;
  turn_number: number;
  inventory: ItemInfo[];
}

export interface ChoiceOption {
  index: number;
  text: string;
}

export interface CommandResponse {
  text: string;
  room_name: string;
  room_changed: boolean;
  turn_number: number;
  game_over: boolean;
  game_status: string;
  inventory: ItemInfo[];
  choices?: ChoiceOption[];
  ending_id?: string;
  ending_title?: string;
}

export interface ItemInfo {
  id: string;
  name: string;
  description: string;
}

export interface GameState {
  id: string;
  room_name: string;
  description: string;
  turn_number: number;
  status: string;
  inventory: ItemInfo[];
}

export interface OutputEntry {
  type: "command" | "narrative" | "system" | "error";
  text: string;
}
