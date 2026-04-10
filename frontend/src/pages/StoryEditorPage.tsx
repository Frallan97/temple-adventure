import { useState, useEffect, useCallback } from "react";
import { useParams, Link } from "react-router-dom";
import { storyApi } from "../lib/api-client";
import type {
  Story,
  RoomDef,
  ItemDef,
  PuzzleDef,
  ValidateResponse,
} from "../types/story";
import { StorySettings } from "../components/editor/StorySettings";
import { RoomEditor } from "../components/editor/RoomEditor";
import { ItemEditor } from "../components/editor/ItemEditor";
import { PuzzleEditor } from "../components/editor/PuzzleEditor";

type Tab = "settings" | "rooms" | "items" | "puzzles";

export function StoryEditorPage() {
  const { storyId } = useParams<{ storyId: string }>();
  const [story, setStory] = useState<Story | null>(null);
  const [rooms, setRooms] = useState<Record<string, RoomDef>>({});
  const [items, setItems] = useState<Record<string, ItemDef>>({});
  const [puzzles, setPuzzles] = useState<Record<string, PuzzleDef>>({});
  const [tab, setTab] = useState<Tab>("settings");
  const [loading, setLoading] = useState(true);
  const [validation, setValidation] = useState<ValidateResponse | null>(null);
  const [statusMsg, setStatusMsg] = useState<string | null>(null);

  const loadStory = useCallback(async () => {
    if (!storyId) return;
    try {
      const resp = await storyApi.get(storyId);
      setStory(resp.story);
      setRooms(resp.rooms || {});
      setItems(resp.items || {});
      setPuzzles(resp.puzzles || {});
    } catch (err) {
      setStatusMsg(`Failed to load: ${err}`);
    } finally {
      setLoading(false);
    }
  }, [storyId]);

  useEffect(() => {
    loadStory();
  }, [loadStory]);

  const showStatus = (msg: string) => {
    setStatusMsg(msg);
    setTimeout(() => setStatusMsg(null), 3000);
  };

  const handleValidate = async () => {
    if (!storyId) return;
    const resp = await storyApi.validate(storyId);
    setValidation(resp);
    showStatus(resp.valid ? "Validation passed!" : `${resp.errors.length} error(s) found`);
  };

  const handlePublish = async () => {
    if (!storyId) return;
    try {
      await storyApi.publish(storyId);
      showStatus("Published!");
      loadStory();
    } catch (err) {
      showStatus(`Publish failed: ${err}`);
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-950 text-gray-500 font-mono flex items-center justify-center">
        Loading...
      </div>
    );
  }

  if (!story || !storyId) {
    return (
      <div className="min-h-screen bg-gray-950 text-red-400 font-mono flex items-center justify-center">
        Story not found
      </div>
    );
  }

  const tabs: { key: Tab; label: string; count: number }[] = [
    { key: "settings", label: "Settings", count: 0 },
    { key: "rooms", label: "Rooms", count: Object.keys(rooms).length },
    { key: "items", label: "Items", count: Object.keys(items).length },
    { key: "puzzles", label: "Puzzles", count: Object.keys(puzzles).length },
  ];

  return (
    <div className="min-h-screen bg-gray-950 text-gray-200 font-mono">
      {/* Header */}
      <div className="border-b border-gray-800 px-4 sm:px-6 py-3 sm:py-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div className="flex items-center gap-3 sm:gap-4 min-w-0">
          <Link
            to="/editor"
            className="text-gray-500 hover:text-gray-400 transition-colors shrink-0"
          >
            &larr;
            <span className="hidden sm:inline"> Stories</span>
          </Link>
          <div className="hidden sm:block w-px h-5 bg-gray-800" />
          <h1 className="text-base sm:text-lg text-amber-400 font-bold truncate">
            {story.name}
          </h1>
          <span
            className={`text-xs px-2.5 py-0.5 rounded-full shrink-0 ${
              story.is_published
                ? "bg-green-900/30 text-green-400 border border-green-800/50"
                : "bg-gray-800 text-gray-500 border border-gray-700"
            }`}
          >
            {story.is_published ? "Published" : "Draft"}
          </span>
        </div>
        <div className="flex items-center gap-2 sm:gap-3">
          {statusMsg && (
            <span className="text-xs sm:text-sm text-gray-400 animate-fade-in truncate">
              {statusMsg}
            </span>
          )}
          <button
            onClick={handleValidate}
            className="bg-gray-800 border border-gray-600 text-gray-300 px-3 sm:px-4 py-1.5 rounded-lg hover:bg-gray-700 transition-colors text-sm shrink-0"
          >
            Validate
          </button>
          <button
            onClick={handlePublish}
            className="bg-green-900/30 border border-green-700 text-green-400 px-3 sm:px-4 py-1.5 rounded-lg hover:bg-green-900/50 transition-all text-sm shrink-0"
          >
            Publish
          </button>
        </div>
      </div>

      {/* Validation errors */}
      {validation && !validation.valid && (
        <div className="bg-red-900/20 border-b border-red-800/50 px-6 py-3">
          {validation.errors.map((e, i) => (
            <p key={i} className="text-red-400 text-sm py-0.5">
              <span className="text-red-500/60">[{e.field}]</span> {e.message}
            </p>
          ))}
        </div>
      )}

      {/* Tabs */}
      <div className="border-b border-gray-800 px-4 sm:px-6 flex gap-0 overflow-x-auto scrollbar-none">
        {tabs.map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`px-4 sm:px-5 py-3 text-sm border-b-2 transition-colors whitespace-nowrap ${
              tab === t.key
                ? "border-amber-400 text-amber-400"
                : "border-transparent text-gray-500 hover:text-gray-300"
            }`}
          >
            {t.label}
            {t.count > 0 && (
              <span className="ml-1.5 text-gray-600 text-xs">
                ({t.count})
              </span>
            )}
          </button>
        ))}
      </div>

      {/* Content */}
      <div className="p-4 sm:p-6 max-w-6xl">
        {tab === "settings" && (
          <StorySettings
            story={story}
            storyId={storyId}
            onUpdate={(updated) => setStory(updated)}
            showStatus={showStatus}
          />
        )}
        {tab === "rooms" && (
          <RoomEditor
            storyId={storyId}
            rooms={rooms}
            onReload={loadStory}
            showStatus={showStatus}
          />
        )}
        {tab === "items" && (
          <ItemEditor
            storyId={storyId}
            items={items}
            onReload={loadStory}
            showStatus={showStatus}
          />
        )}
        {tab === "puzzles" && (
          <PuzzleEditor
            storyId={storyId}
            puzzles={puzzles}
            onReload={loadStory}
            showStatus={showStatus}
          />
        )}
      </div>
    </div>
  );
}
