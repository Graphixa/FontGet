package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
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

	fmt.Println("FontGet CLI Flag Audit")
	fmt.Println("=========================")
	fmt.Println()

	files, err := findGoFiles(cmdDir)
	if err != nil {
		fmt.Printf("Error finding Go files: %v\n", err)
		os.Exit(1)
	}

	commands := make(map[string]*CommandInfo)

	for _, file := range files {
		if err := parseCommandFile(file, commands); err != nil {
			fmt.Printf("Error parsing %s: %v\n", file, err)
			continue
		}
	}

	printAuditResults(commands)
	checkDocumentationSync(commands)
}

func findGoFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

func ensureCommand(commands map[string]*CommandInfo, name string) *CommandInfo {
	if cmd, ok := commands[name]; ok {
		return cmd
	}
	cmd := &CommandInfo{Name: name, Subcommands: []string{}, Flags: []FlagInfo{}}
	commands[name] = cmd
	return cmd
}

func cmdNameFromIdent(name string) string {
	name = strings.TrimSuffix(name, "Cmd")
	if name == "root" {
		return "global"
	}
	return name
}

func isCobraCommandSpec(vspec *ast.ValueSpec) bool {
	// var fooCmd = &cobra.Command{...}
	for _, value := range vspec.Values {
		unary, ok := value.(*ast.UnaryExpr)
		if !ok || unary.Op != token.AND {
			continue
		}
		comp, ok := unary.X.(*ast.CompositeLit)
		if !ok {
			continue
		}
		if sel, ok := comp.Type.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "cobra" && sel.Sel.Name == "Command" {
				return true
			}
		}
	}
	// var fooCmd *cobra.Command (typed nil / later assign) — rare; skip non-cobra locals like execCmd
	if sel, ok := vspec.Type.(*ast.StarExpr); ok {
		if s, ok := sel.X.(*ast.SelectorExpr); ok {
			if ident, ok := s.X.(*ast.Ident); ok && ident.Name == "cobra" && s.Sel.Name == "Command" {
				return true
			}
		}
	}
	return false
}

func parseCommandFile(filePath string, commands map[string]*CommandInfo) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	isRootFile := strings.Contains(filepath.Base(filePath), "root.go")
	var fallbackCmd *CommandInfo

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.GenDecl:
			for _, spec := range x.Specs {
				vspec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				if !isCobraCommandSpec(vspec) {
					continue
				}
				for _, name := range vspec.Names {
					if strings.HasSuffix(name.Name, "Cmd") {
						cmdName := cmdNameFromIdent(name.Name)
						fallbackCmd = ensureCommand(commands, cmdName)
					}
				}
			}
		case *ast.CallExpr:
			parseFlagRegistration(x, commands, fallbackCmd, isRootFile)
		}
		return true
	})

	return nil
}

func flagMethodKind(name string) string {
	switch name {
	case "StringP", "BoolP", "IntP",
		"StringVarP", "BoolVarP", "IntVarP",
		"StringVar", "BoolVar", "IntVar",
		"String", "Bool", "Int":
		return name
	default:
		return ""
	}
}

func receiverCmdName(call *ast.CallExpr) (string, bool, bool) {
	// Expect: <cmd>.Flags().Bool(...) or <cmd>.PersistentFlags().Bool(...)
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false, false
	}
	innerCall, ok := sel.X.(*ast.CallExpr)
	if !ok {
		return "", false, false
	}
	innerSel, ok := innerCall.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false, false
	}
	isPersistent := innerSel.Sel.Name == "PersistentFlags"
	if innerSel.Sel.Name != "Flags" && !isPersistent {
		return "", false, false
	}
	ident, ok := innerSel.X.(*ast.Ident)
	if !ok {
		return "", false, false
	}
	return cmdNameFromIdent(ident.Name), isPersistent, true
}

func litString(expr ast.Expr) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok {
		return "", false
	}
	return strings.Trim(lit.Value, "\""), true
}

func parseFlagRegistration(call *ast.CallExpr, commands map[string]*CommandInfo, fallback *CommandInfo, isRootFile bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	method := flagMethodKind(sel.Sel.Name)
	if method == "" {
		return
	}

	cmdName, isPersistent, ok := receiverCmdName(call)
	var cmdInfo *CommandInfo
	if ok {
		cmdInfo = ensureCommand(commands, cmdName)
	} else if fallback != nil {
		cmdInfo = fallback
		cmdName = fallback.Name
	} else {
		return
	}

	flag := FlagInfo{
		Command:  cmdName,
		Type:     method,
		IsGlobal: isRootFile || isPersistent || cmdName == "global",
	}

	switch {
	case strings.HasSuffix(method, "VarP"):
		// (pointer, longName, shortName, defaultValue, usage)
		if len(call.Args) >= 5 {
			flag.Flag, _ = litString(call.Args[1])
			flag.Short, _ = litString(call.Args[2])
			flag.Default, _ = litString(call.Args[3])
			flag.Usage, _ = litString(call.Args[4])
		}
	case strings.HasSuffix(method, "Var"):
		// (pointer, longName, defaultValue, usage)
		if len(call.Args) >= 4 {
			flag.Flag, _ = litString(call.Args[1])
			flag.Default, _ = litString(call.Args[2])
			flag.Usage, _ = litString(call.Args[3])
		}
	case strings.HasSuffix(method, "P"):
		// (longName, shortName, defaultValue, usage)
		if len(call.Args) >= 4 {
			flag.Flag, _ = litString(call.Args[0])
			flag.Short, _ = litString(call.Args[1])
			flag.Default, _ = litString(call.Args[2])
			flag.Usage, _ = litString(call.Args[3])
		}
	default:
		// String/Bool/Int: (longName, defaultValue, usage)
		if len(call.Args) >= 3 {
			flag.Flag, _ = litString(call.Args[0])
			flag.Default, _ = litString(call.Args[1])
			flag.Usage, _ = litString(call.Args[2])
		}
	}

	if flag.Flag == "" {
		return
	}

	cmdInfo.Flags = append(cmdInfo.Flags, flag)
}

func printAuditResults(commands map[string]*CommandInfo) {
	fmt.Println("Commands and Flags Found:")
	fmt.Println()

	var cmdNames []string
	for name := range commands {
		cmdNames = append(cmdNames, name)
	}
	sort.Strings(cmdNames)

	for _, name := range cmdNames {
		cmd := commands[name]
		fmt.Printf("* %s\n", name)

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

	fmt.Printf("Summary: %d commands, %d total flags (%d global, %d local)\n",
		totalCommands, totalFlags, globalFlags, totalFlags-globalFlags)
	fmt.Println()
}

func knownFlagSet(commands map[string]*CommandInfo) map[string]bool {
	known := make(map[string]bool)
	for _, cmd := range commands {
		for _, flag := range cmd.Flags {
			known[flag.Flag] = true
		}
	}
	return known
}

func checkDocumentationSync(commands map[string]*CommandInfo) {
	fmt.Println("Documentation Sync Check:")
	fmt.Println()

	docPath := "docs/usage.md"
	content, err := os.ReadFile(docPath)
	if err != nil {
		fmt.Printf("Could not read %s: %v\n", docPath, err)
		return
	}

	docContent := string(content)
	missingFlags := []string{}

	for _, cmd := range commands {
		cmdSectionStart, cmdSectionEnd := findCommandSection(docContent, cmd.Name)

		for _, flag := range cmd.Flags {
			flagPattern := fmt.Sprintf("--%s", flag.Flag)
			found := false

			if cmdSectionStart != -1 && cmdSectionEnd != -1 {
				sectionContent := docContent[cmdSectionStart:cmdSectionEnd]
				found = strings.Contains(sectionContent, flagPattern)
			} else {
				found = strings.Contains(docContent, flagPattern)
			}

			if !found {
				missingFlags = append(missingFlags, fmt.Sprintf("%s: --%s", cmd.Name, flag.Flag))
			}
		}
	}

	if len(missingFlags) == 0 {
		fmt.Println("All code flags appear in documentation.")
	} else {
		fmt.Println("Missing flags in documentation:")
		for _, missing := range missingFlags {
			fmt.Printf("   - %s\n", missing)
		}
	}

	// Doc-only phantoms: flag bullets in usage.md that are not registered in cmd/
	known := knownFlagSet(commands)
	docFlagRe := regexp.MustCompile(`(?m)^-\s+\x60--([a-zA-Z0-9-]+)`)
	matches := docFlagRe.FindAllStringSubmatch(docContent, -1)
	phantom := []string{}
	seenPhantom := map[string]bool{}
	for _, m := range matches {
		name := m[1]
		if known[name] || seenPhantom[name] {
			continue
		}
		// Skip narrative mentions of invalid usage that are not flag bullets for real flags
		seenPhantom[name] = true
		phantom = append(phantom, name)
	}
	sort.Strings(phantom)

	if len(phantom) == 0 {
		fmt.Println("No doc-only flag bullets found outside code registrations.")
	} else {
		fmt.Println("Doc-only flag bullets (verify these exist in cmd/):")
		for _, name := range phantom {
			fmt.Printf("   - --%s\n", name)
		}
	}

	fmt.Println()
	fmt.Println("To update documentation, fix docs/usage.md for any items listed above.")
}

func findCommandSection(docContent, cmdName string) (start, end int) {
	start = -1
	end = -1

	title := cmdName
	if cmdName != "" {
		title = strings.ToUpper(cmdName[:1]) + cmdName[1:]
	}
	if cmdName == "global" {
		title = "Global Flags"
	}

	patterns := []string{
		"## " + title,
		"## `" + cmdName + "`",
		"## `" + strings.ToLower(cmdName) + "`",
	}
	if cmdName == "global" {
		patterns = []string{"## Global Flags"}
	}

	for _, pattern := range patterns {
		idx := strings.Index(docContent, pattern)
		if idx != -1 {
			start = idx
			break
		}
	}

	if start == -1 {
		return -1, -1
	}

	remaining := docContent[start+len("## "):]
	nextSection := strings.Index(remaining, "\n## ")
	if nextSection != -1 {
		end = start + len("## ") + nextSection
	} else {
		end = len(docContent)
	}
	return start, end
}
