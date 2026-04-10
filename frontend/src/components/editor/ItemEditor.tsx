import { useState } from "react";
import { storyApi } from "../../lib/api-client";
import type { ItemDef } from "../../types/story";
import { JsonEditor } from "./JsonEditor";

interface Props {
  storyId: string;
  items: Record<string, ItemDef>;
  onReload: () => void;
  showStatus: (msg: string) => void;
}

export function ItemEditor({ storyId, items, onReload, showStatus }: Props) {
  const [selected, setSelected] = useState<string | null>(null);
  const [newId, setNewId] = useState("");

  const itemIds = Object.keys(items).sort();
  const item = selected ? items[selected] : null;

  const handleSave = async (itemId: string, data: ItemDef) => {
    try {
      const { id: _, ...rest } = data;
      await storyApi.upsertItem(storyId, itemId, rest);
      showStatus(`Item "${itemId}" saved`);
      onReload();
    } catch (err) {
      showStatus(`Failed: ${err}`);
    }
  };

  const handleDelete = async (itemId: string) => {
    if (!confirm(`Delete item "${itemId}"?`)) return;
    await storyApi.deleteItem(storyId, itemId);
    setSelected(null);
    showStatus(`Item "${itemId}" deleted`);
    onReload();
  };

  const handleCreate = async () => {
    if (!newId.trim()) return;
    const id = newId.trim();
    await storyApi.upsertItem(storyId, id, {
      name: id,
      aliases: [],
      description: "",
      portable: true,
      interactions: [],
      conditional_descriptions: [],
    });
    setNewId("");
    setSelected(id);
    showStatus(`Item "${id}" created`);
    onReload();
  };

  return (
    <div className="flex gap-6">
      <div className="w-52 space-y-2 shrink-0">
        <div className="flex gap-1.5">
          <input
            value={newId}
            onChange={(e) => setNewId(e.target.value)}
            placeholder="new_item_id"
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
          {itemIds.map((id) => (
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

      <div className="flex-1 min-w-0">
        {item && selected ? (
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-amber-400 font-bold text-lg">{selected}</h3>
              <button
                onClick={() => handleDelete(selected)}
                className="text-red-400/80 text-sm hover:text-red-300 transition-colors"
              >
                Delete Item
              </button>
            </div>
            <JsonEditor
              data={item}
              onSave={(data) => handleSave(selected, data as ItemDef)}
            />
          </div>
        ) : (
          <p className="text-gray-600 py-8">
            Select an item or create a new one.
          </p>
        )}
      </div>
    </div>
  );
}
