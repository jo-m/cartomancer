# TODOs

**This file is ONLY FOR HUMANS. AI agents MUST IGNORE IT.**

## Short term

- [ ] Rename, icon, favicon, branding
- [ ] Track/forecast view
  - [ ] Hide the forecast charts again if no forecast data available
  - [ ] Hovering the map view should show forecast values and distance and time in tooltip
  - [ ] Maybe debounce the tooltip
  - [ ] Remove the time and speed input boxes, leave only the buttons, and show forecast immediately when clicking one of them
  - [ ] Make time and speed inputs global for the track view, not only for forecast. In all the x axes and tooltips show wall clock time and elapsed time.
  - [ ] Simply always show the forecast if available.
  - [ ] Allow to make track map full screen.
  - [ ] Make forecasts access public
- [ ] Store vertical+horizontal grid data for forecasts
- [ ] Show track owner/user in frontend
- [x] Aggressive caching with etag headers for all expensive endpoints
- [ ] Frontend meteo data attribution
- [ ] Forecast also wind, incl. direction, and sunshine duration
- [ ] Track names in DB: Strip whitespace before saving. Strip any leading dots. Do not allow empty. On upload, assign some name if empty.

## Before initial push/deploy

- [ ] Update README.md
- [ ] Fixup/autosquash
- [ ] Check in the generated files
- [ ] In `App.tsx` let `QueryClient` use `staleTime`.
- [ ] Add `<ErrorBoundary>` somewhere, especially considering `client.ts` will throw
- [ ] Grep for TODO in code
- [ ] XSRF protection
- [ ] Update go tool air config
- [ ] At / serve a robots.txt which disallows ANY robot on ANY page, except the front page. Make the front page have no dynamic content when not logged in.
- [ ] How could architecture of the db package be improved. Maybe split up "system" and "app" tables.
- [ ] Disable public signup by default
- [ ] On the tracks page allow sorting by distance, ascent, created_at ("uploaded at"), original_created_at ("file creation date")
- [ ] Tracks filter view is currently inconsistent
- [ ] SQLite without rowid? https://sqlite.org/withoutrowid.html
- [ ] Uncomment all the checks/linters in make check
- [ ] Add https://go.dev/blog/gofix, https://go.dev/blog/inliner
- [ ] Add CI setup/Docker build
- [ ] In CI also run the online tests, but allow them to fail
- [ ] Deduplicate track blobs between users
- [ ] Explore by tags page
- [ ] Show overview of forecasts in db for admin, forecasts, vars, bbox, time window, step
- [ ] Compute/show wind speed: https://github.com/MeteoSwiss/meteodata-lab/blob/main/src/meteodatalab/operators/wind.py
- [ ] Add admin contact email
- [ ] Add privacy policy, impressum
- [ ] App config
  - [ ] Split up/move around config to relevant config/module structs (e.g. separate struct for users, registrations)
  - [ ] Sensible defaults for everything
  - [ ] Log warning for the settings which MUST be set for prod (e.g. JWT secrets)
  - [ ] Consistent env var prefixes
  - [ ] Make external base url actually work
  - [ ] Validate all config options on load/startup
  - [ ] Document which config options MUST be set for a prod deployment

## Before enabling public signup

- [ ] Improve email messages, include instance name and base URL
- [ ] Rate limiting for sensitive endpoints (login etc), or hint for deployment
- [ ] Self-serve full data export for users
- [ ] TOTP login for users
- [ ] Decide on what can be seen by anon/non logged in users.
- [ ] Hints in the frontend:
  - [ ] Email will always stay private, name is shown
  - [ ] Advanced email normalization (e.g. Gmail dots, +tags)
  - [ ] Warning that for public tracks original file will be public

## To enable additional weather providers

- [ ] Add provider name to database
- [ ] Make data attribution go through database

## Later

- [ ] Periodic db `VACUUM` and `.backup`
- [ ] https://brandur.org/two-phase-render
- [ ] Skills/subagents
- [ ] Allow to show meteo forecast in map overlay
- [ ] Load/show GPX wpts, see `internal/pkg/load/testdata/COURSE_436298480.gpx`
- [ ] Geocoding and maybe reverse geocoding
- [ ] Make logg pkg capable of using t.Log() if it is inside a test
- [ ] Calculate/fit model of bike ride speed dep. on terrain, from personal data
- [ ] Frontend polishing
  - [ ] Ensure correct cursors used everywhere (buttons etc), why is this not standard?
  - [ ] Mobile friendly and responsive
  - [ ] Use full width on desktop
  - [ ] Use https://github.com/simonw/rodney to let Claude inspect the frontend and CSS
  - [ ] Nicer UI for track UI editing

## Periodic

- [ ] Review
  - [ ] Docstrings, go doc links `[]` https://tip.golang.org/doc/comment, `go run golang.org/x/pkgsite/cmd/pkgsite@latest -open`
  - [ ] OWASP Top 10
  - [ ] Authn/authz
  - [ ] Login and sessions handling
  - [ ] API endpoints permissions handling
  - [ ] Generic Golang style/architecture issues
  - [ ] State handling in frontend
  - [ ] Frontend aria/accessibility
  - [ ] Update CLAUDE.md and README.md
  - [ ] Review all the panic()s (main.go, tests, and some special cases are ok)
  - [ ] Review any SQL queries which are not in sqlc
- [ ] Bump dependencies
  - [ ] Backend
  - [ ] Frontend
  - [ ] CI
