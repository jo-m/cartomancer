# CLAUDE.md

Web app to manage GPX/FIT tracks for cycling/running.
- Frontend: React + TypeScript + Vite in `frontend/` (see `frontend/CLAUDE.md`).
- Backend: Go REST API. Code in `internal/` plus a few top-level files (`main.go` etc). See `internal/CLAUDE.md` for ALL Go code.

Build/dev tools come from the Nix flake (`flake.nix`). `.envrc` runs `use flake` so direnv loads the devshell automatically; otherwise use `nix develop`.

```bash
go run .                # run backend (configured via .envrc)
nix build               # backend binary -> result/bin/cartomancer
nix build .#frontend    # frontend static assets
nix build .#dockerImage # minimal OCI image tar.gz
```

When adding a Go dep, update `vendorHash` in `flake.nix`: set to `pkgs.lib.fakeHash`, run `nix build`, copy the expected hash from the error. Same for `npmDepsHash` after `frontend/package-lock.json` changes.
