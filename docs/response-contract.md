# response contract

visor enforces a structured response contract before sending any platform reply.

## canonical fields

- `response_text` (derived from content before `---` separator)
- `send_voice` (bool)
- `code_changes` (bool)
- `conversation_finished` (bool)
- `commit_message` (string, required when `code_changes=true`)
- `git_push` (bool)
- `git_push_dir` (string)
- `memories_to_save` ([]string)

## invariants

- when `send_voice=false`, `response_text` must be non-empty
- when `send_voice=true`, empty `response_text` is allowed
- when `code_changes=true`, `commit_message` must be non-empty
- empty memory entries are removed by defaults fixer
- `conversation_finished=true` is only kept when goodbye intent appears in text

## runtime flow

1. parse raw agent output into contract response
2. apply deterministic defaults fixer
3. validate invariants
4. on failure: run one repair prompt once
5. if repair still invalid: fallback to safe `ok` text response

## schema export

the schema string is available via:

- `internal/agent/contract.JSONSchema()`

<!-- restart trigger touch: keep -->
