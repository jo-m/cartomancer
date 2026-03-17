// Corstest is a minimal "third-party" server that serves an HTML page
// attempting cross-origin requests against the main Detour API. It is used
// to manually verify that CSRF protection works correctly.
//
// Usage:
//
//	go run ./internal/cmd/corstest --target http://127.0.0.1:8080
//
// Then open http://127.0.0.1:8888 in a browser and click the buttons.
package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
)

var (
	listen = flag.String("listen", "127.0.0.1:8888", "address to listen on")
	target = flag.String("target", "http://127.0.0.1:8080", "base URL of the Detour server")
)

var page = template.Must(template.New("page").Parse(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>CSRF Test Page (third-party origin)</title>
<style>
  body { font-family: monospace; max-width: 800px; margin: 2rem auto; }
  button { margin: 0.25rem 0; padding: 0.5rem 1rem; cursor: pointer; }
  .result { white-space: pre-wrap; background: #f0f0f0; padding: 0.5rem; margin: 0.25rem 0 1rem 0; font-size: 0.85rem; min-height: 1.5em; }
  h3 { margin: 1.5rem 0 0.5rem 0; }
  .pass { color: green; }
  .fail { color: red; }
</style>
</head>
<body>
<h2>CSRF / Cross-Origin Test Page</h2>
<p>This page is served from <b>{{.Listen}}</b> and makes requests to <b>{{.Target}}</b>.</p>
<p>Because the origins differ, the browser should block cross-origin requests that carry custom headers (CORS preflight will fail), and the server should reject requests without the CSRF header.</p>

<h3>1. GET /api/app_config (should succeed - safe method, no header needed)</h3>
<button onclick="doTest('test1', 'GET', '/api/app_config', false)">Run</button>
<div id="test1" class="result">-</div>

<h3>2. POST /api/sessions/login without CSRF header (should fail - 403 or CORS block)</h3>
<button onclick="doTest('test2', 'POST', '/api/sessions/login', false)">Run</button>
<div id="test2" class="result">-</div>

<h3>3. POST /api/sessions/login with CSRF header (should fail - CORS preflight rejects custom header)</h3>
<button onclick="doTest('test3', 'POST', '/api/sessions/login', true)">Run</button>
<div id="test3" class="result">-</div>

<h3>4. DELETE /api/tracks/fake-uuid without CSRF header (should fail)</h3>
<button onclick="doTest('test4', 'DELETE', '/api/tracks/fake-uuid', false)">Run</button>
<div id="test4" class="result">-</div>

<h3>5. Form POST to /api/sessions/login (classic CSRF via form submission)</h3>
<p>This opens in a new tab. The server should reject it (no CSRF header).</p>
<form method="POST" action="{{.Target}}/api/sessions/login" target="_blank">
  <input type="hidden" name="email" value="test@example.org">
  <input type="hidden" name="password" value="asdf">
  <button type="submit">Submit Form</button>
</form>

<h3>6. Run all fetch tests</h3>
<button onclick="runAll()">Run All</button>

<script>
const BASE = {{.Target}};

async function doTest(id, method, path, withCsrfHeader) {
  const el = document.getElementById(id);
  el.textContent = "Running...";
  el.className = "result";
  try {
    const headers = { "Content-Type": "application/json" };
    if (withCsrfHeader) {
      headers["X-Requested-With"] = "detour";
    }
    const resp = await fetch(BASE + path, {
      method: method,
      headers: headers,
      credentials: "include",
      body: method === "GET" ? undefined : JSON.stringify({}),
    });
    const text = await resp.text();
    const ok = method === "GET" ? resp.ok : !resp.ok;
    el.className = "result " + (ok ? "pass" : "fail");
    el.textContent = resp.status + " " + resp.statusText + "\n" + text;
  } catch (err) {
    // Network error / CORS block shows up here.
    const isPost = method !== "GET";
    el.className = "result " + (isPost ? "pass" : "fail");
    el.textContent = "Blocked by browser (CORS/network error): " + err.message;
  }
}

async function runAll() {
  await doTest("test1", "GET", "/api/app_config", false);
  await doTest("test2", "POST", "/api/sessions/login", false);
  await doTest("test3", "POST", "/api/sessions/login", true);
  await doTest("test4", "DELETE", "/api/tracks/fake-uuid", false);
}
</script>
</body>
</html>`))

func main() {
	flag.Parse()

	data := struct {
		Listen string
		Target string
	}{*listen, *target}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := page.Execute(w, data); err != nil {
			log.Printf("template error: %v", err)
		}
	})

	fmt.Printf("CSRF test page at http://%s (targeting %s)\n", *listen, *target)
	log.Fatal(http.ListenAndServe(*listen, nil))
}
