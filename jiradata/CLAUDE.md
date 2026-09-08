# jiradata — Data model types

93 Go files defining types for the Jira REST API. ~10k lines total.

## Generated files (79 files)

Files with `DO NOT EDIT` / `SlipScheme` headers are auto-generated from JSON schemas by [SlipScheme](https://github.com/coryb/slipscheme). Do not edit these directly — modify the schema fetcher (`schemas/fetch-schemas.go`) or regenerate with `make generate`.

Key generated types: `Issue`, `SearchResults`, `IssueUpdate`, `Field`, `FieldMeta`, `Transition`, `Status`, `User`, `Comment`, `Component`, `Attachment`, `Version`, `Worklog`, `ErrorCollection`, `IssueType`, `Priority`, `Project`, `Resolution`, `StatusCategory`.

## Hand-written files (14 files)

These are safe to edit directly:

- `Activity.go` — Atom XML activity feed types (`Feed`, `Entry`)
- `providers.go` — Implements `Provide*` interfaces (`ProvideIssueUpdate`, `ProvideWorklog`, `ProvideLinkIssueRequest`, etc.)
- `SprintResults.go` — Sprint pagination response
- `CreateMetaFieldsPage.go` / `CreateMetaIssueTypesPage.go` — Paginated V2 createmeta with `PageValues()` abstraction handling both Cloud and Server/DC response formats
- `ErrorCollectionFuncs.go` — Error formatting helpers
- `TransitionsFuncs.go` — Transition lookup helpers
- `ListOfAttachmentFuncs.go` — Attachment list helpers
- `intOrString.go` — Handles fields that can be int or string
- `RankRequest.go` — Issue ranking request type
- `EpicIssues.go` — Epic issue association
- `ServerInfo.go` — Server info response
- Test files: `Attachment_test.go`, `IssueType_test.go`

## Dual format support

`CreateMetaFieldsPage` and `CreateMetaIssueTypesPage` use a `PageValues()` method that returns items from either `"fields"`/`"issueTypes"` (Cloud) or `"values"` (Server/DC) response keys, enabling the same code to work against both Jira Cloud and Server.
