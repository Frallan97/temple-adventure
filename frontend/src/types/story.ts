export interface StorySummary {
  id: string;
  name: string;
  slug: string;
  description: string;
  author: string;
  is_published: boolean;
}

export interface StoryListResponse {
  stories: StorySummary[];
}

export interface Story {
  id: string;
  name: string;
  slug: string;
  description: string;
  author: string;
  start_room: string;
  is_published: boolean;
  created_at: string;
  updated_at: string;
}

export interface Condition {
  type: string;
  key: string;
  value: unknown;
  negate: boolean;
}

export interface Effect {
  type: string;
  key: string;
  value: unknown;
}

export interface ConditionalText {
  condition: Condition;
  text: string;
  replace: boolean;
}

export interface ConditionalHint {
  condition: Condition | null;
  text: string;
}

export interface Interaction {
  verb: string;
  conditions: Condition[];
  effects: Effect[];
  response: string;
  fail_response: string;
}

export interface RoomDef {
  id: string;
  name: string;
  description: string;
  connections: Record<string, string>;
  items: string[];
  puzzles: string[];
  conditional_descriptions: ConditionalText[];
  hints: ConditionalHint[];
}

export interface ItemDef {
  id: string;
  name: string;
  aliases: string[];
  description: string;
  portable: boolean;
  interactions: Interaction[];
  conditional_descriptions: ConditionalText[];
}

export interface TimedWindow {
  start_trigger: string;
  turn_limit: number;
}

export interface PuzzleStep {
  step_id: string;
  prompt: string;
  conditions: Condition[];
  effects: Effect[];
}

export interface PuzzleDef {
  id: string;
  name: string;
  description: string;
  steps: PuzzleStep[];
  timed_window: TimedWindow | null;
  failure_effects: Effect[];
  failure_text: string;
  completion_text: string;
}

export interface DialogueLine {
  topic: string;
  conditions: Condition[];
  response: string;
  effects: Effect[];
}

export interface NpcMovement {
  conditions: Condition[];
  target_room: string;
}

export interface NpcDef {
  id: string;
  name: string;
  description: string;
  aliases: string[];
  room: string;
  dialogue: DialogueLine[];
  movement: NpcMovement[];
  conditional_descriptions: ConditionalText[];
}

export interface StoryResponse {
  story: Story;
  rooms: Record<string, RoomDef>;
  items: Record<string, ItemDef>;
  puzzles: Record<string, PuzzleDef>;
  npcs: Record<string, NpcDef>;
}

export interface ValidateResponse {
  valid: boolean;
  errors: { field: string; message: string }[];
}
