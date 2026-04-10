import { useState, useEffect } from "react";
import { useNavigate, Link } from "react-router-dom";
import { storyApi } from "../lib/api-client";
import type { StorySummary } from "../types/story";

export function EditorPage() {
  const navigate = useNavigate();
  const [stories, setStories] = useState<StorySummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [newStory, setNewStory] = useState({
    name: "",
    slug: "",
    description: "",
    author: "",
    start_room: "entrance",
  });
  const [error, setError] = useState<string | null>(null);

  const loadStories = () => {
    storyApi.listAll().then((resp) => {
      setStories(resp.stories || []);
      setLoading(false);
    });
  };

  useEffect(loadStories, []);

  const handleCreate = async () => {
    setError(null);
    try {
      const story = await storyApi.create(newStory);
      navigate(`/editor/${story.id}`);
    } catch (err) {
      setError(`${err}`);
    }
  };

  const handleDelete = async (id: string, name: string) => {
    if (!confirm(`Delete "${name}"? This cannot be undone.`)) return;
    await storyApi.delete(id);
    loadStories();
  };

  const autoSlug = (name: string) =>
    name
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/^-|-$/g, "");

  return (
    <div className="min-h-screen bg-gray-950 text-gray-200 font-mono px-6 py-10">
      <div className="max-w-4xl mx-auto">
        <div className="flex items-center justify-between mb-10">
          <div className="space-y-1">
            <h1 className="text-2xl font-bold text-amber-400 tracking-wide">
              Story Editor
            </h1>
            <Link
              to="/"
              className="text-gray-500 text-sm hover:text-gray-400 transition-colors"
            >
              &larr; Back to game
            </Link>
          </div>
          <button
            onClick={() => setShowCreate(!showCreate)}
            className="bg-amber-900/30 border border-amber-700 text-amber-400 px-5 py-2 rounded-lg hover:bg-amber-900/50 transition-all hover:shadow-md hover:shadow-amber-900/20"
          >
            {showCreate ? "Cancel" : "New Story"}
          </button>
        </div>

        {showCreate && (
          <div className="border border-gray-700 rounded-xl p-6 mb-8 bg-gray-900/50 space-y-4 shadow-lg">
            <h2 className="text-lg text-amber-400 font-semibold">
              Create New Story
            </h2>
            {error && <p className="text-red-400 text-sm">{error}</p>}
            <div className="grid grid-cols-2 gap-4">
              <input
                placeholder="Story name"
                value={newStory.name}
                onChange={(e) => {
                  const name = e.target.value;
                  setNewStory((s) => ({
                    ...s,
                    name,
                    slug: autoSlug(name),
                  }));
                }}
                className="bg-gray-800 border border-gray-700 rounded-lg px-3 py-2.5 text-gray-200 placeholder-gray-600 focus:border-amber-700/50 focus:outline-none transition-colors"
              />
              <input
                placeholder="Slug (url-safe)"
                value={newStory.slug}
                onChange={(e) =>
                  setNewStory((s) => ({ ...s, slug: e.target.value }))
                }
                className="bg-gray-800 border border-gray-700 rounded-lg px-3 py-2.5 text-gray-200 placeholder-gray-600 focus:border-amber-700/50 focus:outline-none transition-colors"
              />
              <input
                placeholder="Author"
                value={newStory.author}
                onChange={(e) =>
                  setNewStory((s) => ({ ...s, author: e.target.value }))
                }
                className="bg-gray-800 border border-gray-700 rounded-lg px-3 py-2.5 text-gray-200 placeholder-gray-600 focus:border-amber-700/50 focus:outline-none transition-colors"
              />
              <input
                placeholder="Start room ID"
                value={newStory.start_room}
                onChange={(e) =>
                  setNewStory((s) => ({ ...s, start_room: e.target.value }))
                }
                className="bg-gray-800 border border-gray-700 rounded-lg px-3 py-2.5 text-gray-200 placeholder-gray-600 focus:border-amber-700/50 focus:outline-none transition-colors"
              />
            </div>
            <textarea
              placeholder="Description"
              value={newStory.description}
              onChange={(e) =>
                setNewStory((s) => ({ ...s, description: e.target.value }))
              }
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2.5 text-gray-200 placeholder-gray-600 h-24 focus:border-amber-700/50 focus:outline-none transition-colors"
            />
            <button
              onClick={handleCreate}
              className="bg-green-900/30 border border-green-700 text-green-400 px-5 py-2 rounded-lg hover:bg-green-900/50 transition-all"
            >
              Create
            </button>
          </div>
        )}

        {loading ? (
          <p className="text-gray-500 animate-pulse py-8">Loading...</p>
        ) : stories.length === 0 ? (
          <p className="text-gray-600 py-8">
            No stories yet. Create one to get started.
          </p>
        ) : (
          <div className="space-y-4">
            {stories.map((story) => (
              <div
                key={story.id}
                className="border border-gray-800 rounded-xl p-5 bg-gray-900/30 flex items-center justify-between hover:border-gray-700 transition-colors"
              >
                <div className="space-y-1">
                  <div className="flex items-center gap-3">
                    <h3 className="text-amber-400 font-bold">{story.name}</h3>
                    <span
                      className={`text-xs px-2.5 py-0.5 rounded-full ${
                        story.is_published
                          ? "bg-green-900/30 text-green-400 border border-green-800/50"
                          : "bg-gray-800 text-gray-500 border border-gray-700"
                      }`}
                    >
                      {story.is_published ? "Published" : "Draft"}
                    </span>
                  </div>
                  <p className="text-gray-500 text-sm">
                    {story.description || "No description"}
                  </p>
                  <p className="text-gray-600 text-xs">by {story.author}</p>
                </div>
                <div className="flex gap-2 ml-4">
                  <Link
                    to={`/editor/${story.id}`}
                    className="bg-gray-800 border border-gray-600 text-gray-300 px-4 py-1.5 rounded-lg hover:bg-gray-700 transition-colors text-sm"
                  >
                    Edit
                  </Link>
                  <button
                    onClick={() => handleDelete(story.id, story.name)}
                    className="bg-red-900/20 border border-red-800/50 text-red-400 px-4 py-1.5 rounded-lg hover:bg-red-900/40 transition-colors text-sm"
                  >
                    Delete
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
