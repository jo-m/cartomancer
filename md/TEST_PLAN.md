# Frontend Functional Test Plan

Manual/agent test plan covering the important functionality of every frontend page.
Instructions are intentionally brief; each item is a check to perform and the
expected result. Group "Setup" notes list the precondition (who must be logged in).

Roles: **Guest** (logged out), **User** (logged in), **Admin** (logged in admin).

---

## Global / Layout (every page)

- Top nav logo links to home (`/`); instance name renders under the logo.
- "Public Tracks" link works from nav and footer About/Help links work.
- Guest sees Login (and Register if registration enabled); no user menu.
- User sees "My Tracks" dropdown (Tracks, Groups, Upload) and user-avatar dropdown (Account, Logout, plus Admin if admin).
- Dropdowns open on hover/focus, close on Escape, navigable by arrow keys.
- Mobile (narrow viewport): hamburger toggles the same links.
- Logout returns to `/login` and clears the user menu.
- Page title (browser tab) updates per page, suffixed with `| Cartomancer`.

## Welcome (`/`)

- Guest: shows "Browse tracks", "Log in", and "Create account" (only if registration enabled).
- User: shows "Public tracks" and "My tracks" buttons.
- Three feature cards (Library, Oracle, Compass) render.

## Login (`/login`) — Guest only

- Empty submit shows "Required" errors; invalid email shows "Invalid email".
- Wrong credentials show an error toast; rate-limit (429) shows the throttle message.
- Valid login redirects to `/`.
- Demo-mode banner with demo email/password appears only when demo mode is on.
- "Create one" link present only when registration enabled.
- Logged-in user visiting `/login` is redirected away (guest route).

## Register (`/register`) — Guest only

- Validation: required fields, invalid email, mismatched passwords ("Passwords do not match").
- Successful submit shows "Check your email" screen.
- If registration disabled, redirects to `/login`.

## Confirm Email (`/confirm-email`)

- No `token` query param: shows "Missing confirmation token".
- With token, clicking "Confirm email" on success shows "Email confirmed" with a login link; on failure shows error toast.

## Account (`/account`) — User

- Avatar shows; "Rotate avatar" changes it.
- Profile: email is read-only; edit name; location search (type >=2 chars, pick a result, or Clear); Save shows "Profile updated".
- Change Email: requires new email + current password; success shows "Confirmation email sent".
- Change Password: old + new password; success shows "Password changed".
- Data Export: "Export my data" downloads a ZIP.
- Danger Zone: Delete account requires confirm step; confirming deletes, logs out, and redirects to `/login`.

## Tracks grid — Public (`/tracks`) and My Tracks (`/account/tracks`)

Shared TrackGrid behavior:
- List/Map view toggle works; selection resets to page 1.
- Search by name/location/filename filters results (debounced).
- Filters: distance & ascent dual sliders, track type (All/Recorded/Planned), sport + sub-sport chips, tags (with OR/AND when >1 tag), sort field + asc/desc.
- My Tracks only: Visibility filter (All/Public/Private) and "Starred" toggle.
- Pagination: Previous/Next disabled at ends; page size selector (12/24/48/96).
- "No tracks found" shown when empty.
- Star/unstar a card (logged-in only) toggles immediately.
- Track card links to detail; private cards show lock icon.
- My Tracks selection mode: click selects, shift-click range-selects, click outside clears, "Select all on page" works.
- Bulk toolbar (My Tracks, selection active): set public/private, set type, apply sport/sub-sport, apply tags, delete (with confirm); success toasts.

### Map view

- Tracks render as polylines; hover shows popover (name, user, distance, ascent, forecast); click navigates to track.
- "Filter by start location": pick mode crosshair, click sets center + radius circle; radius buttons change radius; Move re-picks; Reset clears; Esc cancels pick.
- Viewport (center/zoom) persists in URL (`m=`) across view switches.
- "Showing N of M" cap notice appears when results are capped.

## Upload (`/tracks/upload`) — User

- Drag-and-drop or click-to-select; only `.gpx`/`.fit` accepted.
- Active uploads list shows Pending/Uploading; failures list shows error + Dismiss / Dismiss all.
- Failed uploads persist across reload (session storage).
- "Pending review" lists uploaded tracks with preview, stats, visibility, sport, tags; Dismiss / Dismiss all.
- Bulk controls (when >1 pending): set all public/private, apply sport, apply tags; success toasts.
- Each pending track links to its detail page.

## Track detail (`/tracks/:uuid`)

- Loads track; invalid uuid shows "Track not found" / error.
- Header: author avatar+name, title, location (GeoNames link), description.
- Owner: click title to rename (Enter saves, Escape cancels); inline-edit sport, sub-sport, type, visibility, tags; Delete with confirm navigates home.
- Star/unstar (logged-in) toggles.
- Map renders track; fullscreen button opens dialog; hover overlay syncs with elevation/forecast charts. Falls back to preview SVG if no points.
- Road-closure warning shown when closures exist; "bring lights" warning when ride spans darkness.
- Forecast controls: "Start in" (+1/2/5/10/20h) and "Est. speed" (20/25/28/30) buttons re-fetch; shows est. duration + clock window; "no data"/"partial" warnings.
- Elevation profile and forecast chart render when data available.
- "Download original file" link works.
- Comments: visible if public or owner; list renders, deleted comments show placeholder; logged-in users can post (markdown), edit/delete own (per canEdit/canDelete) with confirm-less delete.

## Groups (`/tracks/groups`) — User

- Lists groups with sample name + member count; each links to group detail.
- "No groups found" when empty; loading/error states.

## Group detail (`/tracks/groups/:uuid`) — User

- "All groups" back link; heading shows track count; renders member track cards (no star/select).

## Segments (`/admin/segments`) — Admin

- Map renders segments (lines) and junctions (red dots) on Swisstopo tiles.
- Hover highlights a segment and shows distance/track-count overlay.
- Click a segment navigates to its detail page.
- Segment + junction counts shown; loading/error states.

## Segment detail (`/admin/segments/:uuid`) — Admin

- "All segments" back link; shows distance, tracks, H3 resolution, ascent.
- Segment map renders; start/end junction coords + H3 cells listed.
- Track UUID list links to each track detail.

## Admin: Users (`/admin/users`) — Admin

- Table (desktop) / cards (mobile) list users; search filters by name/email; URL `q=` synced.
- Create user: validation (email, name 3-32 alnum/-/_); success reveals one-time initial password banner.
- Edit user inline (name, email, admin); Save shows "User updated".
- Reset password (not self): confirm step, shows one-time password banner.
- Confirm email button shown only for users with pending verification.
- Delete user: confirm step.
- Copy ID cell copies uuid and toasts.

## Admin: Jobs (`/admin/jobs`) — Admin

- Worker status (none / 1 / split-brain), total rows, per-status counts.
- Auto-refresh every 5s; Pause/Resume and manual Refresh work.
- By-kind summary table; "Filter" sets kind filter.
- Filters: kind text, status select, "only with error", row limit (50/200/500/1000); all synced to URL.
- Jobs table/cards render; "No jobs found" when empty; truncation notice when capped.

## Admin: Forecasts (`/admin/forecasts`) — Admin

- Table/cards list forecasts; search filters by attribution/reference time; URL `q=` synced.
- Row click expands per-file detail (variable, valid time/until, size); attribution link opens external.
- Copy ID cells toast.

## Admin: Maps (`/admin/maps`) — Admin

- Table/cards list map builds (key, version, zoom, bbox, size, status, created).
- Status renders Ready / Not ready / Pending deletion.
- Delete: confirm step marks for deletion and toasts; marked rows dimmed and lose actions.
- Copy ID cells toast.

## About (`/about`)

- Loads version info (source/module links, Go version, version).
- Data Sources table and Dependencies table render with working external links.

## Help (`/help`)

- Renders help content from markdown.

## Leaving (`/leaving`)

- No/invalid `url` param: "Invalid link" with home link.
- Valid http(s) url: shows destination, "Go back" returns, "Continue to site" opens the external URL.

## Not Found (`*`)

- Unknown route shows 404 with "Back to home".

## Route guards

- Guest visiting protected routes (`/account`, `/account/tracks`, `/tracks/upload`, `/tracks/groups*`) is redirected to login.
- Non-admin visiting `/admin/*` is blocked.
- Logged-in user visiting `/login` or `/register` is redirected (guest route).
