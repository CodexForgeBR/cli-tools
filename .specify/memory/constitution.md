<!--
Sync Impact Report - Version 1.0.0

Version Change: Initial constitution → 1.0.0
Modified Principles: N/A (initial creation)
Added Sections:
  - Core Principles (I-V)
  - Development Standards
  - Safety & Protection
  - Governance

Templates Status:
  ✅ plan-template.md - Reviewed, no changes needed (references constitution check)
  ✅ spec-template.md - Reviewed, no changes needed (user story structure compatible)
  ✅ tasks-template.md - Reviewed, no changes needed (task organization compatible)
  ⚠ Command templates - Not present in .specify/templates/commands/ (no action needed)

Follow-up TODOs: None
-->

# CodexForge CLI Tools Constitution

## Core Principles

### I. Single-Purpose Utilities

Every script MUST solve one specific problem clearly and completely.

**Rules:**
- Each script has exactly one well-defined purpose stated in its header comments
- No multi-function "Swiss Army knife" scripts
- Complex workflows compose simple scripts rather than creating monolithic solutions
- Script names clearly indicate their single purpose (e.g., `get-coderabbit-comments.sh`, not `github-tools.sh`)

**Rationale:** Single-purpose tools are easier to understand, test, maintain, and compose. They follow the Unix philosophy and enable better reusability across different contexts.

### II. Shell-First Implementation

Bash/shell scripting MUST be the default implementation choice unless clearly inappropriate.

**Rules:**
- Use POSIX-compatible shell (bash 3.2+) for maximum portability
- Prefer shell built-ins and standard Unix utilities over external dependencies
- Document any non-standard dependencies explicitly in script headers
- Python/other languages only when shell limitations are clear (e.g., complex data structures, API clients)
- All scripts MUST include proper error handling (`set -euo pipefail` or equivalent)

**Rationale:** Shell scripts are universally available, require no compilation or package management, and integrate seamlessly with Unix/Linux environments where these tools are used.

### III. Self-Documenting Code

Every script MUST be immediately understandable without external documentation.

**Rules:**
- Header comment block includes: Purpose, Usage, Examples, Exit codes
- Complex operations have inline comments explaining "why" not "what"
- Error messages are actionable and include next steps
- Help text (`--help` flag) is mandatory for all user-facing scripts
- Variable names are descriptive (`pr_number` not `pn`, `backup_dir` not `bd`)

**Rationale:** CLI tools are often used infrequently and in urgent situations. Self-documenting code reduces cognitive load and eliminates the need to reference external documentation.

### IV. Safety by Default

Scripts MUST fail safely and protect users from destructive operations.

**Rules:**
- Destructive operations require explicit confirmation or `--force` flag
- Wrapper scripts (rm, git) preserve data and prevent automated tools from bypassing safety mechanisms
- Exit codes follow Unix conventions (0 = success, 1 = general error, 2+ = specific errors)
- Always validate inputs before performing operations
- Backup data before transformations (where applicable)

**Rationale:** Automated CLI tools (Claude Code, AI agents) can execute commands without human oversight. Safety mechanisms prevent data loss and maintain user trust.

### V. Integration-First Design

Scripts MUST be designed for integration with AI agents and automation workflows.

**Rules:**
- Support both human-readable and machine-parseable output formats
- Exit codes are meaningful and documented
- Stdin/stdout/stderr protocol strictly followed
- Scripts are stateless where possible
- Environment variables for configuration, not hardcoded paths
- Compatible with Claude Code skills and agent workflows

**Rationale:** These tools primarily serve developers working with AI-assisted development. Integration-first design ensures they work seamlessly in automated contexts while remaining useful for manual execution.

## Development Standards

### Code Quality

- All scripts MUST pass ShellCheck static analysis (if shell scripts)
- POSIX compatibility preferred; bash-specific features allowed if justified
- Functions over repeated code (DRY principle)
- Exit early on errors; avoid nested error handling
- Use `local` for function-scoped variables

### Testing Requirements

- Manual verification checklist required for new scripts
- Integration tests for wrapper scripts (rm, git) that verify process detection
- Example usage in README MUST be tested before commit
- Breaking changes require migration guide

### Documentation

- README.md includes all scripts with purpose, usage, examples
- CLAUDE.md documents integration with Claude Code
- Script headers follow standard template
- Changelog for significant changes

## Safety & Protection

### Wrapper Script Standards

Wrapper scripts (rm, git, etc.) MUST:

- Detect automated CLI tool context via process tree analysis
- Log decisions to debug file (`~/.{script}-wrapper-debug.log`)
- Provide clear bypass instructions in error messages
- Maintain transparent passthrough for legitimate use cases
- Document whitelisted patterns explicitly

### Data Protection

- Backup files preserve full path structure with timestamps
- Backups stored in user home directory (`~/.{script}-backup/`)
- Cleanup scripts/instructions provided for backup directories
- Never silently modify user data without explicit consent

### AI Agent Safety

- Destructive git commands MUST fail fast (exit 1) when called from AI agents
- File deletion wrappers MUST backup files before removal
- No bypassing of pre-commit hooks allowed from automated tools
- All safety mechanisms logged for audit trail

## Governance

### Amendment Process

1. Propose changes via PR with justification in constitution amendment section
2. Document impact on existing scripts and workflows
3. Increment version according to semantic versioning
4. Update all dependent documentation in same PR
5. Migration guide required for breaking changes

### Versioning Policy

- **MAJOR**: Backward-incompatible changes to core principles or removal of safety mechanisms
- **MINOR**: New principles added or significant expansion of existing guidance
- **PATCH**: Clarifications, wording improvements, or non-semantic refinements

### Compliance Review

- All new scripts MUST comply with all five core principles before merge
- PRs include constitution compliance checklist
- Violations require explicit justification documented in PR
- Regular audits of existing scripts for alignment

### Development Guidance

- Constitution supersedes all other development practices
- When principles conflict with expediency, principles win
- Runtime development guidance maintained in `CLAUDE.md`
- Template updates synchronized when constitution changes

**Version**: 1.0.0 | **Ratified**: 2026-01-30 | **Last Amended**: 2026-01-30
