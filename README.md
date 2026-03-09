# ccstatusline

A custom status line for [Claude Code](https://docs.anthropic.com/en/docs/claude-code).

Shows current directory, git branch with status indicators, model name, vim mode, and context window usage.

## Install

```
go install github.com/bguisard/ccstatusline@latest
```

## Configure

In your Claude Code settings (`~/.claude/settings.json`):

```json
{
  "statusLine": {
    "type": "command",
    "command": "ccstatusline"
  }
}
```

## Example

```
~/my-project main!? ❯ Opus 4.6 · ctx 80%
```

## Git status indicators

| Symbol | Meaning |
|--------|---------|
| `+` | Staged changes |
| `!` | Unstaged changes |
| `?` | Untracked files |
| `~` | Status unknown (timeout) |
