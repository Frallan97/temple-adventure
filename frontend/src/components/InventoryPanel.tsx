import { useEffect } from "react";
import type { ItemInfo } from "../types/game";

interface InventoryPanelProps {
  items: ItemInfo[];
  roomName: string;
  turnNumber: number;
  isOpen: boolean;
  onToggle: () => void;
}

export function InventoryPanel({
  items,
  roomName,
  turnNumber,
  isOpen,
  onToggle,
}: InventoryPanelProps) {
  // Escape closes the panel. The open/close button lives in the Terminal
  // header; this panel previously rendered its own toggle underneath itself,
  // which made it impossible to close.
  useEffect(() => {
    if (!isOpen) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onToggle();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [isOpen, onToggle]);

  if (!isOpen) return null;

  return (
    <>
      {/* Click anywhere outside to close (all sizes, not just mobile). */}
      <div
        className="fixed inset-0 bg-black/50 z-10"
        onClick={onToggle}
        aria-hidden="true"
      />

      <aside
        className="fixed top-0 right-0 h-full w-full sm:w-72 bg-gray-900 border-l-2 border-gray-700 font-mono text-sm z-20 flex flex-col"
        role="dialog"
        aria-label="Inventory"
      >
        {/* No title bar or close button of its own: the panel is fixed to the
            top of the viewport and would sit underneath the Terminal header,
            hiding them. Closing is handled by the header's Close button, the
            backdrop, and Escape. */}
        <div className="flex-1 overflow-y-auto px-5 pb-5 pt-16 space-y-5">
          <div>
            <div className="text-cyan-400/80 text-xs uppercase tracking-widest mb-1.5">
              Location
            </div>
            <div className="text-amber-300 font-semibold">
              {roomName || "Unknown"}
            </div>
          </div>

          <div>
            <div className="text-cyan-400/80 text-xs uppercase tracking-widest mb-1.5">
              Turn
            </div>
            <div className="text-gray-400">{turnNumber}</div>
          </div>

          <div className="border-t-2 border-gray-800 pt-4">
            <div className="text-cyan-400/80 text-xs uppercase tracking-widest mb-3">
              Inventory
            </div>
            {items.length === 0 ? (
              <div className="text-gray-600 italic">Empty</div>
            ) : (
              <ul className="space-y-2">
                {items.map((item) => (
                  <li
                    key={item.id}
                    className="text-amber-300 flex items-center gap-2"
                  >
                    <span className="text-amber-600 text-xs">&#9670;</span>
                    {item.name}
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      </aside>
    </>
  );
}
