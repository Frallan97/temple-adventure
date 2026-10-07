import { useState, useRef, useEffect, type KeyboardEvent } from "react";

import type { CommandContext } from "../types/game";
import { CommandText } from "./CommandText";
import { suggestCommands } from "../lib/command-suggestions";
interface CommandInputProps {
  onSubmit: (input: string) => void;
  onNavigateHistory: (direction: "up" | "down") => string;
  disabled: boolean;
  context?: CommandContext;
}

export function CommandInput({
  onSubmit,
  onNavigateHistory,
  disabled,
  context,
}: CommandInputProps) {
  const [value, setValue] = useState("");
  const suggestions = disabled ? [] : suggestCommands(value, context);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    inputRef.current?.focus();
  }, [disabled]);

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Tab" && suggestions.length) {
      e.preventDefault(); setValue(suggestions[0]);
    } else if (e.key === "Enter" && value.trim()) {
      onSubmit(value);
      setValue("");
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      const prev = onNavigateHistory("up");
      setValue(prev);
    } else if (e.key === "ArrowDown") {
      e.preventDefault();
      const next = onNavigateHistory("down");
      setValue(next);
    }
  };

  return (
    <div>
      <div className="flex gap-3 pb-2 text-xs" aria-label="Text colors"><span className="text-green-400">Action</span><span className="text-cyan-300">Object</span><span className="text-purple-300">NPC</span></div>
      {value && <div className="pb-2 text-gray-300" aria-label="Command preview"><CommandText text={value} context={context} command /></div>}
      {suggestions.length > 0 && <div className="flex flex-wrap gap-2 pb-3" aria-label="Command suggestions">
        {suggestions.map(command => <button key={command} type="button" className="border border-gray-700 px-2 py-2 text-gray-300 hover:bg-gray-800" onClick={() => { setValue(command); inputRef.current?.focus(); }}><CommandText text={command} context={context} command /></button>)}
        <span className="text-gray-500 text-xs self-center">Tab to complete · Enter to act</span>
      </div>}
    <div className="flex items-center gap-3 border-t border-gray-800 pt-3">
      <span className="text-green-400 font-bold text-base select-none">
        &gt;
      </span>
      <input
        ref={inputRef}
        type="text"
        value={value}
        onChange={(e) => setValue(e.target.value)}
        onKeyDown={handleKeyDown}
        disabled={disabled}
        placeholder={disabled ? "..." : "Enter command..."}
        autoComplete="off"
        autoCapitalize="none"
        autoCorrect="off"
        spellCheck={false}
        className="min-w-0 flex-1 bg-transparent border-none outline-none text-green-400 caret-green-400 placeholder-gray-700 font-mono text-base sm:text-sm py-1"
        autoFocus
      />
    </div>
    </div>
  );
}
