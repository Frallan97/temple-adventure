export interface CommandEntity { name: string; aliases: string[] }
export interface CommandContext { actions: string[]; objects: CommandEntity[]; npcs: CommandEntity[]; commands: string[] }

export interface CreateGameResponse {
  command_context?: CommandContext;
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
  command_context?: CommandContext;
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
  command_context?: CommandContext;
  id: string;
  room_name: string;
  description: string;
  turn_number: number;
  status: string;
  inventory: ItemInfo[];
}

export interface OutputEntry {
  context?: CommandContext;
  type: "command" | "narrative" | "system" | "error";
  text: string;
}

export interface CommandEntry {
  id: string;
  session_id: string;
  turn_number: number;
  raw_input: string;
  parsed_verb: string;
  parsed_target: string;
  room_id: string;
  response_text: string;
  created_at: string;
}

export interface HistoryResponse {
  commands: CommandEntry[];
  total: number;
}

export interface GameLogSummary {
  id: string;
  story_id: string;
  story_name: string;
  status: string;
  turn_number: number;
  created_at: string;
  updated_at: string;
}

export interface GameLogsResponse {
  games: GameLogSummary[];
}
