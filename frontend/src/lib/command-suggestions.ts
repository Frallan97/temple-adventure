import type { CommandContext } from "../types/game";
function distance(a: string, b: string): number {
 let row = Array.from({length:b.length+1}, (_,i) => i);
 for (let i=1;i<=a.length;i++) { const next=[i]; for(let j=1;j<=b.length;j++) next[j]=Math.min(next[j-1]+1,row[j]+1,row[j-1]+(a[i-1]===b[j-1]?0:1)); row=next; }
 return row[b.length];
}
export function suggestCommands(input: string, context?: CommandContext): string[] {
 const query=input.trim().toLowerCase(); if(!query || !context) return [];
 return context.commands.map(command => {
  const variants=[command.toLowerCase()];
  for(const entity of [...context.objects,...context.npcs]) if(command.endsWith(entity.name)) for(const alias of entity.aliases) variants.push(command.slice(0,-entity.name.length).toLowerCase()+alias.toLowerCase());
  const score=Math.min(...variants.map(candidate => candidate.startsWith(query)?0:candidate.includes(query)?1:distance(query,candidate.slice(0,query.length))+2));
  return {command,score};
 }).filter(e=>e.score<=Math.max(3,Math.floor(query.length/4)+2)).sort((a,b)=>a.score-b.score || a.command.localeCompare(b.command)).slice(0,6).map(e=>e.command);
}
