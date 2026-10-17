# Duplicate detection across package managers

Date: 2026-10-17 · Phase 2 headline feature · Status: approved by the owner section by section

## Goal

Show the user when the same application is installed through more than one package manager (pacman and Flatpak, brew and npm, pip and pipx, `go install` and pacman, …), so they can spot redundant installs and wasted space. pkgmate is a view-only inventory tool: it reports duplicates and never removes anything.

Out of scope: "which copy runs first on `PATH`" (binary shadowing), user-editable aliases, removing or fixing duplicates.

## Decisions

| Topic | Decision |
|---|---|
| What counts as "the same" | Same application by package identity, not by binaries on `PATH`. |
| Matching | Per-ecosystem normalisation rules plus a small built-in alias table. No fuzzy matching. |
| Scope | A group is reported only when it spans at least two managers and at least one copy was installed explicitly (`IsDirect`). |
| Display | A new "Duplicates" tab, plus a `(+manager)` marker in the Manager column of the other tabs, plus a footer count. |
| Architecture | Pure functions in `backend/identity.go`, run once when loading finishes. |

## 1. Matching

### Package identity

`Package` gains one field:

```go
ID string // native identifier when it differs from Name (Flatpak application ID); empty otherwise
```

`parseFlatpakList` keeps the application column (`fields[0]`, today discarded) as `ID`. `getFlatpakInstallDate` is called with the ID instead of the display name (fixes the item in #116).

### Keys

`IdentityKeys(p Package) []string` returns one or two typed keys. Normalisation for every key: lowercase, and `_` is treated as `-`.

| Source (`DB`) | Keys |
|---|---|
| `pacman`, `pacman/AUR` | `python-X` → `py:X`; otherwise `app:<name>` with a trailing `-bin`, `-git` or `-appimage` removed |
| `apt/dpkg` | `python3-X` → `py:X`; otherwise `app:<name>` |
| `brew`, `brew/Cask`, `snap` | `app:<name>` |
| `pip` | `py:<name>` |
| `pipx` | `py:<name>` and `app:<name>` |
| `npm` | `app:<name>` (global packages are matched by name; the owner accepted the occasional false positive) |
| `golang` | `app:<last path element of the module path>` (`golang.org/x/tools/gopls` → `gopls`) |
| `flatpak` | alias table on `ID` first; otherwise `app:<last dot-separated segment of ID>`; the generic segments `client`, `desktop`, `app`, `application` produce no key unless aliased |

### Alias table

A curated `map[string]string` in `backend/identity.go` from a source-specific identifier to a key, starting with roughly 15–20 entries for well-known apps whose identifiers do not normalise to the same name (for example `com.spotify.Client` → `app:spotify`, `org.telegram.desktop` → `app:telegram-desktop`, `com.google.Chrome` → `app:google-chrome`, `com.visualstudio.code` → `app:code`). Every entry has a test case. Not user-editable in this phase.

### Grouping

`FindDuplicates(pkgs []Package) []DuplicateGroup`:

1. Compute keys for every package; union packages that share any key (union-find), so a package with two keys joins both groups into one (pipx `black` links pacman `python-black` via `py:black` and brew `black` via `app:black`).
2. Manager families: `pacman` + `pacman/AUR` are one manager; `brew` + `brew/Cask` are one manager; every other `DB` is its own manager (pip and pipx are different).
3. Keep a group only if it contains at least two manager families and at least one package with `IsDirect`.
4. Within a group, order packages by manager name; order groups by combined `Size`, largest first, then by name.

```go
type DuplicateGroup struct {
    Key      string    // a representative key, e.g. "app:firefox"
    Packages []Package // two or more, at least two manager families
    Size     int64     // sum of Packages' Size
}
```

## 2. UI

### Row identity fix (prerequisite)

`customTable` keys rows and row styles by the first cell (the name). Two packages with the same name from different managers overwrite each other today (npm `typescript` and pacman `typescript` collapse into one row in "All Packages"). Rows and styles are keyed by manager + name instead. Re-sent packages (pacman's fresh update re-emit) still replace their own row.

### Collecting and computing

- `tableModel` keeps the latest copy of every loaded package, keyed by manager + name.
- On `FinishedLoadingPackagesMsg` it calls `backend.FindDuplicates` once and applies the result.

### Duplicates tab

- A fifth tab, "Duplicates", with the same columns (Name, Version, Manager, Size, Installed).
- Copies of the same app are on adjacent rows; groups follow `FindDuplicates`' order (largest combined size first).
- Search, cursor and scrolling behave as in the other tabs.
- While loading it shows the same loading state as the other tabs; after loading with no groups it shows "No duplicates found".

### Marker in the other tabs

For a package that belongs to a group, the Manager column shows the other managers in the group: `pacman (+flatpak)`, `npm (+pacman, brew)`. Searching the Manager column for `pacman` still matches.

### Footer

The footer shows the number of groups, e.g. "3 apps installed more than once", when there is at least one group.

## 3. Edge cases

- Same manager family twice (pacman + pacman/AUR, brew + brew/Cask, dpkg multi-arch `libc6` amd64 + i386) is never a duplicate.
- A group where every copy is a dependency is dropped.
- Generic Flatpak ID segments never match unless aliased.
- Duplicates are recomputed on every load; there is no other refresh.
- Cost: one pass with maps; negligible for thousands of packages.

## 4. Testing

- `backend/identity_test.go`: table tests for `IdentityKeys` covering every source and rule, one case per alias entry, and the generic-segment skip; `FindDuplicates` tests for the union-find merge, the manager-family rule, the at-least-one-direct rule and the ordering.
- `backend/identity_e2e_test.go` (tagged `all_backends`): runs the existing fixtures in `backend/testdata/` through the parsers and then `FindDuplicates`; fixtures get a few overlapping packages added so real groups are asserted.
- `ui` tests: two rows with the same name and different managers both survive; the marker text; the Duplicates tab filled from given groups without touching the `Program` global.
- The existing gate stays: `go build/vet/test -race -tags=all_backends`, untagged `go test`, `gofmt`.

## 5. Docs

README: a short "Duplicates" subsection under "What it does" describing the tab, the marker and the matching rules (including that npm global packages match by name), and the new tab in the tab list.
