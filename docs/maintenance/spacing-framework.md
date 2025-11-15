# Spacing Framework

## Core Principle
**Each output section ends with exactly one `\n`. When transitioning between sections, add one `\n` to create a blank line.**

## Rules

### 1. Progress Bar Component (`progress_bar.go`)
- **Title line**: Starts with `\n`, ends with `\n` (creates space before items)
- **Items**: Each item ends with `\n`
- **Final output**: Always ends with exactly one `\n` (from last item)
- **Never add trailing `\n\n`** - let commands handle spacing after the progress bar

### 2. After Progress Bar (in commands)
- **If there are follow-up messages** ("not found", status report, etc.):
  - Add `\n` before the first follow-up message to create blank line
  - Each follow-up section ends with `\n`
- **If there are NO follow-up messages**:
  - Add `\n` after progress bar completes to create blank line before prompt

### 3. Status Report (`PrintStatusReport`)
- Starts with `\n` (creates blank line before it)
- Ends with `\n\n` (blank line after it, ready for prompt)
- Only shown in verbose mode

### 4. Error Messages / "Not Found" Messages
- Each message line ends with `\n`
- Add `\n` before the first message to create blank line from previous section
- Add `\n` after the last message if there's another section after it

### 5. Verbose Output
- Each verbose info line ends with `\n` (handled by `output.GetVerbose().Info()`)
- Add `\n` before verbose output if it follows another section
- Add `\n` after verbose output if it's followed by progress bar

## Examples

### Normal install (no errors, no verbose)
```
Progress bar output (ends with \n)
[Add \n here]
Prompt
```

### Install with "not found" fonts
```
Progress bar output (ends with \n)
[Add \n here]
"not found" messages (each ends with \n)
[Add \n here if status report follows, otherwise prompt]
```

### Verbose install
```
Verbose info (ends with \n)
[Add \n here]
Progress bar output (ends with \n)
[Add \n here if status report follows]
Status report (ends with \n\n)
Prompt
```

