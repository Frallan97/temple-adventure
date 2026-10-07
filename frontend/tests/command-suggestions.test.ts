import { test, expect } from "bun:test";
import { suggestCommands } from "../src/lib/command-suggestions";
const context = { actions: ["take", "talk"], objects: [{name:"Brass key",aliases:["key"]}], npcs:[{name:"Temple guard",aliases:["keeper"]}], commands:["take Brass key","talk Temple guard","look"] };
test("suggestions match prefixes, aliases and typos", () => {
 expect(suggestCommands("tak",context)[0]).toBe("take Brass key");
 expect(suggestCommands("take key",context)[0]).toBe("take Brass key");
 expect(suggestCommands("tlak keeper",context)[0]).toBe("talk Temple guard");
 expect(suggestCommands("zzzzzzzzzz",context)).toEqual([]);
 expect(suggestCommands("",context)).toEqual([]);
});
