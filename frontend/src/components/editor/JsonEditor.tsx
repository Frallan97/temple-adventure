import { useState, useEffect } from "react";

interface Props {
  data: unknown;
  onSave: (data: unknown) => void;
}

export function JsonEditor({ data, onSave }: Props) {
  const [text, setText] = useState("");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setText(JSON.stringify(data, null, 2));
    setError(null);
  }, [data]);

  const handleSave = () => {
    try {
      const parsed = JSON.parse(text);
      setError(null);
      onSave(parsed);
    } catch (err) {
      setError(`Invalid JSON: ${err}`);
    }
  };

  return (
    <div className="space-y-3">
      <textarea
        value={text}
        onChange={(e) => {
          setText(e.target.value);
          setError(null);
        }}
        spellCheck={false}
        className="w-full bg-gray-900 border border-gray-700 rounded-lg px-4 py-3 text-gray-200 font-mono text-sm leading-relaxed resize-y focus:border-amber-700/50 focus:outline-none transition-colors"
        style={{ minHeight: "400px" }}
      />
      {error && <p className="text-red-400 text-sm">{error}</p>}
      <button
        onClick={handleSave}
        className="bg-amber-900/30 border border-amber-700 text-amber-400 px-5 py-2 rounded-lg hover:bg-amber-900/50 transition-all"
      >
        Save
      </button>
    </div>
  );
}
