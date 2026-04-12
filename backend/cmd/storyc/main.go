package main

import (
	"encoding/json"
	"fmt"
	"os"

	"temple-adventure/storygen"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: storyc <story.json> [story2.json ...]\n")
		os.Exit(1)
	}

	exitCode := 0
	for _, path := range os.Args[1:] {
		if !compileStory(path) {
			exitCode = 1
		}
		fmt.Println()
	}
	os.Exit(exitCode)
}

func compileStory(path string) bool {
	fmt.Printf("%s%s=== Compiling: %s ===%s\n", colorBold, colorCyan, path, colorReset)

	data, err := os.ReadFile(path)
	if err != nil {
		printError("failed to read file: %v", err)
		return false
	}

	var spec storygen.StorySpec
	if err := json.Unmarshal(data, &spec); err != nil {
		printError("invalid JSON: %v", err)
		return false
	}

	fmt.Printf("  Story: %s (%s)\n", spec.Title, spec.Slug)

	// Stage 1: Structural validation
	fmt.Printf("\n  %sStage 1: Structural validation%s\n", colorBold, colorReset)
	specErrs := storygen.ValidateSpec(&spec)
	if len(specErrs) > 0 {
		for _, e := range specErrs {
			printError("%s", e)
		}
		fmt.Printf("\n  %s%sFAIL%s — %d structural error(s), cannot continue\n", colorBold, colorRed, colorReset, len(specErrs))
		return false
	}
	printOk("structural validation passed")

	// Stage 2: Expansion
	fmt.Printf("\n  %sStage 2: Expansion%s\n", colorBold, colorReset)
	world, err := storygen.Expand(&spec)
	if err != nil {
		printError("expansion failed: %v", err)
		return false
	}
	printOk("expanded (%d rooms, %d items, %d NPCs, %d puzzles)",
		len(world.Rooms), len(world.Items), len(world.Npcs), len(world.Puzzles))

	// Stage 3: Deep validation (expanded world)
	fmt.Printf("\n  %sStage 3: Deep validation%s\n", colorBold, colorReset)
	deepErrs := storygen.ValidateWorldDeep(world, spec.StartRoom)
	if len(deepErrs) > 0 {
		for _, e := range deepErrs {
			printError("%s", e)
		}
		fmt.Printf("\n  %s%sFAIL%s — %d deep validation error(s)\n", colorBold, colorRed, colorReset, len(deepErrs))
		return false
	}
	printOk("deep validation passed")

	// Stage 4: Gameplay validation (graph analysis)
	fmt.Printf("\n  %sStage 4: Gameplay analysis%s\n", colorBold, colorReset)
	gameplay := storygen.ValidateGameplay(&spec)
	for _, e := range gameplay.Errors {
		printError("%s", e)
	}
	for _, w := range gameplay.Warnings {
		printWarning("%s", w)
	}
	if len(gameplay.Errors) == 0 && len(gameplay.Warnings) == 0 {
		printOk("all gameplay checks passed")
	} else if len(gameplay.Errors) == 0 {
		printOk("gameplay checks passed with %d warning(s)", len(gameplay.Warnings))
	}

	// Summary
	totalErrors := len(gameplay.Errors)
	totalWarnings := len(gameplay.Warnings)

	fmt.Println()
	if totalErrors > 0 {
		fmt.Printf("  %s%sFAIL%s — %d error(s), %d warning(s)\n", colorBold, colorRed, colorReset, totalErrors, totalWarnings)
		return false
	}
	fmt.Printf("  %s%sPASS%s — 0 errors, %d warning(s)\n", colorBold, colorGreen, colorReset, totalWarnings)
	return true
}

func printOk(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("    %s✓%s %s\n", colorGreen, colorReset, msg)
}

func printError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("    %s✗%s %s\n", colorRed, colorReset, msg)
}

func printWarning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("    %s⚠%s %s\n", colorYellow, colorReset, msg)
}
