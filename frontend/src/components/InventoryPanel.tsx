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
  return (
    <>
      <button
        onClick={onToggle}
        className="fixed top-4 right-4 z-10 bg-gray-900/90 backdrop-blur text-amber-400 px-4 py-1.5 rounded-lg border border-gray-700/80 hover:bg-gray-800 hover:border-gray-600 font-mono text-sm transition-all shadow-lg"
      >
        {isOpen ? "Close" : "Inventory"}
      </button>

      {isOpen && (
        <div className="fixed top-0 right-0 h-full w-64 bg-gray-900/95 backdrop-blur-sm border-l border-gray-800 font-mono text-sm z-5 shadow-2xl">
          <div className="pt-14 px-5 space-y-5">
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

            <div className="border-t border-gray-800 pt-4">
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
        </div>
      )}
    </>
  );
}
