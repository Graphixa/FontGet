# Verbose and debug — guidelines

Rules for **styled terminal output** via `output.GetVerbose()` and `output.GetDebug()`. File logging uses `GetLogger()` and is defined in [logging-guidelines.md](logging-guidelines.md).

## `--verbose`

Use when the user should see **operational** detail without internal implementation noise.

**Include:** paths, installation scope, source names, counts, high-level progress, configuration values the user set.

**Avoid:** bare function names, stack traces, raw internal state dumps.

**Errors:** Where this document and command behavior require it, show the raw error (styled); keep wrappers minimal.

`internal/output` only shows verbose when verbose mode is on **and** debug is off (`verbose && !debug`). With `--debug` set, verbose output is not shown—use debug output instead.

## `--debug`

Use when the reader needs **technical** detail to diagnose behavior.

**Include:** URLs, temp directories, which branch or helper ran, download fallback steps, subsystem or function context that locates failure, wrapped errors with enough context to trace code.

**Avoid:** user-facing success copy that duplicates normal UI; repeating the same line as verbose would use.

## Duplication

- Do not emit the same lifecycle message through both `GetVerbose()` and `GetDebug()` for one step.
- Do not rely on `GetLogger()` console mirroring to stand in for `GetVerbose()` / `GetDebug()` for styled CLI output.
- Timestamped file-log lines on the terminal (if mirroring is enabled) are not a substitute for styled verbose/debug; avoid stacking them as duplicate stories for the same event.

## Relationship to file log

- **`GetLogger()`**: required for persistent log file content (start, parameters, errors, completion). Not controlled by `--verbose` / `--debug`.
- **`GetVerbose()` / `GetDebug()`**: required for styled CLI output. Not a replacement for the log file.

## Errors (terminal)

| Mode | Presentation |
|------|----------------|
| Default | Short, user-readable message |
| `--verbose` | Raw error where these guidelines require it, minimal wrapper |
| `--debug` | Technical context: subsystem, wrapped error, enough to locate the failure |

## Spacing

After a block of verbose lines, use `output.GetVerbose().EndSection()` or a single blank line only when verbose was shown. See [spacing-guidelines.md](spacing-guidelines.md).

```go
output.GetVerbose().Info("Scope: %s", scope)
output.GetVerbose().Info("Removing %d font(s)", count)
if IsVerbose() {
    fmt.Println()
}
```
