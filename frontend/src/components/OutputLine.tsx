import type { OutputEntry } from "../types/game";
import { useTypewriter } from "../hooks/useTypewriter";

interface OutputLineProps {
  entry: OutputEntry;
  isLatest: boolean;
  onComplete?: () => void;
}

export function OutputLine({ entry, isLatest, onComplete }: OutputLineProps) {
  const shouldAnimate = isLatest && entry.type === "narrative";
  const { displayedText, isComplete, skip } = useTypewriter(
    entry.text,
    shouldAnimate ? 10 : 0
  );

  const text = shouldAnimate ? displayedText : entry.text;

  if (isComplete && onComplete && shouldAnimate) {
    onComplete();
  }

  const colorClass = {
    command: "text-green-400",
    narrative: "text-amber-300",
    system: "text-cyan-400",
    error: "text-red-400",
  }[entry.type];

  return (
    <div
      className={`${colorClass} whitespace-pre-wrap leading-relaxed`}
      onClick={shouldAnimate && !isComplete ? skip : undefined}
    >
      {text}
    </div>
  );
}
