import { useMemo, useCallback } from "react";
import {
  ReactFlow,
  Background,
  Controls,
  type Node,
  type Edge,
  type NodeTypes,
  Handle,
  Position,
  useNodesState,
  useEdgesState,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import type { RoomDef, ItemDef, PuzzleDef } from "../../types/story";

interface StoryGraphProps {
  rooms: Record<string, RoomDef>;
  items: Record<string, ItemDef>;
  puzzles: Record<string, PuzzleDef>;
  startRoom: string;
}

// Direction → angle for layout
const directionOffset: Record<string, { x: number; y: number }> = {
  north: { x: 0, y: -220 },
  south: { x: 0, y: 220 },
  east: { x: 280, y: 0 },
  west: { x: -280, y: 0 },
  up: { x: 140, y: -180 },
  down: { x: -140, y: 180 },
};

// Direction → source/target handle positions
const directionHandles: Record<
  string,
  { source: Position; target: Position }
> = {
  north: { source: Position.Top, target: Position.Bottom },
  south: { source: Position.Bottom, target: Position.Top },
  east: { source: Position.Right, target: Position.Left },
  west: { source: Position.Left, target: Position.Right },
  up: { source: Position.Top, target: Position.Bottom },
  down: { source: Position.Bottom, target: Position.Top },
};

const directionArrow: Record<string, string> = {
  north: "↑ N",
  south: "↓ S",
  east: "→ E",
  west: "← W",
  up: "↑ Up",
  down: "↓ Down",
};

interface PuzzleInfo {
  id: string;
  name: string;
  type: string;
  isTimed: boolean;
}

function getPuzzleColor(p: PuzzleInfo): string {
  if (p.isTimed) return "text-red-400 border-red-700 bg-red-950/40";
  if (p.type === "key_lock") return "text-yellow-400 border-yellow-700 bg-yellow-950/40";
  if (p.type === "examine_learn") return "text-cyan-400 border-cyan-700 bg-cyan-950/40";
  if (p.type === "fetch_quest") return "text-purple-400 border-purple-700 bg-purple-950/40";
  return "text-gray-400 border-gray-600 bg-gray-800/40";
}

function inferPuzzleType(p: PuzzleDef): string {
  if (p.timed_window) return "timed";
  if (p.steps?.length >= 3) return "examine_learn";
  return "puzzle";
}

function RoomNode({ data }: { data: RoomNodeData }) {
  const { label, isStart, itemNames, puzzleInfos, lockedDirections } = data;

  return (
    <div
      className={`rounded-xl border-2 px-4 py-3 min-w-[180px] max-w-[240px] font-mono shadow-lg ${
        isStart
          ? "border-amber-500 bg-gray-900 shadow-amber-900/30"
          : "border-gray-600 bg-gray-900 shadow-black/40"
      }`}
    >
      <Handle type="target" position={Position.Top} id="top" className="!bg-gray-600 !w-2 !h-2" />
      <Handle type="source" position={Position.Top} id="top-src" className="!bg-gray-600 !w-2 !h-2" />
      <Handle type="target" position={Position.Bottom} id="bottom" className="!bg-gray-600 !w-2 !h-2" />
      <Handle type="source" position={Position.Bottom} id="bottom-src" className="!bg-gray-600 !w-2 !h-2" />
      <Handle type="target" position={Position.Left} id="left" className="!bg-gray-600 !w-2 !h-2" />
      <Handle type="source" position={Position.Left} id="left-src" className="!bg-gray-600 !w-2 !h-2" />
      <Handle type="target" position={Position.Right} id="right" className="!bg-gray-600 !w-2 !h-2" />
      <Handle type="source" position={Position.Right} id="right-src" className="!bg-gray-600 !w-2 !h-2" />

      {/* Room name */}
      <div className="flex items-center gap-2 mb-1">
        {isStart && <span className="text-amber-400 text-xs">▶</span>}
        <span className={`text-sm font-bold ${isStart ? "text-amber-400" : "text-gray-200"}`}>
          {label}
        </span>
      </div>

      {/* Items */}
      {itemNames.length > 0 && (
        <div className="mt-2 flex flex-wrap gap-1">
          {itemNames.map((name) => (
            <span
              key={name}
              className="text-[10px] px-1.5 py-0.5 rounded bg-gray-800 text-gray-400 border border-gray-700"
            >
              {name}
            </span>
          ))}
        </div>
      )}

      {/* Puzzles */}
      {puzzleInfos.length > 0 && (
        <div className="mt-2 flex flex-col gap-1">
          {puzzleInfos.map((p) => (
            <span
              key={p.id}
              className={`text-[10px] px-1.5 py-0.5 rounded border ${getPuzzleColor(p)}`}
            >
              {p.isTimed && "⏱ "}
              {p.name}
            </span>
          ))}
        </div>
      )}

      {/* Locked directions indicator */}
      {lockedDirections.length > 0 && (
        <div className="mt-2 flex flex-wrap gap-1">
          {lockedDirections.map((dir) => (
            <span
              key={dir}
              className="text-[10px] px-1.5 py-0.5 rounded bg-red-950/30 text-red-400 border border-red-800/50"
            >
              🔒 {dir}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}

interface RoomNodeData {
  label: string;
  isStart: boolean;
  itemNames: string[];
  puzzleInfos: PuzzleInfo[];
  lockedDirections: string[];
  [key: string]: unknown;
}

const nodeTypes: NodeTypes = {
  room: RoomNode,
};

// BFS-based layout that respects directional relationships
function layoutRooms(
  rooms: Record<string, RoomDef>,
  startRoom: string,
  allConnections: Map<string, Map<string, string>>
): Map<string, { x: number; y: number }> {
  const positions = new Map<string, { x: number; y: number }>();
  const visited = new Set<string>();
  const queue: string[] = [];

  // Start room at center
  positions.set(startRoom, { x: 0, y: 0 });
  visited.add(startRoom);
  queue.push(startRoom);

  while (queue.length > 0) {
    const roomId = queue.shift()!;
    const pos = positions.get(roomId)!;
    const connections = allConnections.get(roomId) || new Map();

    for (const [dir, targetId] of connections) {
      if (visited.has(targetId)) continue;
      if (!rooms[targetId]) continue;

      const offset = directionOffset[dir] || { x: 280, y: 0 };
      let newX = pos.x + offset.x;
      let newY = pos.y + offset.y;

      // Nudge if position is already taken
      let attempts = 0;
      while (
        attempts < 8 &&
        Array.from(positions.values()).some(
          (p) => Math.abs(p.x - newX) < 100 && Math.abs(p.y - newY) < 100
        )
      ) {
        newX += 60;
        newY += 30;
        attempts++;
      }

      positions.set(targetId, { x: newX, y: newY });
      visited.add(targetId);
      queue.push(targetId);
    }
  }

  // Place any unreachable rooms in a row below
  const roomIds = Object.keys(rooms);
  let unplacedX = 0;
  for (const id of roomIds) {
    if (!positions.has(id)) {
      positions.set(id, { x: unplacedX, y: 500 });
      unplacedX += 300;
    }
  }

  return positions;
}

export function StoryGraph({ rooms, items, puzzles, startRoom }: StoryGraphProps) {
  // Collect ALL connections (including locked ones from puzzle effects)
  const allConnections = useMemo(() => {
    const conns = new Map<string, Map<string, string>>();

    // Base connections from rooms
    for (const [roomId, room] of Object.entries(rooms)) {
      const roomConns = new Map<string, string>();
      for (const [dir, target] of Object.entries(room.connections || {})) {
        roomConns.set(dir, target);
      }
      conns.set(roomId, roomConns);
    }

    // Locked connections from puzzle step effects (unlock_connection)
    for (const puzzle of Object.values(puzzles)) {
      for (const step of puzzle.steps || []) {
        for (const effect of step.effects || []) {
          if (effect.type === "unlock_connection" && typeof effect.key === "string") {
            const [roomId, dir] = effect.key.split(".");
            if (roomId && dir && typeof effect.value === "string") {
              if (!conns.has(roomId)) conns.set(roomId, new Map());
              conns.get(roomId)!.set(dir, effect.value as string);
            }
          }
        }
      }
    }

    // Locked connections from item interaction effects
    for (const item of Object.values(items)) {
      for (const inter of item.interactions || []) {
        for (const effect of inter.effects || []) {
          if (effect.type === "unlock_connection" && typeof effect.key === "string") {
            const [roomId, dir] = effect.key.split(".");
            if (roomId && dir && typeof effect.value === "string") {
              if (!conns.has(roomId)) conns.set(roomId, new Map());
              conns.get(roomId)!.set(dir, effect.value as string);
            }
          }
        }
      }
    }

    return conns;
  }, [rooms, items, puzzles]);

  // Determine which connections are locked (not in base room connections)
  const lockedConnections = useMemo(() => {
    const locked = new Map<string, Set<string>>();
    for (const [roomId, conns] of allConnections) {
      const baseConns = rooms[roomId]?.connections || {};
      for (const [dir] of conns) {
        if (!baseConns[dir]) {
          if (!locked.has(roomId)) locked.set(roomId, new Set());
          locked.get(roomId)!.add(dir);
        }
      }
    }
    return locked;
  }, [allConnections, rooms]);

  const { initialNodes, initialEdges } = useMemo(() => {
    const positions = layoutRooms(rooms, startRoom, allConnections);

    const nodes: Node[] = [];
    const edges: Edge[] = [];
    const edgeSet = new Set<string>();

    for (const [roomId, room] of Object.entries(rooms)) {
      const pos = positions.get(roomId) || { x: 0, y: 0 };

      // Get item names for this room
      const itemNames = (room.items || [])
        .map((id) => items[id]?.name || id)
        .filter(Boolean);

      // Get puzzle info for this room
      const puzzleInfos: PuzzleInfo[] = (room.puzzles || [])
        .map((id) => {
          const p = puzzles[id];
          if (!p) return null;
          return {
            id: p.id,
            name: p.name,
            type: inferPuzzleType(p),
            isTimed: !!p.timed_window,
          };
        })
        .filter((p): p is PuzzleInfo => p !== null);

      // Locked directions for this room
      const roomLocked = lockedConnections.get(roomId);
      const lockedDirs = roomLocked ? Array.from(roomLocked) : [];

      nodes.push({
        id: roomId,
        type: "room",
        position: pos,
        data: {
          label: room.name,
          isStart: roomId === startRoom,
          itemNames,
          puzzleInfos,
          lockedDirections: lockedDirs,
        } satisfies RoomNodeData,
      });
    }

    // Edges from all connections
    for (const [roomId, conns] of allConnections) {
      for (const [dir, targetId] of conns) {
        if (!rooms[targetId]) continue;

        // Deduplicate bidirectional edges
        const edgeKey = [roomId, targetId].sort().join("--");
        const isLocked = lockedConnections.get(roomId)?.has(dir);

        if (!edgeSet.has(edgeKey)) {
          edgeSet.add(edgeKey);

          const handles = directionHandles[dir] || {
            source: Position.Right,
            target: Position.Left,
          };

          const sourceHandleId = handles.source === Position.Top ? "top-src"
            : handles.source === Position.Bottom ? "bottom-src"
            : handles.source === Position.Left ? "left-src"
            : "right-src";

          const targetHandleId = handles.target === Position.Top ? "top"
            : handles.target === Position.Bottom ? "bottom"
            : handles.target === Position.Left ? "left"
            : "right";

          edges.push({
            id: `${roomId}-${dir}-${targetId}`,
            source: roomId,
            target: targetId,
            sourceHandle: sourceHandleId,
            targetHandle: targetHandleId,
            label: directionArrow[dir] || dir,
            type: "default",
            animated: isLocked,
            style: {
              stroke: isLocked ? "#b91c1c" : "#6b7280",
              strokeWidth: isLocked ? 2 : 1.5,
              strokeDasharray: isLocked ? "6 3" : undefined,
            },
            labelStyle: {
              fill: isLocked ? "#f87171" : "#9ca3af",
              fontSize: 11,
              fontFamily: "monospace",
              fontWeight: 500,
            },
            labelBgStyle: {
              fill: "#030712",
              fillOpacity: 0.9,
            },
            labelBgPadding: [4, 2] as [number, number],
          });
        }
      }
    }

    return { initialNodes: nodes, initialEdges: edges };
  }, [rooms, items, puzzles, startRoom, allConnections, lockedConnections]);

  const [nodes, , onNodesChange] = useNodesState(initialNodes);
  const [edges, , onEdgesChange] = useEdgesState(initialEdges);

  const onInit = useCallback(() => {
    // fit view is handled by ReactFlow's fitView prop
  }, []);

  return (
    <div className="w-full h-[calc(100vh-200px)] min-h-[500px] rounded-xl border border-gray-800 overflow-hidden bg-gray-950">
      {/* Legend */}
      <div className="absolute top-2 right-2 z-10 bg-gray-900/90 border border-gray-700 rounded-lg px-3 py-2 font-mono text-[10px] flex flex-col gap-1">
        <div className="text-gray-400 font-bold mb-0.5">Legend</div>
        <div className="flex items-center gap-1.5">
          <span className="text-amber-400">▶</span>
          <span className="text-gray-400">Start room</span>
        </div>
        <div className="flex items-center gap-1.5">
          <span className="w-4 border-t-2 border-gray-500 inline-block" />
          <span className="text-gray-400">Open path</span>
        </div>
        <div className="flex items-center gap-1.5">
          <span className="w-4 border-t-2 border-dashed border-red-700 inline-block" />
          <span className="text-gray-400">Locked path</span>
        </div>
        <div className="flex items-center gap-1.5">
          <span className="text-red-400">⏱</span>
          <span className="text-gray-400">Timed puzzle</span>
        </div>
        <div className="flex items-center gap-1.5">
          <span className="text-red-400">🔒</span>
          <span className="text-gray-400">Locked exit</span>
        </div>
      </div>

      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onInit={onInit}
        nodeTypes={nodeTypes}
        fitView
        fitViewOptions={{ padding: 0.3 }}
        minZoom={0.3}
        maxZoom={2}
        proOptions={{ hideAttribution: true }}
        className="bg-gray-950"
      >
        <Background color="#1f2937" gap={20} size={1} />
        <Controls
          className="!bg-gray-900 !border-gray-700 !rounded-lg [&>button]:!bg-gray-800 [&>button]:!border-gray-700 [&>button]:!text-gray-400 [&>button:hover]:!bg-gray-700"
          showInteractive={false}
        />
      </ReactFlow>
    </div>
  );
}
