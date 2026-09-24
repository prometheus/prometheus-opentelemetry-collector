# Changelog Entries

This repository uses `go.opentelemetry.io/build-tools/chloggen` to collect
release notes before cutting receiver module tags.

Create an entry for user-visible changes:

```bash
make chlog-new NAME=<short-entry-name>
```

Then edit the generated `.chloggen/<short-entry-name>.yaml` file.

Use one of these change types:

- `breaking`
- `deprecation`
- `new_component`
- `enhancement`
- `bug_fix`

Run validation before opening or merging a pull request:

```bash
make chlog-validate
```

Preview the generated changelog without modifying files:

```bash
make chlog-preview
```

Pull request CI requires at least one new `.chloggen/*.yaml` entry and rejects
direct edits to `CHANGELOG.md`. It also validates the entries and checks links
in the rendered preview.

Small internal-only maintenance changes may omit a changelog entry. To skip
the CI requirement, start the pull request title with `[chore]` or ask a
maintainer to add the `Skip Changelog` label. Dependency update pull requests
and pull requests opened by Dependabot or Renovate are also exempt.
