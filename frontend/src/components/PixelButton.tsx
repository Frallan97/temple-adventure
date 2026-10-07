import type { ReactNode } from "react";

type Tone = "neutral" | "amber" | "cyan";

interface PixelButtonProps {
  children: ReactNode;
  onClick: () => void;
  disabled?: boolean;
  tone?: Tone;
  title?: string;
  className?: string;
}

const TONES: Record<Tone, string> = {
  neutral:
    "bg-gray-900 text-gray-300 border-gray-600 hover:bg-gray-800 hover:text-white hover:border-gray-400",
  amber:
    "bg-gray-900 text-amber-400 border-amber-700 hover:bg-amber-950 hover:text-amber-200 hover:border-amber-500",
  cyan: "bg-gray-900 text-cyan-400 border-cyan-800 hover:bg-cyan-950 hover:text-cyan-200 hover:border-cyan-500",
};

/** Chunky 8-bit style button: notched corners, hard offset shadow, no blur. */
export function PixelButton({
  children,
  onClick,
  disabled = false,
  tone = "neutral",
  title,
  className = "",
}: PixelButtonProps) {
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      title={title}
      // No font-bold: Press Start 2P ships weight 400 only, so bolding is
      // synthesised by the browser and smears the pixel edges. The face is
      // already wide, so it needs less tracking and a smaller size than a
      // normal monospace would.
      className={`pixel-btn border-2 px-2.5 sm:px-3 py-2 text-[9px] sm:text-[10px] uppercase tracking-[0.05em] leading-none ${TONES[tone]} ${className}`}
    >
      {children}
    </button>
  );
}
