import { useState } from "react";
import { storyApi } from "../../lib/api-client";
import type { Story } from "../../types/story";

interface Props {
  story: Story;
  storyId: string;
  onUpdate: (story: Story) => void;
  showStatus: (msg: string) => void;
}

export function StorySettings({ story, storyId, onUpdate, showStatus }: Props) {
  const [form, setForm] = useState({
    name: story.name,
    slug: story.slug,
    description: story.description,
    author: story.author,
    start_room: story.start_room,
  });

  const handleSave = async () => {
    try {
      const updated = await storyApi.update(storyId, form);
      onUpdate(updated);
      showStatus("Settings saved");
    } catch (err) {
      showStatus(`Failed to save: ${err}`);
    }
  };

  return (
    <div className="max-w-xl space-y-6">
      <h2 className="text-lg text-amber-400 font-semibold">Story Settings</h2>
      <div className="space-y-4">
        <label className="block">
          <span className="text-gray-400 text-sm mb-1.5 block">Name</span>
          <input
            value={form.name}
            onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
            className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2.5 text-gray-200 focus:border-amber-700/50 focus:outline-none transition-colors"
          />
        </label>
        <label className="block">
          <span className="text-gray-400 text-sm mb-1.5 block">Slug</span>
          <input
            value={form.slug}
            onChange={(e) => setForm((f) => ({ ...f, slug: e.target.value }))}
            className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2.5 text-gray-200 focus:border-amber-700/50 focus:outline-none transition-colors"
          />
        </label>
        <label className="block">
          <span className="text-gray-400 text-sm mb-1.5 block">Author</span>
          <input
            value={form.author}
            onChange={(e) => setForm((f) => ({ ...f, author: e.target.value }))}
            className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2.5 text-gray-200 focus:border-amber-700/50 focus:outline-none transition-colors"
          />
        </label>
        <label className="block">
          <span className="text-gray-400 text-sm mb-1.5 block">
            Start Room ID
          </span>
          <input
            value={form.start_room}
            onChange={(e) =>
              setForm((f) => ({ ...f, start_room: e.target.value }))
            }
            className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2.5 text-gray-200 focus:border-amber-700/50 focus:outline-none transition-colors"
          />
        </label>
        <label className="block">
          <span className="text-gray-400 text-sm mb-1.5 block">
            Description
          </span>
          <textarea
            value={form.description}
            onChange={(e) =>
              setForm((f) => ({ ...f, description: e.target.value }))
            }
            className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2.5 text-gray-200 h-28 focus:border-amber-700/50 focus:outline-none transition-colors"
          />
        </label>
      </div>
      <button
        onClick={handleSave}
        className="bg-amber-900/30 border border-amber-700 text-amber-400 px-5 py-2 rounded-lg hover:bg-amber-900/50 transition-all hover:shadow-md hover:shadow-amber-900/20"
      >
        Save Settings
      </button>
    </div>
  );
}
