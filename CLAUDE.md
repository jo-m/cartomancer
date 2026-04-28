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
