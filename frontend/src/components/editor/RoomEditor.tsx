import { useState } from "react";
import { storyApi } from "../../lib/api-client";
import type { RoomDef } from "../../types/story";
import { JsonEditor } from "./JsonEditor";

interface Props {
  storyId: string;
  rooms: Record<string, RoomDef>;
  onReload: () => void;
  showStatus: (msg: string) => void;
}

export function RoomEditor({ storyId, rooms, onReload, showStatus }: Props) {
  const [selected, setSelected] = useState<string | null>(null);
  const [newId, setNewId] = useState("");

  const roomIds = Object.keys(rooms).sort();
  const room = selected ? rooms[selected] : null;

  const handleSave = async (roomId: string, data: RoomDef) => {
    try {
      const { id: _, ...rest } = data;
      await storyApi.upsertRoom(storyId, roomId, rest);
      showStatus(`Room "${roomId}" saved`);
      onReload();
    } catch (err) {
      showStatus(`Failed: ${err}`);
    }
  };

  const handleDelete = async (roomId: string) => {
    if (!confirm(`Delete room "${roomId}"?`)) return;
    await storyApi.deleteRoom(storyId, roomId);
    setSelected(null);
    showStatus(`Room "${roomId}" deleted`);
    onReload();
  };

  const handleCreate = async () => {
    if (!newId.trim()) return;
    const id = newId.trim();
    await storyApi.upsertRoom(storyId, id, {
      name: id,
      description: "",
      connections: {},
      items: [],
      puzzles: [],
      conditional_descriptions: [],
      hints: [],
    });
    setNewId("");
    setSelected(id);
    showStatus(`Room "${id}" created`);
    onReload();
  };

  return (
    <div className="flex flex-col sm:flex-row gap-4 sm:gap-6">
      {/* Sidebar */}
      <div className="w-full sm:w-52 space-y-2 shrink-0">
        <div className="flex gap-1.5">
          <input
            value={newId}
            onChange={(e) => setNewId(e.target.value)}
            placeholder="new_room_id"
            className="flex-1 bg-gray-800 border border-gray-700 rounded-lg px-2.5 py-1.5 text-gray-200 text-sm placeholder-gray-600 focus:border-amber-700/50 focus:outline-none transition-colors"
            onKeyDown={(e) => e.key === "Enter" && handleCreate()}
          />
          <button
            onClick={handleCreate}
            className="bg-green-900/30 border border-green-700 text-green-400 px-2.5 py-1.5 rounded-lg text-sm hover:bg-green-900/50 transition-colors"
          >
            +
          </button>
        </div>
        <div className="space-y-1">
          {roomIds.map((id) => (
            <button
              key={id}
              onClick={() => setSelected(id)}
              className={`block w-full text-left px-3 py-2 rounded-lg text-sm truncate transition-colors ${
                selected === id
                  ? "bg-amber-900/30 text-amber-400 border border-amber-700"
                  : "text-gray-400 hover:bg-gray-800/60 border border-transparent"
              }`}
            >
              {id}
            </button>
          ))}
        </div>
      </div>

      {/* Editor */}
      <div className="flex-1 min-w-0">
        {room && selected ? (
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-amber-400 font-bold text-lg">{selected}</h3>
              <button
                onClick={() => handleDelete(selected)}
                className="text-red-400/80 text-sm hover:text-red-300 transition-colors"
              >
                Delete Room
              </button>
            </div>
            <JsonEditor
              data={room}
              onSave={(data) => handleSave(selected, data as RoomDef)}
            />
          </div>
        ) : (
          <p className="text-gray-600 py-8">
            Select a room or create a new one.
          </p>
        )}
      </div>
    </div>
  );
}
