// This go.mod prevents Go tools (staticcheck, etc.) from scanning
// frontend/node_modules, which may contain .go files that fail checks.
module frontend

go 1.20
