import { useState, useRef, useEffect, type KeyboardEvent } from "react";

interface CommandInputProps {
  onSubmit: (input: string) => void;
  onNavigateHistory: (direction: "up" | "down") => string;
  disabled: boolean;
}

export function CommandInput({
  onSubmit,
  onNavigateHistory,
  disabled,
}: CommandInputProps) {
  const [value, setValue] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    inputRef.current?.focus();
  }, [disabled]);

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter" && value.trim()) {
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
        className="flex-1 bg-transparent border-none outline-none text-green-400 caret-green-400 placeholder-gray-700 font-mono text-sm"
        autoFocus
      />
    </div>
  );
}
