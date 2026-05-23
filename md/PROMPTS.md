# Rodney Setup
```bash
go install github.com/simonw/rodney@latest
google-chrome-stable --remote-debugging-port=9222 --user-data-dir="/tmp/chrome_debug"
rodney connect localhost:9222
```

Use the rodney tool (rodney --help) to connect to a live Chrome session to inspect the app.
Rodney was already connected to a Chrome instance and the dev server is running, you can start right with `rodney open http://localhost:5173/`.
Log in as admin@example.com password "admin123".

# Frontend Tests

1. (Opus)
Write a function check test plan for the frontend.
Go through all the frontend pages (in the source code) and take note of all important functionality for the plan.
Keep the test instructions brief and common, such that a human or agent can interpret easily without going into to many details.
Put it into TEST_PLAN.md.

2. (Sonnet)
Execute the frontend test plan in TEST_PLAN.md against the local dev server.
Use the rodney tool (rodney --help) to connect to a live Chrome session to inspect the app.
Rodney was already connected to a Chrome instance and the dev server is running,
you can start right with `rodney open http://localhost:5173/`.
Log in as admin@example.com password "admin123".
First, set the stage by creating users and uploading tracks which may then be used for the entire test plan.
Find some GPX tracks in data/testuploads-minimal.
Then, deploy subagents for the individual tests (but, in serial, parallel tests won't work on the single instance).
You may create or delete any data on this dev instance.
Write the results to FRONTEND_TEST_RESULTS.md.
