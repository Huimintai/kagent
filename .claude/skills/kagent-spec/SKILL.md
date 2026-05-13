---
name: kagent-spec
description: >
  Full-repo CLAUDE.md consistency scanner. Verifies all per-directory CLAUDE.md files match
  current code state. Detects drift and refreshes stale sections in-place.
user-invocable: true
argument-hint: "[path] — optional: specific directory or CLAUDE.md to check (default: all)"
---

# kagent-spec — CLAUDE.md Consistency Scanner

Scans all per-directory CLAUDE.md files and verifies they match the current code.
Strategy: **REFRESH** — rewrite stale sections with current state. No changelog, no timestamps.

---

## Rule: Where CLAUDE.md Must Exist

A directory gets its own CLAUDE.md if and only if it contains at least one subdirectory.
Leaf directories (files only, no subdirs) are described by their parent's CLAUDE.md.

---

## Workflow

### Step 1: Discovery

Find all CLAUDE.md files and all directories that should have one:

```bash
# All existing CLAUDE.md files (excluding root)
find . -name "CLAUDE.md" -not -path "./CLAUDE.md" -not -path "./node_modules/*" -not -path "./.next/*" -not -path "./.claude/*"

# All directories that SHOULD have a CLAUDE.md (have at least one subdir)
find . -type d -not -path "./node_modules/*" -not -path "./.next/*" -not -path "./.claude/*" -not -path "./.git/*" -not -path "*/__pycache__/*" | while read dir; do
  if find "$dir" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | grep -q .; then
    echo "$dir"
  fi
done
```

Compare the two lists:
- **MISSING**: Directory should have CLAUDE.md but doesn't
- **ORPHAN**: CLAUDE.md exists but directory no longer has subdirs (became a leaf)

### Step 2: Structural Verification

For each CLAUDE.md, verify its claims against the filesystem:

| Check | Method |
|-------|--------|
| Sub-package/sub-module table | `ls` the directory; every listed item must exist; detect unlisted subdirs |
| Key Files table | Verify each listed file exists |
| File role table (leaf-parents) | Verify each .go/.py/.ts file listed exists; detect unlisted source files |
| Module boundaries / import rules | `grep -r "import"` in .go files to verify no forbidden imports |
| Route listings (UI) | Compare against actual `src/app/` directory structure |
| Handler listings | Compare against actual handler .go files |
| Controller listings | Compare against actual controller .go files |
| SQL query listings | Compare against actual .sql files in queries/ |
| Package listings | Compare against actual package directories |

### Step 3: Drift Classification

For each discrepancy, classify:

| Status | Meaning |
|--------|---------|
| `OK` | Section matches code |
| `DRIFT:ADDED` | Code has items not documented in CLAUDE.md |
| `DRIFT:REMOVED` | CLAUDE.md references items that no longer exist |
| `DRIFT:WRONG` | Documented role/description contradicts implementation |
| `MISSING` | Directory needs CLAUDE.md but doesn't have one |
| `ORPHAN` | CLAUDE.md exists but shouldn't (no subdirs) |

### Step 4: Report

Present findings as a summary table:

```markdown
| CLAUDE.md | Status | Issue |
|-----------|--------|-------|
| go/core/internal/controller/CLAUDE.md | DRIFT:ADDED | new_controller.go not documented |
| go/api/CLAUDE.md | OK | — |
| ui/src/app/CLAUDE.md | DRIFT:REMOVED | /settings route listed but directory deleted |
| go/core/internal/newpkg/ | MISSING | Directory has subdirs but no CLAUDE.md |
```

If run with a path argument, scope to that single file/directory.

### Step 5: Refresh

For each drifted CLAUDE.md:

1. Identify the drifted section by `##` heading boundaries
2. Re-scan the source of truth:
   - For sub-package tables: `ls` the directory, read each subdir's purpose from its own CLAUDE.md or source
   - For file tables: `ls *.go/*.py/*.ts`, read file headers/package docs
   - For type lists: parse source for exported type/interface declarations
   - For route lists: scan directory structure
3. Regenerate the section content from scratch
4. Present proposed replacement (show old vs new)
5. After user approval: overwrite the section in the file

**Refresh algorithm:**
```
for each CLAUDE.md with drift:
    read full file content
    for each drifted section:
        find section start (## Heading line)
        find section end (next ## or EOF)
        generate replacement by scanning filesystem + source
        replace old section with new
    write file
```

For MISSING directories:
1. Scan the directory structure
2. Read source files to determine roles
3. Generate a new CLAUDE.md following the template
4. Present for approval, then write

For ORPHAN files:
1. Confirm directory truly has no subdirs
2. Propose deletion
3. Ensure parent CLAUDE.md covers the content
4. Delete after approval

---

## CLAUDE.md Template

```markdown
# <Directory Name>

<One-line purpose>

## Sub-packages

| Package | Role | Dependencies |
|---------|------|-------------|
| name/ | what it does | what it imports from other modules |

## Module Boundaries

- **Imports**: list of modules this code imports from
- **Imported by**: what depends on this
- **NEVER imports**: forbidden dependencies

## Key Types

- `TypeName` — one-line description

## Quick Commands

\`\`\`bash
# relevant commands for this module
\`\`\`
```

Adapt sections based on depth:
- **Top-level** (go/, python/, ui/): Include all sections
- **Mid-level** (go/core/internal/): Sub-packages + Boundaries
- **Leaf-parent** (directory whose children are all leaves): Replace Sub-packages with Files table

---

## Scope Limits

This skill verifies **structural claims only**:
- Files and directories exist/don't exist
- Types and interfaces are declared where documented
- Import rules are followed
- Listings are complete

It does NOT verify behavioral claims ("this function does X") — that requires semantic analysis beyond filesystem scanning.

---

## Quick Commands

```bash
# Run full scan
/kagent-spec

# Check specific module
/kagent-spec go/core/internal/controller

# Check single file
/kagent-spec go/api/CLAUDE.md
```
