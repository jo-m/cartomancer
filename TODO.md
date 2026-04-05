# TODOs

**This file is ONLY FOR HUMANS. AI agents MUST IGNORE IT.**

copilot --allow-tool='write' --allow-tool='shell(go:*)' --allow-tool='shell(npm:*)' --allow-tool='shell(make:*)'

## Short term

- [x] Rename
- [x] Track/forecast view
  - [x] Hide data from the forecast charts if no forecast data available.
  - [x] Maybe debounce the tooltip
  - [x] Make time and speed inputs global for the track view, not only for forecast.
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
- [x] Track names in DB: Strip whitespace and normalize before saving.
- [x] Run periodic jobs immediately the first time
- [x] XSRF protection
- [x] Disallow reset admin password functionality for admin's own accounts (too dangerous)
- [x] Only show email confirmation button in the frontend if there actually is a pending one
- [x] At / serve a robots.txt which disallows ANY robot on ANY page, except the front page.
- [x] Make the front page have no dynamic content when not logged in.
- [x] On the tracks page allow sorting by distance, ascent, created_at
- [x] Tracks filter view is currently inconsistent
- [x] Deduplicate track blobs between users
- [x] Try to be smarter about imported tracks - recorded vs. planned. For Garmin also file name - COURSE vs. ACTIVITY.
- [x] Ensure long running periodic jobs cannot pile up
- [x] Debouncing for job submitting
- [x] Show overview of forecasts in db for admin, forecasts, vars, bbox, time window, step
- [x] Compute/show wind speed
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
  - [x] `geocode` -> `geonames` in api
  - [ ] Track filtering by location
  - [ ] Maybe use simple index for search (faster, but can do only prefix)
- [ ] Extract segments to allow for "remixing" of tracks
  - [ ] Ensure deleting tracks also correctly deletes segments
  - [ ] Ensure the segments job also cleans the database before inserting
  - [ ] Make segments available to non admins
  - [ ] Increase the segments job debounce time
  - [ ] Ensure the segmenting job gets triggered reliably in all scenarios
  - [ ] Re-enable the segmenting job
  - [ ] New approach for finding junctions: Iterate through all tracks, with cells, and for each cell keep the (interpolated) GPX track points. Then do negative filtering based on that.
- [ ] Track grouping: treat recorded vs. planned tracks differently
- [ ] Frontend polishing
  - [ ] Unified display of dates/times in frontend (iso and 24h format, "x ago")
  - [x] Shared components
  - [ ] New tagline (also update repo desc)
  - [x] Theming
  - [x] Branding
  - [x] Favicon, Logo
  - [x] Move at least some page state to URL in frontend
  - [x] Ensure correct cursors used everywhere (buttons etc), why is this not standard?
  - [x] Mobile friendly and responsive
  - [x] Use full width on desktop
  - [x] Use https://github.com/simonw/rodney to let Claude inspect the frontend and CSS, or https://github.com/ChromeDevTools/chrome-devtools-mcp/tree/main
  - [ ] Nicer UI for track editing
  - [ ] In `App.tsx` let `QueryClient` use `staleTime`.
  - [ ] Add `<ErrorBoundary>` somewhere, especially considering `client.ts` will throw
  - [x] Frontend aria/accessibility
  - [x] Avoid any layout shifts with interactive elements
  - [x] Bulk editing and deletion of tracks
  - [x] Automatic dark mode

## Before initial push/deploy

- [x] Periodic db `.backup` (atomic via mv)
- [x] Allow to create an initial admin account (allow setting password only in dev mode)
- [x] Demo mode, locks user table via trigger, insert some initial data, delete data periodically
- [x] Disable public signup by default
- [x] 404 page in frontend
- [ ] Setup and installation instructions
- [ ] Uncomment all the checks/linters in make check
- [x] Log message cleanup and unification (case, punctuation)
- [ ] Add CI setup/Docker build
  - [ ] In CI also run the online tests, but allow them to fail
- [x] App config
  - [x] Split up/move around config to relevant config/module structs (e.g. separate struct for users, registrations)
  - [x] Sensible defaults for everything
  - [x] Log warning for the settings which MUST be set for prod (e.g. JWT secrets)
  - [x] Consistent env var prefixes
  - [x] Validate all config options on load/startup
  - [x] Document which config options MUST be set for a prod deployment
- [x] Improve data sources attribution CC-BY 4.0 for meteo, geonames, map
  - [x] Systematically add online unit tests which ensure the license has not changed.
- [ ] Update README.md
- [ ] Fixup/autosquash
- [x] Grep for TODO in code
- [x] SQLite without rowid
- [ ] Go through the periodic TODOs

## Before enabling public signup

- [ ] Find a solution for segments extraction privacy
- [ ] Add privacy policy, imprint, admin contact (https://notermsnoconditions.com/)
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
  - [ ] Warning that for public tracks original file will be public, incl. start points and potential other data in it

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
- [x] Improve the tracks geoname labelling algorithm
- [ ] Extract all (some) of the hardcoded consts/limits into app settings
- [ ] Explore by tags page
- [ ] Make tags on track view page clickable and go to list view, with filter applied
- [ ] Correct elevation on upload
- [x] Periodically compute wind rose and average temp for all tracks
- [ ] On long tracks, the track on the map is now imprecise when zooming in due to subsampling
- [ ] Share link to private track with signed link (limited time)
- [ ] Filter tracks by start/end point
- [ ] Let users configure their location and then show their loc on maps, and allow to filter tracks by relative location
- [x] Show road closures, construction (entered manually, or sourced somewhere)
- [ ] Download road closures from more sources (e.g. https://www.geocat.ch/geonetwork/srv/ger/catalog.search#/search?any=Baustellen)
- [ ] Download country shapes, and issue a warning if a track includes a border crossing
- [ ] Map view, showing all tracks and their starting points
- [ ] Allow to attach images to tracks

## Periodic

- [ ] Review
  - [ ] Docstrings, go doc links `[]` https://tip.golang.org/doc/comment, `go run golang.org/x/pkgsite/cmd/pkgsite@latest -open`
    - [ ] Add examples/usage to docstrings of packages.
  - [ ] OWASP Top 10
  - [ ] Authn/authz, Login and sessions handling, API endpoints permissions handling
  - [ ] Generic Golang style/architecture issues
  - [ ] State handling in frontend
  - [ ] Update CLAUDE.md and README.md
  - [ ] Review all the panic()s (main.go, tests, and some special cases are ok)
  - [ ] Review any SQL queries which are not in sqlc
  - [ ] Remove unused code
- [ ] `go fix ./...`
- [ ] Bump dependencies
  - [ ] Backend
  - [ ] Frontend
  - [ ] CI
