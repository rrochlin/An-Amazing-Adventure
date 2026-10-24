# End-to-end play

Play the real client in a browser against the real ws-chat and ws-game-action handlers on this
machine, with real Bedrock calls. Use it to play through a campaign and look for bugs. Each turn costs a
model call, as in production.

```bash
# 1. The backend: real handlers, sessions in memory (kept in the state file across restarts)
cd server && AWS_PROFILE=aa-debug AWS_REGION=us-west-2 \
  go run ./tools/localserver -addr :8787 -traces /tmp/traces -state /tmp/sessions.json

# 2. The client, pointed at it
cd client && VITE_APP_URI=http://localhost:8787 VITE_WS_ENDPOINT=ws://localhost:8787/ws \
  pnpm exec vite --port 5174

# 3. A browser to drive (Chromium with software WebGL), with HTTP commands on :9333
cd e2e && npm install && node play-driver.mjs /tmp/shots
```

The driver puts a stand-in token in localStorage (the local server reads the user from its `sub`, as
ws-connect does) and logs console errors, page errors and every WebSocket frame with a timestamp.

| Command | Does |
|---|---|
| `/goto?url=` | open a page |
| `/state` | what's on screen: the sign, the line and speaker, choices, the text box, the scene (player tile, room, who's there, locked or not) |
| `/click?sel=` or `?x=&y=` · `/button?name=` · `/text?t=` | click |
| `/key?k=&ms=` | press, or hold for `ms` |
| `/type?text=` | type into the text box and send |
| `/tile?x=&y=` | click a map tile (walk there) · `/talk?id=` click a character |
| `/until?ms=` | wait for the story to settle |
| `/shot?name=` | screenshot · `/perf` frame rate, long tasks, heap · `/log` events |

Software WebGL caps the frame rate at about 22 fps, so measure performance on real hardware.
