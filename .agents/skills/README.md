# Repository Ponytail skills

The six Ponytail skills are vendored here for local and Codex Cloud work.
Codex discovers repository skills in `.agents/skills`; `AGENTS.md` requests
Ponytail `full` for coding tasks unless the user changes or disables the mode.

- Upstream: https://github.com/DietrichGebert/ponytail
- Version: `v5.1.0`, matching the installed desktop plugin
- Commit: `9cc65d03aa2da1db7121b912d03596409ee340b8`
- Files: unchanged upstream `skills/ponytail*/SKILL.md`
- License: MIT, retained in `PONYTAIL-LICENSE`

Start a new task after installing. In Codex, invoke `$ponytail`,
`$ponytail-review`, `$ponytail-audit`, `$ponytail-debt`, `$ponytail-gain`, or
`$ponytail-help`. Say `ponytail lite`, `ponytail full`, or `ponytail ultra`
to choose a level; say `stop ponytail` or `normal mode` to disable it.

This is an instruction-only installation. Plugin lifecycle hooks, automatic
updates, statusline, environment-variable mode selection, and persistent
mode flags are not installed. The plugin-specific instructions in the
upstream help skill apply to a full plugin installation, not this copy.

Cloud tasks must check out a pushed branch containing these files. After
merging the installation PR, new tasks based on `main` will include them.
No skill download or plugin installation is needed during cloud setup.

The gain skill cites upstream benchmark results, not TradeLens measurements:
https://github.com/DietrichGebert/ponytail/blob/9cc65d03aa2da1db7121b912d03596409ee340b8/benchmarks/results/2026-10-07-agentic.md

To update, use the skill-installer helper with a reviewed, pinned upstream
ref and `--dest <repo>/.agents/skills` (it refuses existing directories).
Replace only the six Ponytail directories, update this provenance and license,
and review the upstream instruction changes before committing.
