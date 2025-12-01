# Component-Style Mapping

**Date:** Generated during styles cleanup
**Purpose:** Map each UI component to the styles it uses

## Component Inventory

### 1. Tables (Search, List, Sources)

**Files:**
- `cmd/search.go` - Font search results table
- `cmd/list.go` - Installed fonts list table
- `cmd/add.go` - Font selection table
- `cmd/remove.go` - Font removal table
- `cmd/sources.go` - Sources info table

**Styles Used:**
- `TableHeader` - Column headers
- `TableSourceName` - Font names and source names
- `TableRow` - ❌ UNUSED (not actually used)
- `TableSelectedRow` - ⚠️ Should be `TableRowSelected` for clarity

**Issues:**
- `TableRow` is defined but never used
- `TableSelectedRow` name is ambiguous

**Recommendations:**
- Remove `TableRow` if truly unused
- Rename `TableSelectedRow` → `TableRowSelected`
- Create dedicated table styles if needed

---

### 2. Forms (Sources Manage, Config Edit)

**Files:**
- `cmd/sources_manage.go` - Source management form
- `internal/components/form.go` - Generic form component

**Styles Used:**
- `FormLabel` - Field labels
- `FormInput` - Input field content
- `FormPlaceholder` - Placeholder text
- `FormReadOnly` - Read-only field content
- `TableSelectedRow` - ⚠️ MISUSED for checkbox selection

**Issues:**
- `TableSelectedRow` is reused for checkbox selection (should be `CheckboxItemSelected`)

**Recommendations:**
- Create `CheckboxItemSelected` style
- Create `CheckboxItemNormal` style
- Create `CheckboxChecked` style
- Create `CheckboxUnchecked` style
- Create `CheckboxCursor` style

---

### 3. Cards (Font Info)

**Files:**
- `internal/components/card.go` - Card component
- `cmd/sources.go` - Source info cards

**Styles Used:**
- `CardTitle` - Card titles
- `CardLabel` - Labels within cards
- `CardContent` - Content within cards
- `CardBorder` - Card borders
- `CardContainer` - ❌ UNUSED

**Issues:**
- `CardContainer` is defined but never used

**Recommendations:**
- Remove `CardContainer` if truly unused
- All other card styles are properly used

---

### 4. Progress Bars

**Files:**
- `internal/components/progress_bar.go` - Progress bar component

**Styles Used:**
- `GetProgressBarGradient()` - ✅ Used
- `ProgressBarHeader` - ❌ UNUSED
- `ProgressBarText` - ❌ UNUSED
- `ProgressBarContainer` - ❌ UNUSED

**Issues:**
- Three progress bar styles are defined but never used
- Only the gradient function is actually used

**Recommendations:**
- Remove unused progress bar styles
- Keep `GetProgressBarGradient()` as it's used

---

### 5. Spinners

**Files:**
- `internal/ui/components.go` - Spinner component

**Styles Used:**
- `SpinnerColor` - ✅ Used
- `SpinnerDoneColor` - ✅ Used
- `PinColorMap` - ✅ Used

**Issues:**
- None - all spinner styles are properly used

**Recommendations:**
- Keep as-is

---

### 6. Confirm Dialogs

**Files:**
- `internal/components/confirm.go` - Confirmation dialog

**Styles Used:**
- `PageTitle` - Dialog title
- `FeedbackText` - Dialog message
- `CommandKey` - Keyboard shortcuts
- `CommandLabel` - Button labels
- `TableSourceName` - Item names in delete confirmations

**Issues:**
- None - styles are used appropriately

**Recommendations:**
- Keep as-is

---

### 7. Onboarding Steps

**Files:**
- `internal/onboarding/steps.go` - Onboarding flow

**Styles Used:**
- `PageTitle` - Step titles
- `FeedbackText` - Regular text
- `FeedbackInfo` - Informational text
- `FeedbackWarning` - Warnings
- `FeedbackSuccess` - Success messages
- `FeedbackError` - Error messages
- `ContentHighlight` - Highlighted content
- `TableSourceName` - Source names
- `CommandExample` - Example commands

**Issues:**
- None - styles are used appropriately

**Recommendations:**
- Will need new styles for buttons, checkboxes, and switches (future work)

---

### 8. Command Output (Various Commands)

**Files:**
- `cmd/add.go`, `cmd/remove.go`, `cmd/update.go`, `cmd/version.go`, etc.

**Styles Used:**
- `PageTitle` - Command titles
- `FeedbackInfo` - Informational messages
- `FeedbackText` - Regular text
- `FeedbackWarning` - Warnings
- `FeedbackError` - Error messages
- `FeedbackSuccess` - Success messages
- `ContentHighlight` - Highlighted content
- `CommandExample` - Example commands
- `ReportTitle` - Status reports

**Issues:**
- None - styles are used appropriately

**Recommendations:**
- Keep as-is

---

## Style Reuse Analysis

### Problematic Reuse

1. **`TableSelectedRow`** is used for:
   - ✅ Table row selection (intended)
   - ❌ Checkbox item selection in `sources_manage.go` (misuse)

**Solution:** Create dedicated checkbox styles

### Acceptable Reuse

1. **`PageTitle`** is used for:
   - Page titles
   - Dialog titles
   - Component titles
   - ✅ This is acceptable - it's a generic page structure style

2. **`FeedbackText`** is used for:
   - Regular text in various contexts
   - ✅ This is acceptable - it's a generic text style

3. **`TableSourceName`** is used for:
   - Font names in tables
   - Source names in various contexts
   - ✅ This is acceptable - it's for highlighting names/identifiers

## Component-Style Dependencies

```
Tables
  ├── TableHeader
  ├── TableSourceName
  └── TableRowSelected (rename from TableSelectedRow)

Forms
  ├── FormLabel
  ├── FormInput
  ├── FormPlaceholder
  └── FormReadOnly

Cards
  ├── CardTitle
  ├── CardLabel
  ├── CardContent
  └── CardBorder

Checkboxes (NEW - to be created)
  ├── CheckboxChecked
  ├── CheckboxUnchecked
  ├── CheckboxItemSelected
  ├── CheckboxItemNormal
  └── CheckboxCursor

Buttons (NEW - to be created)
  ├── ButtonNormal
  ├── ButtonSelected
  └── ButtonGroup

Switches (NEW - to be created)
  ├── SwitchContainer
  ├── SwitchLeftSelected
  ├── SwitchLeftNormal
  ├── SwitchRightSelected
  ├── SwitchRightNormal
  └── SwitchSeparator
```

## Summary

- **Total Components:** 8
- **Components with Issues:** 3 (Tables, Forms, Progress Bars)
- **Unused Styles:** 6
- **Misused Styles:** 1 (`TableSelectedRow`)
- **Styles Needing Creation:** 15 (for buttons, checkboxes, switches)

