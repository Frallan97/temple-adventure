import type { CommandContext } from "../types/game";
export function CommandText({ text, context, command = false }: { text: string; context?: CommandContext; command?: boolean }) {
 const terms = [...(context?.objects || []).flatMap(e => [e.name, ...e.aliases].map(term => ({term, color: "text-cyan-300"}))), ...(context?.npcs || []).flatMap(e => [e.name, ...e.aliases].map(term => ({term, color: "text-purple-300"})))].filter(e => e.term).sort((a,b) => b.term.length-a.term.length);
 const action = command ? text.match(/^(?:>\s*)?(\S+)/) : null;
 const nodes = []; let start = 0;
 for (let i = 0; i < text.length;) {
  const term = terms.find(e => text.slice(i,i+e.term.length).toLowerCase() === e.term.toLowerCase() && !/[\p{L}\p{N}_]/u.test(text[i-1] || "") && !/[\p{L}\p{N}_]/u.test(text[i+e.term.length] || ""));
  const isAction = action && i === action[0].length-action[1].length;
  const length = isAction ? action[1].length : term?.term.length;
  if (length) { nodes.push(text.slice(start,i), <span key={i} className={isAction ? "text-green-400" : term?.color}>{text.slice(i,i+length)}</span>); i += length; start = i; } else i++;
 }
 nodes.push(text.slice(start)); return <>{nodes}</>;
}
