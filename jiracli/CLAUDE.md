# jiracli — CLI infrastructure

Shared CLI framework used by all commands. This package owns:

## Core framework (`cli.go`)
- `GlobalOptions`: endpoint, auth-method, login, user, password-source, rate-limit, burst, validate-token, download flag, User-Agent fields (AppName, Contact, Runtime, Hostname, AwsAccountID, AwsRegion, Environment)
- `CommonOptions`: browse, editor, file, gjq, skip-editing, template
- `CommandRegistryEntry` / `CommandRegistry`: command registration system
- `register()`: sets up global flags and oreo pre/post callbacks for auth, rate limiting, retry
- `EditLoop()`: interactive edit-submit-retry cycle for create/edit commands

## Template system (`templates.go`)
- `TemplateProcessor()` returns a `text/template` with custom functions: `jira`, `env`, `fit`, `shellquote`, `toJson`, `toMinJson`, `termWidth`, `indent`, `comment`, `color`, `colorDiff`, `regReplace`, `split`, `join`, `mapField`, `abbrev`, `age`, `dateFormat`, `wrap`, `sprint`, `fieldLabel`, `fieldKey`, `fieldID`, `optionLabel`, `optionKey`, `headers`, `row`, `cell`, `defaultColWidth`, plus all Sprig functions
- `AllTemplates`: map of built-in default templates

## Field mapping (`fieldmap.go`)
- Bidirectional mapping between human-readable field names and Jira custom field IDs
- `FieldMap` loaded from `.jira.d/fields.json`
- Key functions: `FieldIDForKey()`, `FieldLabelForID()`, `FieldKeyForID()`, `OptionIDForKey()`, `TransformFieldKeys()`

## Auth and credentials (`password.go`, `keyring.go`, `validate.go`)
- `GetPass()`: retrieves from keyring/pass/gopass/stdin/env/prompt
- `SetPass()` / `ClearCachedPass()`: manages credential lifecycle
- `ValidateToken()`: probes `/rest/api/2/myself`, clears cache on 401

## Resilience (`ratelimit.go`, `retry.go`)
- Token-bucket rate limiter with configurable rate/burst
- Exponential backoff with jitter, Retry-After header support on 429
- Retryable status codes: 408, 429, 500, 502, 503, 504

## Security (`redact.go`)
- `RedactString()`, `RedactBytes()`, `RedactArgs()`: regex-based scrubbing of Authorization headers, session cookies, credential query parameters from log output

## Disney-specific (`useragent.go`)
- `BuildUserAgent()`: AES REST client User-Agent format with auto-detected runtime (Lambda/ECS/K8s/Jenkins/CI/local-dev)

## Tests
Unit tests exist for: `fieldmap`, `ratelimit`, `redact`, `retry`, `useragent`, `sprints_template`. Run with `go test ./jiracli/`.
