# TODOs

**This file is ONLY FOR HUMANS. AI agents MUST IGNORE IT.**

## Short term

- [ ] Rename
- [x] Track/forecast view
  - [x] Hide data from the forecast charts if no forecast data available.
  - [x] Hovering the map view should show forecast values and distance and time in tooltip
  - [x] Maybe debounce the tooltip
  - [x] Remove the time and speed input boxes, leave only the buttons, and show forecast immediately when clicking one of them
  - [x] Make time and speed inputs global for the track view, not only for forecast. In all the x axes and tooltips show wall clock time and elapsed time.
  - [x] Simply always show the forecast if available.
  - [x] Allow to make track map full screen.
  - [x] Make forecasts access public
  - [x] Frontend meteo data attribution
  - [x] Attribution for map (Swisstopo)
  - [x] Window averaging and subsampling of points on plots, if there are too many
- [x] Store vertical+horizontal grid data for forecasts
- [x] Show track owner/user in frontend
- [x] Aggressive caching with etag headers for all expensive endpoints
- [x] Forecast also wind, incl. direction
- [x] Analyze wind direction relative to travel direction (head/tailwind)
- [x] Track names in DB: Strip whitespace before saving. Strip any leading dots. Do not allow empty. On upload, assign some name if empty.
- [x] Run periodic jobs immediately the first time
- [x] XSRF protection
- [ ] ~~Update go tool air config~~
- [x] Disallow reset admin password functionality for admin's own accounts (too dangerous)
- [x] Only show email confirmation button in the frontend if there actually is a pending one
- [x] At / serve a robots.txt which disallows ANY robot on ANY page, except the front page. Make the front page have no dynamic content when not logged in.
- [x] On the tracks page allow sorting by distance, ascent, created_at ("uploaded at"), original_created_at ("file creation date")
- [x] Tracks filter view is currently inconsistent
- [x] Deduplicate track blobs between users
- [x] Ensure long running periodic jobs cannot pile up
- [x] Debouncing for job submitting
- [x] Show overview of forecasts in db for admin, forecasts, vars, bbox, time window, step
- [x] Compute/show wind speed: https://github.com/MeteoSwiss/meteodata-lab/blob/main/src/meteodatalab/operators/wind.py
- [x] Make logg pkg capable of using t.Log() if it is inside a test
- [x] Track on map - add white line/background along track
- [x] Tracks grouping
  - [x] Job
  - [x] REST API
  - [x] Show in frontend
- [x] Remove anonymous sessions
- [x] Geonames
  - [x] Attribution
  - [x] Offline test with a subsample of downloaded data
  - [x] Filter out any undersea U features.
  - [x] Also load the featureCodes_en.txt file in an online test and generate the codes as Go code.
  - [x] Also load the admin1CodesASCII.txt and admin2Codes.txt files and load them into tables
  - [x] Drop the following cols: modification date, timezone, dem, elevation, population
  - [x] Maybe do import in a temp table which is then renamed
- [ ] Frontend polishing
  - [ ] Unified display of dates/times in frontend (iso and 24h format, "x ago")
  - [ ] Shared components
  - [ ] Theming
  - [ ] Branding
  - [ ] Favicon, Logo
  - [ ] Move at least some page state to URL in frontend
  - [ ] Ensure correct cursors used everywhere (buttons etc), why is this not standard?
  - [ ] Mobile friendly and responsive
  - [ ] Use full width on desktop
  - [ ] Use https://github.com/simonw/rodney to let Claude inspect the frontend and CSS, or https://github.com/ChromeDevTools/chrome-devtools-mcp/tree/main
  - [ ] Nicer UI for track editing
  - [ ] In `App.tsx` let `QueryClient` use `staleTime`.
  - [ ] Add `<ErrorBoundary>` somewhere, especially considering `client.ts` will throw

## Test (manually)

- [ ] Forecasts - incomplete and missing data
- [ ] Forecasts - reasonable data for precip

## Before initial push/deploy

- [ ] Periodic db `VACUUM` and `.backup` (atomic via mv)
- [x] Allow to create an initial admin account (allow setting password only in dev mode)
- [ ] Demo mode, locks user table via trigger, insert some initial data, delete data periodically
- [ ] Disable public signup by default
- [ ] Uncomment all the checks/linters in make check
- [ ] Log message cleanup and unification (case, punctuation)
- [ ] Add CI setup/Docker build
  - [ ] In CI also run the online tests, but allow them to fail
- [x] App config
  - [x] Split up/move around config to relevant config/module structs (e.g. separate struct for users, registrations)
  - [x] Sensible defaults for everything
  - [x] Log warning for the settings which MUST be set for prod (e.g. JWT secrets)
  - [x] Consistent env var prefixes
  - [x] Validate all config options on load/startup
  - [x] Document which config options MUST be set for a prod deployment
- [x] Improve data sources attribution CC-BY 4.0 for meteo, geonames, map (https://wiki.creativecommons.org/wiki/Recommended_practices_for_attribution#Attributing_materials_from_multiple_sources).
  - [x] Systematically add online unit tests which ensure the license has not changed.
- [ ] Update README.md
- [ ] Fixup/autosquash
- [ ] Grep for TODO in code
- [ ] SQLite without rowid? https://sqlite.org/withoutrowid.html
- [ ] Go through the periodic TODOs

## Before enabling public signup

- [ ] Add privacy policy, imprint, admin contact
- [ ] Periodically delete user accounts which have never had their email confirmed
- [ ] Improve email messages sent to users, include instance name and base URL
- [ ] Self serve password reset flow for users
- [ ] Rate limiting for sensitive endpoints (login etc), or hint for deployment
- [ ] Self-serve full data export for users
- [ ] TOTP login for users
- [ ] Decide on what can be seen by anon/non logged in users.
- [ ] Hints in the frontend:
  - [ ] Email will always stay private, name is shown
  - [ ] Advanced email normalization (e.g. Gmail dots, +tags)
  - [ ] Warning that for public tracks original file will be public

## Later (maybe)

- [ ] Check in the generated files
- [ ] Update to Vite 8
- [ ] https://brandur.org/two-phase-render
- [ ] Skills/subagents
- [ ] Load/show GPX wpts, see `internal/pkg/load/testdata/COURSE_436298480.gpx`
- [x] Reverse geocoding via https://download.geonames.org/export/dump/
- [ ] Calculate/fit model of bike ride speed dep. on terrain, from personal data
- [ ] How could architecture of the db package be improved. Maybe split up "system" and "app" tables.
- [ ] Improve error JSON struct, e.g. https://platform.claude.com/docs/en/api/errors#error-shapes, at least include request id
- [ ] Allow to show meteo forecast in map overlay
- [ ] Improve the tracks geoname labelling algorithm
- [ ] Extract all (some) of the hardcoded consts/limits into app settings
- [ ] Explore by tags page
- [ ] Periodically compute wind rose and average temp for all tracks
- [ ] Let users configure their location and then show their loc on maps, and allow to filter tracks by relative location
- [ ] On long tracks track on map is now imprecise when zooming in
- [ ] Track sharing link with limited time (JWT)

## To enable additional weather providers

- [ ] Add provider name to database
- [ ] Make data attribution go through database

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
- [ ] `go fix ./...`
- [ ] Bump dependencies
  - [ ] Backend
  - [ ] Frontend
  - [ ] CI
