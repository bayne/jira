# jiracmd — Command implementations

~40 files implementing ~70 registered CLI commands. All commands registered in `registry.go` via `RegisterAllCommands()`.

## File pattern

Every command file follows the same structure:
1. `type Cmd<Name>Options struct` — command-specific options (embeds `jiracli.CommonOptions`)
2. `func CmdRegistry<Name>()` — registers the command with kingpin, returns `CommandRegistryEntry`
3. `func CmdUsage<Name>()` — returns usage string
4. `func Cmd<Name>(...)` — implements the command logic

## Key commands by category

**Issue lifecycle**: `create.go`, `edit.go`, `view.go`, `list.go`, `transition.go`, `subtask.go`
**Assignment**: `assign.go`, `take.go`, `unassign.go`
**Sprint analytics**: `currentSprint.go`, `previousSprint.go`, `nextSprint.go`, `sprints.go`
**Issue linking**: `block.go`, `dup.go`, `issuelink.go`, `issuelinktypes.go`
**Labels**: `labelsAdd.go`, `labelsRemove.go`, `labelsSet.go`
**Epics**: `epicAdd.go`, `epicCreate.go`, `epicList.go`, `epicRemove.go`
**Attachments**: `attachCreate.go`, `attachGet.go`, `attachList.go`, `attachRemove.go`
**Metadata inspection**: `fields.go`, `fieldsmap.go`, `createmeta.go`, `editmeta.go`, `issuetypes.go`, `transitions.go`
**Graph traversal**: `crawl.go` — BFS crawler of linked issues with depth limiting and cycle detection
**Auth**: `login.go`, `logout.go`, `session.go`
**Templates**: `exportTemplates.go`, `unexportTemplates.go`
**Worklogs**: `worklogAdd.go`, `worklogList.go`
**Raw API**: `request.go` — pass-through to arbitrary Jira REST endpoints

## Notable implementations

- `crawl.go`: Pure-function BFS graph traversal (`crawlGraph`) with load/fetch/save callbacks. `crawlLinkedKeys` extracts linked issue keys from untyped JSON maps. Handles cycles via visited set, configurable depth, relationship type toggles (links/subtasks/parents).
- `currentSprint.go`: Computes point distribution per "first assignee" by walking issue changelog via `resolveFirstAssignee()`. `hasMultipleSprints()` filters out rollover issues.
- `edit.go`: `fixGDPRUserFields()` handles Cloud GDPR user field migration (name -> accountId).
- `fieldmaputil.go`: `applyFieldMappings()` transforms field keys in `IssueUpdate` before API submission.

## Tests
`crawl_test.go`, `previousSprint_test.go`. Run with `go test ./jiracmd/`.
