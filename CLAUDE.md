# CLAUDE.md

This is a web app to manage GPX and FIT tracks for cycling/running.
Frontend is React + TypeScript + Vite in `frontend/`.
Backend is a Golang REST API.
Most Go code is in `internal/`, some files (main.go etc) at top level.
CLAUDE.md for ALL go code is at `internal/CLAUDE.md`.

All build and dev tools are provided by the Nix flake (`flake.nix`).
`.envrc` triggers `use flake` so direnv loads the devshell automatically.
Outside of direnv, use `nix develop` to enter the shell.

```bash
# Run backend, implicitly configured by `.envrc`.
go run .

# Reproducible builds via Nix
nix build              # backend binary (result/bin/cartomancer)
nix build .#frontend   # frontend static assets
nix build .#dockerImage # minimal OCI image tar.gz
```

When adding a new Go dependency, the `vendorHash` in `flake.nix` must be
updated: set it to `pkgs.lib.fakeHash`, run `nix build`, copy the
expected hash from the error message. Same for `npmDepsHash` after
changes to `frontend/package-lock.json`.

# Universal Guidelines

- NEVER print or log URLs to console if they contain an API key
- NEVER hardcode sensitive configuration (keys/passwords) into the code
- MUST keep functions focused on a single responsibility
- MUST include docstrings for all public functions, classes, and methods
  - MUST document function parameters, return values, and exceptions raised
  - Keep comments up-to-date with code changes
- MUST use meaningful, descriptive variable and function names
- NEVER use emoji, or unicode that emulates emoji in the source code (e.g. ✓, ✗). The only exception is when writing tests and testing the impact of multibyte characters.
- MUST avoid including redundant comments which are tautological or self-demonstrating (e.g. cases where it is easily parsable what the code does at a glance so the comment does)
- MUST avoid including comments which leak what this file contains, or leak the original user prompt, ESPECIALLY if it's irrelevant to the output code.
