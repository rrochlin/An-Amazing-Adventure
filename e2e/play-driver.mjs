// A long-running browser for playing the game by hand: HTTP commands on :9333.
import { chromium } from 'playwright';
import http from 'node:http';
import fs from 'node:fs';
const OUT = process.argv[2];
const b64 = (o) => Buffer.from(JSON.stringify(o)).toString('base64url');
const jwt = (claims) => `${b64({ alg: 'none', typ: 'JWT' })}.${b64(claims)}.sig`;
const sub = 'rob-e2e';
const tokens = { accessToken: jwt({ sub }), idToken: jwt({ sub, email: 'e2e@local', 'cognito:groups': [] }), refreshToken: 'r', expiresAt: Date.now() + 864e5, userSub: sub };

const b = await chromium.launch({ args: ['--use-gl=angle', '--use-angle=swiftshader', '--enable-unsafe-swiftshader'] });
const ctx = await b.newContext({ viewport: { width: 1440, height: 900 }, deviceScaleFactor: 1 });
await ctx.addInitScript((t) => localStorage.setItem('AAA_TOKENS', JSON.stringify(t)), tokens);
await ctx.addInitScript(() => {
  // Frame-rate and long-task probes.
  window.__perf = { frames: [], long: [] };
  const tick = (t) => { window.__perf.frames.push(t); if (window.__perf.frames.length > 600) window.__perf.frames.shift(); requestAnimationFrame(tick); };
  requestAnimationFrame(tick);
  try { new PerformanceObserver((l) => l.getEntries().forEach((e) => window.__perf.long.push({ at: Math.round(e.startTime), ms: Math.round(e.duration) }))).observe({ type: 'longtask', buffered: true }); } catch {}
});
const page = await ctx.newPage();
let log = [];
const note = (kind, data) => log.push({ t: Date.now(), kind, ...data });
page.on('console', (m) => { if (['error', 'warning'].includes(m.type())) note('console', { level: m.type(), text: m.text().slice(0, 400) }); });
page.on('pageerror', (e) => note('pageerror', { text: String(e).slice(0, 400) }));
page.on('websocket', (ws) => {
  note('ws', { open: ws.url().split('?')[0] });
  ws.on('framesent', (f) => note('sent', { data: String(f.payload).slice(0, 300) }));
  ws.on('framereceived', (f) => { let type = '?'; try { type = JSON.parse(f.payload).type; } catch {} note('recv', { type, size: String(f.payload).length, data: type === 'error' || type === 'narrative_chunk' ? String(f.payload).slice(0, 200) : undefined }); });
  ws.on('close', () => note('ws', { closed: true }));
});

const state = async () => page.evaluate(() => {
  const q = (s) => document.querySelector(s);
  const s = window.__stage;
  return {
    url: location.pathname,
    chip: q('[data-testid=scene-chip]')?.innerText?.replace(/\n/g, ' | '),
    text: q('[data-testid=stage-text]')?.innerText,
    speaker: q('[data-testid=stage-text]')?.previousElementSibling?.innerText,
    playerLine: q('[data-testid=stage-player-line]')?.innerText?.replace(/\n/g, ' '),
    continue: !!q('[aria-label=continue]'),
    choices: [...document.querySelectorAll('[data-testid=stage-panel] button')].map((b) => b.innerText).filter((t) => t && !['Act', 'Say', 'Leave'].includes(t)),
    input: q('[data-testid=stage-panel] input')?.placeholder,
    buttons: [...document.querySelectorAll('[data-testid=stage-panel] button')].map((b) => b.innerText).filter(Boolean),
    alerts: [...document.querySelectorAll('[role=alert]')].map((a) => a.innerText),
    stage: s ? { ...s.probe(), locked: !!s.view?.controlsLocked, people: s.placed.filter((t) => t.roomId === s.probe().room).map((t) => `${t.id}@${t.x},${t.y}`) } : null,
  };
});
// Page coordinates of a tile's centre (the canvas sits below the header).
const tileToScreen = (x, y) => page.evaluate(([x, y]) => { const c = window.__stage.cameras.main; const r = document.querySelector('[data-testid=phaser-stage] canvas').getBoundingClientRect(); return { x: r.left + (x * 16 + 8 - c.worldView.x) * c.zoom, y: r.top + (y * 16 + 8 - c.worldView.y) * c.zoom }; }, [x, y]);

http.createServer(async (req, res) => {
  const u = new URL(req.url, 'http://x');
  const p = Object.fromEntries(u.searchParams);
  const body = await new Promise((r) => { let d = ''; req.on('data', (c) => (d += c)); req.on('end', () => r(d)); });
  const reply = (o) => { res.setHeader('Content-Type', 'application/json'); res.end(JSON.stringify(o, null, 1)); };
  try {
    switch (u.pathname) {
      case '/goto': await page.goto(p.url, { waitUntil: 'domcontentloaded' }); await page.waitForTimeout(+p.wait || 1500); return reply(await state());
      case '/state': return reply(await state());
      case '/log': { const since = +p.since || 0; const out = log.filter((e) => e.t > since); if (p.clear) log = []; return reply(out); }
      case '/shot': { const f = `${OUT}/${p.name || 'shot'}.png`; await page.screenshot({ path: f }); return reply({ file: f }); }
      case '/click': if (p.sel) await page.click(p.sel, { timeout: 5000 }); else await page.mouse.click(+p.x, +p.y); await page.waitForTimeout(+p.wait || 300); return reply(await state());
      case '/viewport': await page.setViewportSize({ width: +p.w, height: +p.h }); await page.waitForTimeout(+p.wait || 800); return reply(await state());
      case '/button': await page.getByRole('button', { name: p.name, exact: p.exact === '1' }).first().click({ timeout: 5000 }); await page.waitForTimeout(+p.wait || 300); return reply(await state());
      case '/text': await page.getByText(p.t, { exact: false }).first().click({ timeout: 5000 }); await page.waitForTimeout(+p.wait || 300); return reply(await state());
      case '/key': if (p.ms) { await page.keyboard.down(p.k); await page.waitForTimeout(+p.ms); await page.keyboard.up(p.k); } else await page.keyboard.press(p.k); await page.waitForTimeout(+p.wait || 300); return reply(await state());
      case '/type': await page.locator('[data-testid=stage-panel] input').fill(p.text); if (p.enter !== '0') await page.keyboard.press('Enter'); await page.waitForTimeout(+p.wait || 300); return reply(await state());
      case '/tile': { const at = await tileToScreen(+p.x, +p.y); await page.mouse.click(at.x, at.y); await page.waitForTimeout(+p.wait || 2500); return reply(await state()); }
      case '/talk': { const t = await page.evaluate((id) => window.__stage.placed.find((x) => x.id === id), p.id); const at = await tileToScreen(t.x, t.y); await page.mouse.click(at.x, at.y); await page.waitForTimeout(+p.wait || 3000); return reply(await state()); }
      case '/perf': { const r = await page.evaluate(() => { const f = window.__perf.frames; const span = f.at(-1) - f[0]; const gaps = f.slice(1).map((t, i) => t - f[i]); return { fps: Math.round((f.length - 1) / (span / 1000)), worstFrameMs: Math.round(Math.max(...gaps)), longTasks: window.__perf.long.slice(-10), heapMB: performance.memory ? Math.round(performance.memory.usedJSHeapSize / 1e6) : null }; }); return reply(r); }
      case '/eval': return reply(await page.evaluate(body));
      case '/wait': await page.waitForTimeout(+p.ms || 1000); return reply(await state());
      case '/until': { // wait until the story settles: no streaming, no typing, or timeout
        const t0 = Date.now(); const max = +p.ms || 30000;
        while (Date.now() - t0 < max) { const s = await state(); if (!s.playerLine && (s.continue || s.choices.length || s.input)) return reply({ waited: Date.now() - t0, ...s }); await page.waitForTimeout(250); }
        return reply({ waited: max, timeout: true, ...(await state()) });
      }
      default: return reply({ error: 'unknown' });
    }
  } catch (e) { reply({ error: String(e).slice(0, 500), state: await state().catch(() => null) }); }
}).listen(9333, () => console.log('driver on :9333'));
