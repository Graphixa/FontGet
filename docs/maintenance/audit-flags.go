package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FlagInfo represents information about a command flag
type FlagInfo struct {
	Command  string
	Flag     string
	Short    string
	Type     string
	Default  string
	Usage    string
	IsGlobal bool
}

// CommandInfo represents information about a command
type CommandInfo struct {
	Name        string
	Subcommands []string
	Flags       []FlagInfo
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run audit-flags.go <cmd-directory>")
		fmt.Println("Example: go run audit-flags.go cmd/")
		os.Exit(1)
	}

	cmdDir := os.Args[1]

	fmt.Println("🔍 FontGet CLI Flag Audit")
	fmt.Println("=========================")
	fmt.Println()

	// Find all Go files in the cmd directory
	files, err := findGoFiles(cmdDir)
	if err != nil {
		fmt.Printf("Error finding Go files: %v\n", err)
		os.Exit(1)
	}

	// Parse all command files
	commands := make(map[string]*CommandInfo)

	for _, file := range files {
		cmdInfo, err := parseCommandFile(file)
		if err != nil {
			fmt.Printf("Error parsing %s: %v\n", file, err)
			continue
		}

		if cmdInfo != nil {
			commands[cmdInfo.Name] = cmdInfo
		}
	}

	// Print audit results
	printAuditResults(commands)

	// Check documentation sync
	checkDocumentationSync(commands)
}

func findGoFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

func parseCommandFile(filePath string) (*CommandInfo, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var cmdInfo *CommandInfo
	isRootFile := strings.Contains(filePath, "root.go")

	// Look for command definitions and flag registrations
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.GenDecl:
			// Look for command variable declarations
			for _, spec := range x.Specs {
				if vspec, ok := spec.(*ast.ValueSpec); ok {
					for _, name := range vspec.Names {
						if strings.HasSuffix(name.Name, "Cmd") {
							cmdName := strings.TrimSuffix(name.Name, "Cmd")
							if cmdName == "root" {
								cmdName = "global"
							}
							cmdInfo = &CommandInfo{
								Name:        cmdName,
								Subcommands: []string{},
								Flags:       []FlagInfo{},
							}
						}
					}
				}
			}
		case *ast.CallExpr:
			// Look for flag registrations
			if cmdInfo != nil {
				parseFlagRegistration(x, cmdInfo, filePath, isRootFile)
			}
		}
		return true
	})

	return cmdInfo, nil
}

func parseFlagRegistration(call *ast.CallExpr, cmdInfo *CommandInfo, filePath string, isRootFile bool) {
	// Look for .Flags().StringP(), .Flags().BoolP(), etc.
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if sel.Sel.Name == "StringP" || sel.Sel.Name == "BoolP" || sel.Sel.Name == "IntP" ||
			sel.Sel.Name == "StringVarP" || sel.Sel.Name == "BoolVarP" || sel.Sel.Name == "IntVarP" ||
			sel.Sel.Name == "StringVar" || sel.Sel.Name == "BoolVar" || sel.Sel.Name == "IntVar" {

			flag := FlagInfo{
				Command: cmdInfo.Name,
				Type:    sel.Sel.Name,
			}

			// Handle different argument orders for different flag types
			if sel.Sel.Name == "StringVarP" || sel.Sel.Name == "BoolVarP" || sel.Sel.Name == "IntVarP" {
				// VarP functions: (pointer, longName, shortName, defaultValue, usage)
				if len(call.Args) >= 5 {
					// Extract flag name (2nd argument)
					if lit, ok := call.Args[1].(*ast.BasicLit); ok {
						flag.Flag = strings.Trim(lit.Value, "\"")
					}
					// Extract short flag (3rd argument)
					if lit, ok := call.Args[2].(*ast.BasicLit); ok {
						flag.Short = strings.Trim(lit.Value, "\"")
					}
					// Extract default value (4th argument)
					if lit, ok := call.Args[3].(*ast.BasicLit); ok {
						flag.Default = strings.Trim(lit.Value, "\"")
					}
					// Extract usage (5th argument)
					if lit, ok := call.Args[4].(*ast.BasicLit); ok {
						flag.Usage = strings.Trim(lit.Value, "\"")
					}
				}
			} else if sel.Sel.Name == "StringVar" || sel.Sel.Name == "BoolVar" || sel.Sel.Name == "IntVar" {
				// Var functions: (pointer, longName, defaultValue, usage)
				if len(call.Args) >= 4 {
					// Extract flag name (2nd argument)
					if lit, ok := call.Args[1].(*ast.BasicLit); ok {
						flag.Flag = strings.Trim(lit.Value, "\"")
					}
					// No short flag for Var functions
					flag.Short = ""
					// Extract default value (3rd argument)
					if lit, ok := call.Args[2].(*ast.BasicLit); ok {
						flag.Default = strings.Trim(lit.Value, "\"")
					}
					// Extract usage (4th argument)
					if lit, ok := call.Args[3].(*ast.BasicLit); ok {
						flag.Usage = strings.Trim(lit.Value, "\"")
					}
				}
			} else {
				// P functions: (longName, shortName, defaultValue, usage)
				if len(call.Args) >= 4 {
					// Extract flag name (1st argument)
					if lit, ok := call.Args[0].(*ast.BasicLit); ok {
						flag.Flag = strings.Trim(lit.Value, "\"")
					}
					// Extract short flag (2nd argument)
					if lit, ok := call.Args[1].(*ast.BasicLit); ok {
						flag.Short = strings.Trim(lit.Value, "\"")
					}
					// Extract default value (3rd argument)
					if lit, ok := call.Args[2].(*ast.BasicLit); ok {
						flag.Default = strings.Trim(lit.Value, "\"")
					}
					// Extract usage (4th argument)
					if lit, ok := call.Args[3].(*ast.BasicLit); ok {
						flag.Usage = strings.Trim(lit.Value, "\"")
					}
				}
			}

			// Check if it's a global flag (PersistentFlags or root file)
			flag.IsGlobal = isRootFile

			cmdInfo.Flags = append(cmdInfo.Flags, flag)
		}
	}
}

func printAuditResults(commands map[string]*CommandInfo) {
	fmt.Println("📋 Commands and Flags Found:")
	fmt.Println()

	// Sort commands for consistent output
	var cmdNames []string
	for name := range commands {
		cmdNames = append(cmdNames, name)
	}
	sort.Strings(cmdNames)

	for _, name := range cmdNames {
		cmd := commands[name]
		fmt.Printf("🔹 %s\n", name)

		if len(cmd.Flags) == 0 {
			fmt.Println("   No flags found")
		} else {
			for _, flag := range cmd.Flags {
				scope := "local"
				if flag.IsGlobal {
					scope = "global"
				}

				shortFlag := ""
				if flag.Short != "" {
					shortFlag = fmt.Sprintf(" (-%s)", flag.Short)
				}

				fmt.Printf("   --%s%s [%s] (%s) - %s\n",
					flag.Flag, shortFlag, flag.Type, scope, flag.Usage)
			}
		}
		fmt.Println()
	}

	// Summary
	totalCommands := len(commands)
	totalFlags := 0
	globalFlags := 0

	for _, cmd := range commands {
		for _, flag := range cmd.Flags {
			totalFlags++
			if flag.IsGlobal {
				globalFlags++
			}
		}
	}

	fmt.Printf("📊 Summary: %d commands, %d total flags (%d global, %d local)\n",
		totalCommands, totalFlags, globalFlags, totalFlags-globalFlags)
	fmt.Println()
}

func checkDocumentationSync(commands map[string]*CommandInfo) {
	fmt.Println("📚 Documentation Sync Check:")
	fmt.Println()

	// Read the current usage.md
	docPath := "docs/usage.md"
	content, err := os.ReadFile(docPath)
	if err != nil {
		fmt.Printf("❌ Could not read %s: %v\n", docPath, err)
		return
	}

	docContent := string(content)

	// Check for missing flags in documentation
	missingFlags := []string{}

	for _, cmd := range commands {
		// Find the command section in documentation
		// Look for "## `commandname`" or "## `command-name`" pattern
		cmdSectionStart := -1
		cmdSectionEnd := -1

		// Try different command name formats
		cmdPatterns := []string{
			fmt.Sprintf("## `%s`", cmd.Name),
			fmt.Sprintf("## `%s`", strings.ToLower(cmd.Name)),
		}

		// Handle special case for "global" command (root flags)
		if cmd.Name == "global" {
			cmdPatterns = []string{"## Global Flags"}
		}

		for _, pattern := range cmdPatterns {
			idx := strings.Index(docContent, pattern)
			if idx != -1 {
				cmdSectionStart = idx
				break
			}
		}

		// If we found the section, find where it ends (next ## or end of file)
		if cmdSectionStart != -1 {
			// Find the next section header (##) after this one
			remaining := docContent[cmdSectionStart+10:] // Skip past "## `cmd`\n"
			nextSection := strings.Index(remaining, "\n## ")
			if nextSection != -1 {
				cmdSectionEnd = cmdSectionStart + 10 + nextSection
			} else {
				cmdSectionEnd = len(docContent)
			}
		}

		// Check each flag for this command
		for _, flag := range cmd.Flags {
			flagPattern := fmt.Sprintf("--%s", flag.Flag)
			found := false

			if cmdSectionStart != -1 && cmdSectionEnd != -1 {
				// Check only within the command's section
				sectionContent := docContent[cmdSectionStart:cmdSectionEnd]
				found = strings.Contains(sectionContent, flagPattern)
			} else {
				// Fallback: check entire document (original behavior)
				found = strings.Contains(docContent, flagPattern)
			}

			if !found {
				missingFlags = append(missingFlags, fmt.Sprintf("%s: --%s", cmd.Name, flag.Flag))
			}
		}
	}

	if len(missingFlags) == 0 {
		fmt.Println("✅ All flags are documented!")
	} else {
		fmt.Println("⚠️  Missing flags in documentation:")
		for _, missing := range missingFlags {
			fmt.Printf("   - %s\n", missing)
		}
	}

	fmt.Println()
	fmt.Println("💡 To update documentation, run this script and manually update docs/usage.md")
	fmt.Println("   with any missing flags found above.")
}
