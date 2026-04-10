import type { CreateGameResponse, CommandResponse, GameState } from "../types/game";
import type {
  StoryListResponse,
  StoryResponse,
  Story,
  ValidateResponse,
  RoomDef,
  ItemDef,
  PuzzleDef,
} from "../types/story";

const API_BASE = import.meta.env.VITE_API_URL || "http://127.0.0.1:8080/api/v1";

async function apiRequest<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${endpoint}`, {
    headers: { "Content-Type": "application/json", ...options?.headers },
    ...options,
  });
  if (!response.ok) {
    const error = await response.text();
    throw new Error(`API Error: ${response.status} - ${error}`);
  }
  return response.json();
}

export const gameApi = {
  create: (storyId: string) =>
    apiRequest<CreateGameResponse>("/games", {
      method: "POST",
      body: JSON.stringify({ story_id: storyId }),
    }),

  getState: (id: string) => apiRequest<GameState>(`/games/${id}`),

  sendCommand: (id: string, input: string) =>
    apiRequest<CommandResponse>(`/games/${id}/command`, {
      method: "POST",
      body: JSON.stringify({ input }),
    }),
};

export const storyApi = {
  list: () => apiRequest<StoryListResponse>("/stories"),
  listAll: () => apiRequest<StoryListResponse>("/stories?all=true"),
  get: (id: string) => apiRequest<StoryResponse>(`/stories/${id}/`),
  create: (data: { name: string; slug: string; description: string; author: string; start_room: string }) =>
    apiRequest<Story>("/stories", { method: "POST", body: JSON.stringify(data) }),
  update: (id: string, data: Record<string, string>) =>
    apiRequest<Story>(`/stories/${id}/`, { method: "PUT", body: JSON.stringify(data) }),
  delete: (id: string) =>
    apiRequest<void>(`/stories/${id}/`, { method: "DELETE" }),
  validate: (id: string) =>
    apiRequest<ValidateResponse>(`/stories/${id}/validate`, { method: "POST" }),
  publish: (id: string) =>
    apiRequest<void>(`/stories/${id}/publish`, { method: "POST" }),

  // Rooms
  upsertRoom: (storyId: string, roomId: string, data: Omit<RoomDef, "id">) =>
    apiRequest<void>(`/stories/${storyId}/rooms/${roomId}`, { method: "PUT", body: JSON.stringify(data) }),
  deleteRoom: (storyId: string, roomId: string) =>
    apiRequest<void>(`/stories/${storyId}/rooms/${roomId}`, { method: "DELETE" }),

  // Items
  upsertItem: (storyId: string, itemId: string, data: Omit<ItemDef, "id">) =>
    apiRequest<void>(`/stories/${storyId}/items/${itemId}`, { method: "PUT", body: JSON.stringify(data) }),
  deleteItem: (storyId: string, itemId: string) =>
    apiRequest<void>(`/stories/${storyId}/items/${itemId}`, { method: "DELETE" }),

  // Puzzles
  upsertPuzzle: (storyId: string, puzzleId: string, data: Omit<PuzzleDef, "id">) =>
    apiRequest<void>(`/stories/${storyId}/puzzles/${puzzleId}`, { method: "PUT", body: JSON.stringify(data) }),
  deletePuzzle: (storyId: string, puzzleId: string) =>
    apiRequest<void>(`/stories/${storyId}/puzzles/${puzzleId}`, { method: "DELETE" }),
};
