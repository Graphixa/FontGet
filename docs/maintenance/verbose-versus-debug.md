In a CLI program, the --verbose and --debug flags serve distinct purposes in controlling the level of output information:

`--verbose` flag:
Target Audience: General users who want more insight into the program's execution.
Purpose: To provide a more detailed and descriptive explanation of the program's operations and progress.
Content: This typically includes information like:
- Detailed transaction logs.
- Progress updates during longer operations.
- Raw error messages directly from the interpreter (no wrapper text, just the error as-is).
- Contextual details about actions being performed.
- Error messages should be displayed in red (FeedbackError style) for visibility.
Analogy: Similar to a detailed log or a commentary explaining what the program is doing from a user's perspective.

Note: Normal users (without --verbose) still receive friendly, user-friendly error messages. Verbose mode shows the raw error for users who want to see exactly what went wrong.

`--debug` flag:
Target Audience: Developers and advanced users for troubleshooting and debugging.
Purpose: To expose low-level, internal details about the program's execution for diagnostic purposes.
Content: This often includes:
- Internal execution steps and function calls.
- Values of internal variables.
- Detailed stack traces in case of errors.
- Information about system interactions or resource usage.
- Raw error objects and internal error states.
Analogy: Similar to a trace log or a developer's console, revealing the "under the hood" workings of the program.
Key Differences:
Level of Detail: --verbose provides a higher-level, user-friendly explanation, while --debug delves into the low-level implementation details.
Intended Use: --verbose enhances understanding for general use, while --debug is primarily for problem identification and resolution during development or advanced troubleshooting.
Output Volume: --debug output is typically significantly more extensive and technical than --verbose output.
Error Handling: 
- Normal users (no flags): Receive friendly, user-friendly error messages that describe what went wrong in plain language (e.g., "unable to load font repository: ...", "unable to access system fonts: ...").
- --verbose: Shows raw error messages directly from the interpreter (no wrapper text, just the error as-is, styled in red).
- --debug: Shows technical error details including function names, stack traces, and raw error objects.

Examples:
- Normal user sees: "unable to access system fonts: permission denied"
- Verbose user sees: "permission denied" (raw error, styled in red)
- Debug user sees: "platform.NewFontManager() failed: permission denied" (with function name and technical context)