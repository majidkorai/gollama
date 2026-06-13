package ui

import _ "embed"

//go:embed logo.svg
var LogoSVG string

const Page = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta name="theme-color" content="#0c0c12">
<link rel="icon" type="image/svg+xml" href="/logo.svg">
<title>gollama</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600&family=Plus+Jakarta+Sans:wght@400;500;600;700;800&display=swap" rel="stylesheet">
<style>
:root {
  --bg: #0c0c12;
  --surface: #14141e;
  --surface-2: #1c1c2a;
  --border: #26263a;
  --border-hover: #363654;
  --text: #e8e8f0;
  --text-muted: #8888a8;
  --text-dim: #5c5c7a;
  --accent: #00d4aa;
  --accent-hover: #00f0c0;
  --accent-bg: rgba(0, 212, 170, 0.08);
  --accent-glow: rgba(0, 212, 170, 0.2);
  --green: #00d4aa;
  --green-bg: rgba(0, 212, 170, 0.1);
  --red: #ff4060;
  --red-bg: rgba(255, 64, 96, 0.1);
  --amber: #ffaa40;
  --amber-bg: rgba(255, 170, 64, 0.1);
  --blue: #5090ff;
  --blue-bg: rgba(80, 144, 255, 0.1);
  --sidebar-w: 220px;
  --radius: 10px;
  --radius-sm: 6px;
  --font: 'Plus Jakarta Sans', system-ui, sans-serif;
  --font-mono: 'JetBrains Mono', 'SF Mono', monospace;
  --shadow: 0 1px 3px rgba(0,0,0,.3), 0 1px 2px rgba(0,0,0,.2);
  --shadow-lg: 0 4px 12px rgba(0,0,0,.4), 0 2px 4px rgba(0,0,0,.2);
  --transition: 180ms cubic-bezier(0.4, 0, 0.2, 1);
}
.light {
  --bg: #f4f4f8;
  --surface: #ffffff;
  --surface-2: #fafafe;
  --border: #d8d8e4;
  --border-hover: #b8b8cc;
  --text: #14141e;
  --text-muted: #686880;
  --text-dim: #9898b0;
  --accent: #009977;
  --accent-hover: #00b388;
  --accent-bg: rgba(0, 153, 119, 0.06);
  --accent-glow: rgba(0, 153, 119, 0.12);
  --green: #009977;
  --green-bg: rgba(0, 153, 119, 0.08);
  --red: #cc3344;
  --red-bg: rgba(204, 51, 68, 0.06);
  --amber: #b87722;
  --amber-bg: rgba(184, 119, 34, 0.08);
  --blue: #3366cc;
  --blue-bg: rgba(51, 102, 204, 0.06);
  --shadow: 0 1px 3px rgba(0,0,0,.06), 0 1px 2px rgba(0,0,0,.04);
  --shadow-lg: 0 4px 12px rgba(0,0,0,.08), 0 2px 4px rgba(0,0,0,.04);
}
* { margin: 0; padding: 0; box-sizing: border-box; }
:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; border-radius: 4px; }
html { color-scheme: dark; height: 100%; }
.light { color-scheme: light; }
body {
  font-family: var(--font); background: var(--bg); color: var(--text);
  display: flex; font-size: 14px; line-height: 1.6; height: 100%;
  -webkit-font-smoothing: antialiased;
}

/* ── Sidebar ─────────────────────────────────────────── */
.sidebar {
  width: var(--sidebar-w); background: var(--surface);
  border-right: 1px solid var(--border);
  display: flex; flex-direction: column; flex-shrink: 0;
  height: 100vh; position: sticky; top: 0; z-index: 10;
  transition: width var(--transition); overflow: hidden;
}
.sidebar.collapsed { width: 60px; }
.sidebar .logo {
  padding: 24px 20px 16px; border-bottom: 1px solid var(--border);
  display: flex; flex-direction: column; align-items: center;
  gap: 4px; color: var(--text); overflow: hidden;
}
.sidebar .logo img { width: 100%; max-width: 80px; height: auto; flex-shrink: 0; filter: brightness(0) invert(1); clip-path: inset(2px);}
.light .sidebar .logo img { filter: none; }
.sidebar.collapsed .logo img { max-width: 28px; }
.sidebar nav { flex: 1; padding: 10px 6px; display: flex; flex-direction: column; gap: 2px; }
.sidebar .nav-item {
  display: flex; align-items: center; gap: 10px; padding: 10px 12px;
  border-radius: var(--radius-sm); color: var(--text-muted);
  text-decoration: none; font-size: 13px; font-weight: 600; cursor: pointer;
  transition: all var(--transition); white-space: nowrap; overflow: hidden;
  border: none; background: none; width: 100%; text-align: left;
  font-family: var(--font);
}
.sidebar .nav-item:hover { background: var(--surface-2); color: var(--text); }
.sidebar .nav-item.active { background: var(--accent-bg); color: var(--accent); }
.sidebar .nav-item .icon { font-size: 16px; width: 22px; text-align: center; flex-shrink: 0; }
.sidebar .nav-item .label { transition: opacity var(--transition); white-space: nowrap; }
.sidebar.collapsed .nav-item .label { opacity: 0; width: 0; overflow: hidden; }
.sidebar .bottom { padding: 6px; border-top: 1px solid var(--border); width: 100%; display: flex; }
.sidebar .bottom .nav-item { width: 100%; justify-content: flex-start; }
.sidebar.collapsed .bottom { justify-content: center; }
.sidebar.collapsed .bottom .nav-item { justify-content: center; padding: 8px; }
.sidebar.collapsed .bottom .label { opacity: 0; width: 0; overflow: hidden; }

/* ── Main ────────────────────────────────────────────── */
.main { flex: 1; overflow-y: auto; padding: 32px 40px; position: relative; }
.main::before {
  content: ''; position: fixed; top: 0; left: 0; right: 0; bottom: 0;
  background-image:
    radial-gradient(circle at 1px 1px, var(--border) 1px, transparent 0);
  background-size: 32px 32px;
  pointer-events: none; z-index: 0; opacity: 0.5;
}
.main > * { position: relative; z-index: 1; }
.view { display: none; animation: fadeIn 250ms ease; }
.view.active { display: block; }

@keyframes fadeIn { from { opacity: 0; transform: translateY(6px); } to { opacity: 1; transform: translateY(0); } }
@keyframes slideUp { from { opacity: 0; transform: translateY(12px); } to { opacity: 1; transform: translateY(0); } }
@keyframes pulse { 0%,100%{opacity:1} 50%{opacity:.4} }
@keyframes spin { to{transform:rotate(360deg)} }

@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after { animation-duration: 0.01ms !important; transition-duration: 0.01ms !important; }
  .view { animation: none !important; }
}

/* ── Page Header ─────────────────────────────────────── */
.page-header { margin-bottom: 28px; }
.page-header h1 { font-size: 22px; font-weight: 800; letter-spacing: -.5px; text-wrap: balance; }
.page-header p { color: var(--text-muted); font-size: 13px; margin-top: 4px; }

/* ── Metrics ──────────────────────────────────────────── */
.metrics { display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 14px; margin-bottom: 32px; }
.metric-card {
  background: var(--surface); border: 1px solid var(--border);
  border-radius: var(--radius); padding: 18px 20px;
  transition: border-color var(--transition), box-shadow var(--transition);
  box-shadow: var(--shadow);
}
.metric-card:hover { border-color: var(--border-hover); box-shadow: var(--shadow-lg); }
.metric-card .label { font-size: 11px; color: var(--text-muted); font-weight: 600; text-transform: uppercase; letter-spacing: .6px; margin-bottom: 6px; }
.metric-card .value { font-size: 24px; font-weight: 800; letter-spacing: -.5px; font-variant-numeric: tabular-nums; }
.metric-card .value .accent { color: var(--accent); }
.metric-card .sub { font-size: 12px; color: var(--text-dim); margin-top: 2px; }

/* ── Section ──────────────────────────────────────────── */
.section { margin-bottom: 32px; }
.section-header {
  display: flex; align-items: center; justify-content: space-between; margin-bottom: 14px;
}
.section-header h2 {
  font-size: 11px; font-weight: 700; color: var(--text-muted);
  text-transform: uppercase; letter-spacing: .8px;
}

/* ── Cards ────────────────────────────────────────────── */
.card { background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius); box-shadow: var(--shadow); transition: border-color var(--transition); }
.card:hover { border-color: var(--border-hover); }
.card-body { padding: 18px; }

/* ── Instance Card ────────────────────────────────────── */
.instance-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(380px, 1fr)); gap: 14px; }
.inst-card {
  background: var(--surface); border: 1px solid var(--border);
  border-radius: var(--radius); padding: 18px;
  transition: border-color var(--transition), box-shadow var(--transition);
  box-shadow: var(--shadow);
  position: relative; overflow: hidden;
}
.inst-card::before {
  content: ''; position: absolute; top: 0; left: 0; width: 3px; height: 100%;
  background: var(--green); border-radius: 0 2px 2px 0;
  transition: background var(--transition);
}
.inst-card:hover { border-color: var(--border-hover); box-shadow: var(--shadow-lg); }
.inst-card.stopped::before { background: var(--text-dim); }
.inst-card.stopped { opacity: .55; }
.inst-card.error::before { background: var(--red); }
.inst-card .title { font-size: 14px; font-weight: 600; word-break: break-all; font-family: var(--font-mono); }
.inst-card .meta { font-size: 12px; color: var(--text-dim); margin-top: 8px; display: flex; gap: 14px; flex-wrap: wrap; align-items: center; }
.inst-card .actions { margin-top: 14px; display: flex; gap: 6px; flex-wrap: wrap; }

/* ── Quick Launch ──────────────────────────────────────── */
.launch-row { display: grid; grid-template-columns: 1fr 120px auto; gap: 10px; align-items: end; margin-bottom: 14px; }
.launch-row .field { display: flex; flex-direction: column; gap: 5px; }
.launch-row .field label { font-size: 11px; color: var(--text-dim); text-transform: uppercase; letter-spacing: .6px; font-weight: 700; }

/* ── Forms ─────────────────────────────────────────────── */
select, input[type=text], input[type=number] {
  width: 100%; padding: 9px 12px; background: var(--bg);
  border: 1px solid var(--border); border-radius: var(--radius-sm);
  color: var(--text); font-size: 13px; outline: none;
  transition: border-color var(--transition);
  font-family: var(--font);
}
select:focus, input:focus { border-color: var(--accent); }
select { cursor: pointer; appearance: none; background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' fill='%238888a8'%3E%3Cpath d='M2 4l4 4 4-4'/%3E%3C/svg%3E"); background-repeat: no-repeat; background-position: right 10px center; padding-right: 30px; }

/* ── Buttons ───────────────────────────────────────────── */
button {
  padding: 9px 18px; border-radius: var(--radius-sm); border: none;
  font-size: 13px; font-weight: 600; cursor: pointer;
  transition: all var(--transition); font-family: var(--font);
  display: inline-flex; align-items: center; gap: 6px;
  white-space: nowrap;
}
button.primary { background: var(--accent); color: #0c0c12; }
button.primary:hover { background: var(--accent-hover); transform: translateY(-1px); }
button.primary:active { transform: translateY(0); }
button.secondary { background: var(--surface-2); color: var(--text); border: 1px solid var(--border); }
button.secondary:hover { background: var(--border-hover); border-color: var(--border-hover); }
button.danger { background: var(--red); color: #fff; }
button.danger:hover { opacity: .9; }
button.small { padding: 6px 12px; font-size: 11px; }
button:disabled { opacity: .35; cursor: default; pointer-events: none; }
button.ghost { background: transparent; color: var(--text-muted); padding: 6px 8px; }
button.ghost:hover { background: var(--surface-2); color: var(--text); }

/* ── Badge ──────────────────────────────────────────────── */
.badge { display: inline-block; padding: 2px 8px; border-radius: 4px; font-size: 10px; font-weight: 700; letter-spacing: .3px; font-family: var(--font-mono); }
.badge-green { background: var(--green-bg); color: var(--green); }
.badge-red { background: var(--red-bg); color: var(--red); }
.badge-amber { background: var(--amber-bg); color: var(--amber); }
.badge-blue { background: var(--blue-bg); color: var(--blue); }

/* ── Advanced flags toggle ────────────────────────────── */
.advanced-flags summary { cursor: pointer; font-size: 12px; color: var(--text-muted); font-weight: 600; user-select: none; padding: 4px 0; }
.advanced-flags summary:hover { color: var(--text); }
.advanced-flags[open] summary { color: var(--accent); }
.advanced-flags[open] summary::after { content: ''; }

/* ── Flag rows ─────────────────────────────────────────── */
.flag-row { display: flex; gap: 6px; margin-bottom: 6px; align-items: center; }
.flag-row select.flag-name { width: auto; min-width: 160px; flex-shrink: 0; }
.flag-row input.flag-custom { flex: 1; }
.flag-row input.flag-value { flex: 1; }

/* ── Empty state ───────────────────────────────────────── */
.empty-state { text-align: center; padding: 48px 20px; color: var(--text-dim); }
.instance-grid .empty-state { grid-column: 1 / -1; display: flex; flex-direction: column; align-items: center; justify-content: center; min-height: 200px; }
.empty-state .icon { font-size: 36px; margin-bottom: 12px; opacity: .6; }
.empty-state .title { font-size: 15px; font-weight: 600; color: var(--text-muted); margin-bottom: 4px; }
.empty-state p { font-size: 13px; }

/* ── Chat ──────────────────────────────────────────────── */
.chat-container { display: flex; flex-direction: column; height: calc(100vh - 64px); }
.chat-header { padding-bottom: 14px; margin-bottom: 14px; border-bottom: 1px solid var(--border); }
.chat-header select { width: auto; min-width: 220px; }
.chat-msgs { flex: 1; overflow-y: auto; padding: 6px 0; margin-bottom: 12px; }
.chat-msgs .msg { margin-bottom: 12px; padding: 12px 16px; border-radius: 12px; max-width: 80%; line-height: 1.7; font-size: 13.5px; animation: slideUp 200ms ease both; position: relative; }
.chat-msgs .user { background: var(--accent-bg); margin-left: auto; border-bottom-right-radius: 4px; color: var(--text); }
.chat-msgs .assistant { background: var(--surface); border: 1px solid var(--border); border-bottom-left-radius: 4px; padding-right: 40px; }
.chat-msgs .system { background: transparent; color: var(--text-dim); font-style: italic; font-size: 12px; text-align: center; max-width: 100%; }
.chat-msgs .assistant .copy-btn { position: absolute; top: 8px; right: 8px; font-size: 12px; background: none; border: none; cursor: pointer; opacity: 0; padding: 2px 4px; border-radius: 4px; transition: opacity var(--transition), background var(--transition); }
.chat-msgs .assistant:hover .copy-btn { opacity: 0.5; }
.chat-msgs .assistant .copy-btn:hover { opacity: 1; background: var(--surface-2); }
.chat-input-row { display: flex; gap: 8px; padding-top: 12px; border-top: 1px solid var(--border); align-items: flex-end; }
.chat-input-row textarea { flex: 1; max-height: 200px; }
.chat-loading { animation: pulse 1.2s infinite; display: inline-block; letter-spacing: 4px; font-size: 18px; line-height: 1; color: var(--text-dim); }
.reasoning { color: var(--text-dim); font-style: italic; font-size: 12px; border-left: 2px solid var(--accent); padding-left: 12px; margin-bottom: 8px; opacity: .85; }

/* ── Logs Modal ────────────────────────────────────────── */
.modal { display: none; position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,.75); z-index: 100; animation: fadeIn 150ms ease; }
.modal-content { background: var(--surface); margin: 6% auto; padding: 24px; width: 86%; max-width: 720px; max-height: 72vh; border-radius: var(--radius); overflow: auto; border: 1px solid var(--border); box-shadow: var(--shadow-lg); overscroll-behavior: contain; }
.modal-content h2 { font-size: 15px; font-weight: 700; }
.modal-content pre { background: var(--bg); padding: 14px; border-radius: var(--radius-sm); margin-top: 14px; font-size: 11.5px; line-height: 1.5; overflow: auto; max-height: 55vh; white-space: pre-wrap; color: var(--text-muted); font-family: var(--font-mono); }

/* ── Error line ───────────────────────────────────────── */
.error-line { font-size: 11px; color: var(--red); margin-top: 8px; padding: 6px 10px; background: var(--red-bg); border-radius: var(--radius-sm); word-break: break-all; font-family: var(--font-mono); }

/* ── Model list ────────────────────────────────────────── */
.model-row { display: flex; justify-content: space-between; align-items: center; padding: 12px 14px; border-radius: var(--radius-sm); transition: background var(--transition); cursor: pointer; }
.model-row:hover .name { color: var(--accent); }
.model-row .info-icon { font-size: 11px; opacity: 0; transition: opacity var(--transition); margin-left: 4px; }
.model-row:hover .info-icon { opacity: 0.6; }
.model-row:hover { background: var(--surface-2); }
.model-row .name { font-size: 13px; font-weight: 500; word-break: break-all; }
.model-row .info { font-size: 11px; color: var(--text-dim); margin-top: 4px; display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }

/* ── Animations / Utilities ────────────────────────────── */
.spinner { display: inline-block; width: 14px; height: 14px; border: 2px solid var(--border); border-top-color: var(--accent); border-radius: 50%; animation: spin .6s linear infinite; vertical-align: middle; }
.refreshing { opacity: .5; pointer-events: none; transition: opacity var(--transition); }
.tag { display: inline-block; padding: 1px 6px; border-radius: 3px; font-size: 10px; font-weight: 600; background: var(--surface-2); color: var(--text-muted); font-family: var(--font-mono); }
.tps { font-variant-numeric: tabular-nums; }

/* ── Pull suggestions dropdown ──────────────────────── */
.suggestion { display: flex; justify-content: space-between; align-items: center; padding: 8px 12px; cursor: pointer; transition: background var(--transition); }
.suggestion:hover { background: var(--surface-2); }
.suggestion + .suggestion { border-top: 1px solid var(--border); }
.suggestion .name { font-size: 12px; font-weight: 500; word-break: break-all; }
.suggestion .meta { font-size: 10px; color: var(--text-dim); margin-top: 1px; display: flex; gap: 8px; }
.suggestion .badge { font-size: 9px; }
.suggestion .pull-btn { font-size: 10px; padding: 2px 8px; flex-shrink: 0; margin-left: 8px; }

/* ── Pull model row ────────────────────────────────────── */
.pull-row { display: flex; gap: 8px; margin-bottom: 12px; }
.pull-row input { flex: 1; }

/* ── Responsive ────────────────────────────────────────── */
@media (max-width: 768px) {
  .sidebar { width: 60px; }
  .sidebar .logo { padding: 12px 8px; }
  .sidebar .logo img { max-width: 28px; }
  .sidebar .nav-item .label { display: none; }
  .sidebar .nav-item { justify-content: center; padding: 10px; }
  .sidebar .bottom button .label { display: none; }
  .main { padding: 20px 16px; }
  .metrics { grid-template-columns: 1fr 1fr; }
  .launch-row { grid-template-columns: 1fr; }
  .instance-grid { grid-template-columns: 1fr; }
  .chat-msgs .msg { max-width: 92%; }
}
</style>
</head>
<body>

<!-- ─── Sidebar ──────────────────────────────────────── -->
<aside class="sidebar" role="navigation" aria-label="Main navigation">
  <div class="logo" style="flex-direction:column; gap:4px; padding:20px 16px">
    <img src="/logo.svg" alt="gollama" style="display:block; width:100%; max-width:80px; height:auto">
    <button class="toggle" onclick="toggleSidebar()" aria-label="Toggle sidebar" style="font-size:11px; color:var(--text-dim); background:none; border:none; cursor:pointer; padding:2px; margin-top:2px">◀</button>
  </div>
  <nav>
    <button class="nav-item active" onclick="switchView('dashboard')" aria-label="Dashboard">
      <span class="icon">📊</span><span class="label">Dashboard</span>
    </button>
    <button class="nav-item" onclick="switchView('models')" aria-label="Models">
      <span class="icon">📦</span><span class="label">Models</span>
    </button>
    <button class="nav-item" onclick="switchView('chat')" aria-label="Chat">
      <span class="icon">💬</span><span class="label">Chat</span>
    </button>
    <button class="nav-item" onclick="switchView('settings')" aria-label="Settings">
      <span class="icon">⚙️</span><span class="label">Settings</span>
    </button>
  </nav>
  <div class="bottom">
    <button class="nav-item" onclick="toggleTheme()" id="themeToggle" aria-label="Toggle theme">
      <span class="icon">🌙</span><span class="label">Theme</span>
    </button>
  </div>
</aside>

<!-- ─── Main ──────────────────────────────────────────── -->
<main class="main" id="main">

<!-- ── Dashboard ────────────────────────────────────── -->
<div id="view-dashboard" class="view active" role="tabpanel" aria-label="Dashboard">
  <div class="page-header">
    <h1>Dashboard</h1>
    <p>Monitor and manage your llama.cpp instances</p>
  </div>

  <div class="section" id="launch-section">
    <div class="section-header"><h2>Quick Launch</h2></div>
    <div class="card"><div class="card-body">
      <div class="launch-row">
        <div class="field"><label for="modelSelect">Model</label><select id="modelSelect" autocomplete="off"><option value="">Loading…</option></select></div>
        <div class="field"><label for="portInput">Port</label><input type="number" id="portInput" value="8081" min="8081" max="8099" autocomplete="off"></div>
        <div class="field" style="align-self: end"><button class="primary" onclick="launchInstance()" id="launchBtn">Launch</button></div>
      </div>
      <details class="advanced-flags" style="margin-top:8px">
        <summary style="cursor:pointer;font-size:12px;color:var(--text-muted);font-weight:600;user-select:none">Advanced flags</summary>
        <div style="margin-top:8px">
          <div id="flagsContainer"></div>
          <button class="ghost small" onclick="addFlag()" style="margin-top:4px">＋ Add Flag</button>
        </div>
      </details>
      <div style="margin-top:10px;display:flex;gap:6px;flex-wrap:wrap;align-items:center">
        <select id="presetSelect" onchange="applyPreset()" style="width:auto;min-width:140px;flex:1">
          <option value="">Presets</option>
        </select>
        <button class="small secondary" onclick="savePreset()" id="savePresetBtn" style="font-size:11px" title="Save current flags as preset">💾 Save</button>
        <button class="small danger" onclick="deletePreset()" id="deletePresetBtn" style="font-size:11px;display:none" title="Delete preset">🗑</button>
      </div>
    </div></div>
  </div>

  <div class="section">
    <div class="section-header">
      <h2>Running Instances</h2>
      <span class="badge badge-blue" id="instanceCount"></span>
    </div>
    <div id="instances" class="instance-grid">
      <div class="empty-state">
        <div class="icon">🚀</div>
        <div class="title">No running instances</div>
        <p>Launch one above to get started.</p>
      </div>
    </div>
  </div>
</div>

<!-- ── Models ────────────────────────────────────────── -->
<div id="view-models" class="view" role="tabpanel" aria-label="Models">
  <div class="page-header">
    <h1>Models</h1>
    <p id="modelCount"><span class="spinner" id="modelCountSpinner"></span> Loading…</p>
  </div>
  <div id="modelList" class="card"><div class="card-body"><div class="empty-state"><span class="spinner"></span> Loading models…</div></div></div>
  <div class="section" style="margin-top:32px">
    <div class="section-header"><h2>Pull Model</h2></div>
    <div class="card"><div class="card-body">
      <div class="pull-row" style="position:relative">
        <input type="text" id="pullInput" placeholder="Search or enter hf.co/user/repo:Q4_K_M…" autocomplete="off" oninput="onPullInputChange(this.value)">
        <button class="primary" onclick="pullModel()" id="pullBtn">Pull</button>
      </div>
      <div id="pullSuggestions" style="display:none;margin-top:4px;border:1px solid var(--border);border-radius:var(--radius-sm);overflow:hidden"></div>
      <div id="pullProgress" style="display:none;margin-top:10px">
        <div style="display:flex;justify-content:space-between;font-size:11px;color:var(--text-muted);margin-bottom:4px">
          <span id="pullPct">0%</span>
          <span id="pullSpeed"></span>
        </div>
        <div style="height:6px;background:var(--border);border-radius:3px;overflow:hidden">
          <div id="pullBar" style="height:100%;width:0%;background:var(--accent);border-radius:3px;transition:width 200ms ease"></div>
        </div>
        <div id="pullStatus" style="font-size: 11px; color: var(--text-muted); margin-top: 4px;"></div>
      </div>
    </div></div>
  </div>
</div>

<!-- ── Model Details Modal ──────────────────────────── -->
<div class="modal" id="modelModal" role="dialog" aria-modal="true" aria-label="Model details">
  <div class="modal-content" onclick="event.stopPropagation()" style="max-width:500px">
    <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:12px">
      <h2 id="modelModalTitle">Model</h2>
      <button class="small danger" onclick="closeModelDetails()" aria-label="Close">Close</button>
    </div>
    <div id="modelModalBody" style="line-height:2">
      <div class="detail-row"><span class="detail-label">Architecture</span><span class="detail-value" id="md-arch">—</span></div>
      <div class="detail-row"><span class="detail-label">Quantization</span><span class="detail-value" id="md-quant">—</span></div>
      <div class="detail-row"><span class="detail-label">Context Length</span><span class="detail-value" id="md-ctx">—</span></div>
      <div class="detail-row"><span class="detail-label">Size</span><span class="detail-value" id="md-size">—</span></div>
      <div class="detail-row"><span class="detail-label">Path</span><span class="detail-value" id="md-path" style="word-break:break-all;font-family:var(--font-mono);font-size:11px">—</span></div>
    </div>
    <button class="primary" onclick="launchModelFromDetails()" id="md-launch-btn" style="margin-top:16px;width:100%">🚀 Launch</button>
  </div>
</div>

<style>
.detail-row { display: flex; justify-content: space-between; align-items: center; padding: 4px 0; border-bottom: 1px solid var(--border); }
.detail-row:last-child { border-bottom: none; }
.detail-label { font-size: 12px; color: var(--text-muted); font-weight: 600; text-transform: uppercase; letter-spacing: .5px; }
.detail-value { font-size: 13px; color: var(--text); text-align: right; }
</style>

<!-- ── Chat History Modal ──────────────────────────── -->
<div class="modal" id="chatHistoryModal" role="dialog" aria-modal="true" aria-label="Chat history">
  <div class="modal-content" onclick="event.stopPropagation()" style="max-width:520px">
    <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:12px">
      <h2>📋 Chat History</h2>
      <button class="small danger" onclick="closeChatHistory()" aria-label="Close">Close</button>
    </div>
    <div id="chatHistoryList" style="max-height:50vh;overflow-y:auto">
      <div class="empty-state"><span class="spinner"></span> Loading…</div>
    </div>
  </div>
</div>

<!-- ── Chat ──────────────────────────────────────────── -->
<div id="view-chat" class="view" role="tabpanel" aria-label="Chat">
  <div class="chat-container">
    <div class="chat-header">
      <div style="display: flex; align-items: center; gap: 12px; flex-wrap: wrap">
        <h1 style="font-size: 18px; font-weight: 800">Chat</h1>
        <select id="chatInstanceSelect" onchange="selectChatInstance()" aria-label="Select instance"><option value="">— select a running instance —</option></select>
        <button class="ghost small" onclick="selectChatFor(chatPort, '')" aria-label="Refresh">↻</button>
        <button class="ghost small" onclick="showChatHistory()" aria-label="Chat history" title="Chat history">📋</button>
        <button class="ghost small" onclick="clearChat()" aria-label="Clear chat" title="Clear chat" style="color:var(--text-dim)">✕</button>
      </div>
    </div>
    <div id="chatPanel" class="chat-msgs" style="display: none" aria-live="polite"></div>
    <div id="chatEmpty" class="empty-state" style="flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center">
      <div class="icon">💬</div>
      <div class="title">No instance selected</div>
      <p>Launch an instance from the Dashboard to start chatting</p>
    </div>
    <div class="chat-input-row">
      <textarea id="chatInput" rows="1" placeholder="Type a message… (Enter to send)" onkeydown="if(event.key=='Enter'&&!event.shiftKey){event.preventDefault();sendChat()}" autocomplete="off" style="resize:none;padding:9px 12px;font-family:inherit;font-size:13px;line-height:1.5;background:var(--bg);border:1px solid var(--border);border-radius:var(--radius-sm);color:var(--text);outline:none;width:100%" oninput="autoGrow(this)"></textarea>
      <button class="primary" onclick="sendChat()">Send</button>
    </div>
  </div>
</div>

<!-- ── Settings ──────────────────────────────────────── -->
<div id="view-settings" class="view" role="tabpanel" aria-label="Settings">
  <div class="page-header">
    <h1>Settings</h1>
    <p>Configuration and system information</p>
  </div>
  <div class="card"><div class="card-body" style="line-height: 2.2">
    <div style="font-size: 13px; color: var(--text-muted);">
      <div><strong style="color: var(--text)">gollama</strong> <span class="tag" id="s-version">—</span></div>
      <div><strong style="color: var(--text)">llama-server</strong> <span class="tag" id="s-llama-version">—</span></div>
      <div><strong style="color: var(--text)">Backend</strong> <span class="tag" id="s-backend">—</span></div>
      <div><strong style="color: var(--text)">Config</strong> <code style="font-size: 11px; color: var(--text-dim)">~/.gollama/config.json</code></div>
      <div><strong style="color: var(--text)">Models dir</strong> <code style="font-size: 11px; color: var(--text-dim)">~/.gollama/models/</code></div>
    </div>
  </div></div>
  <div class="card" style="margin-top:16px"><div class="card-body">
    <div style="display:flex;align-items:center;gap:12px;flex-wrap:wrap">
      <label for="idleTtlInput" style="font-size:13px;font-weight:600">Auto-stop idle instances after</label>
      <input type="number" id="idleTtlInput" value="30" min="0" max="1440" style="width:70px;text-align:center">
      <span style="font-size:12px;color:var(--text-muted)">minutes (0 = disable)</span>
      <button class="primary small" onclick="saveIdleTTL()">Save</button>
      <span id="idleTtlStatus" style="font-size:12px;color:var(--text-muted)"></span>
    </div>
  </div></div>
</div>

<!-- ── Logs Modal ──────────────────────────────────── -->
<div class="modal" id="logModal" role="dialog" aria-modal="true" aria-label="Instance logs">
  <div class="modal-content" onclick="event.stopPropagation()">
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 4px">
      <h2>📋 Logs</h2>
      <button class="small danger" onclick="closeLogs()" aria-label="Close logs">Close</button>
    </div>
    <pre id="logContent" aria-live="polite"></pre>
  </div>
</div>

<script>
var chatPort = 0, chatHistory = [], chatSessionId = null;
var currentView = 'dashboard';
var cachedModelCount = 0;
var cachedModels = [];

// ── Navigation ──────────────────────────────────────
function switchView(name) {
  document.querySelectorAll('.view').forEach(function(v) { v.classList.remove('active'); });
  document.querySelectorAll('.nav-item').forEach(function(a) { a.classList.remove('active'); });
  var view = document.getElementById('view-' + name);
  if (view) view.classList.add('active');
  document.querySelector('.nav-item[onclick*="' + name + '"]').classList.add('active');
  currentView = name;
  if (name == 'dashboard') loadInstances();
  if (name == 'chat' && chatPort) selectChatFor(chatPort, '');
  if (name == 'models') loadModels();
}

// ── Models ───────────────────────────────────────────
async function loadModels() {
  var mc = document.getElementById('modelCount'), ml = document.getElementById('modelList'), ms = document.getElementById('modelCountSpinner');
  var s = document.getElementById('modelSelect');
  if (ms) ms.style.display = 'inline-block';
  if (mc) mc.innerHTML = '<span class="spinner"></span> Loading…';
  ml.classList.add('refreshing');
  try {
    var r = await fetch('/api/v1/models'), m = await r.json();
    cachedModelCount = m.length;
    mc.innerHTML = m.length + ' downloaded';

    s.innerHTML = '<option value="">— Select model —</option>';
    if (!m || !m.length) {
      s.innerHTML += '<option value="" disabled>No models found. Use gollama pull.</option>';
      ml.innerHTML = '<div class="card-body"><div class="empty-state"><div class="icon">📦</div><div class="title">No models yet</div><p>Pull a model from HuggingFace to get started.</p></div></div>';
      return;
    }

    cachedModels = m;
    var names = [], seen = {};
    m.forEach(function(x) { var n = x.name || '?'; if (!seen[n]) { seen[n] = 1; names.push(n); } });
    names.forEach(function(n, i) { s.innerHTML += '<option value="' + escAttr(n) + '"' + (names.length === 1 ? ' selected' : '') + '>' + escHtml(n) + '</option>'; });
    ml.innerHTML = '<div class="card-body">' + m.map(function(x) {
      var name = x.name || '?', size = x.size ? fmtSize(x.size) : '?';
      var arch = x.architecture || '', quant = x.quantization || '', ctx = x.context_length || 0, badges = [];
      if (quant) badges.push('<span class="badge badge-blue">' + escHtml(quant) + '</span>');
      if (arch) badges.push('<span class="badge badge-amber">' + escHtml(arch) + '</span>');
      if (ctx) badges.push('<span class="badge badge-green">' + (ctx > 999 ? Math.round(ctx / 1000) + 'K' : '<1K') + ' ctx</span>');
      return '<div class="model-row" onclick="showModelDetails(\'' + escAttr(name.replace(/'/g, '')) + '\')"><div><div class="name">' + escHtml(name.length > 55 ? name.slice(0, 55) + '…' : name) + ' <span class="info-icon">ⓘ</span></div><div class="info">' + size + ' ' + (badges.length ? badges.join(' ') : '') + '</div></div><button class="small danger" onclick="event.stopPropagation();deleteModel(\'' + escAttr(name.replace(/'/g, '')) + '\')" aria-label="Delete ' + escAttr(name) + '">🗑</button></div>';
    }).join('') + '</div>';
  } catch (e) {
    mc.textContent = 'Error loading models';
  }
  ml.classList.remove('refreshing');
  if (ms) ms.style.display = 'none';
}

async function deleteModel(name) {
  if (!confirm('Delete model "' + name + '"?')) return;
  try {
    await fetch('/api/v1/models/delete', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name: name }) });
    loadModels();
  } catch (e) { alert('Error deleting model: ' + e); }
}

// ── Model Details ────────────────────────────────────
var detailModelName = '';

function showModelDetails(name) {
  detailModelName = name;
  var m = null;
  for (var i = 0; i < cachedModels.length; i++) { if (cachedModels[i].name == name) { m = cachedModels[i]; break; } }
  if (!m) return;
  document.getElementById('modelModalTitle').textContent = m.name.length > 50 ? m.name.slice(0, 50) + '…' : m.name;
  document.getElementById('md-arch').textContent = m.architecture || '—';
  document.getElementById('md-quant').textContent = m.quantization || '—';
  document.getElementById('md-ctx').textContent = m.context_length ? (m.context_length > 999 ? Math.round(m.context_length / 1000) + 'K' : m.context_length.toString()) : '—';
  document.getElementById('md-size').textContent = m.size ? fmtSize(m.size) : '—';
  document.getElementById('md-path').textContent = m.blob_path || '—';
  document.getElementById('modelModal').style.display = 'block';
}

function closeModelDetails() {
  document.getElementById('modelModal').style.display = 'none';
}

function launchModelFromDetails() {
  closeModelDetails();
  document.getElementById('modelSelect').value = detailModelName;
  switchView('dashboard');
}

document.getElementById('modelModal').addEventListener('click', closeModelDetails);

// ── Instances + Metrics ──────────────────────────────
async function loadInstances() {
  var ic = document.getElementById('instanceCount'), c = document.getElementById('instances'), cs = document.getElementById('chatInstanceSelect');
  try {
    var r = await fetch('/api/v1/instances'), list = await r.json();
    ic.textContent = '(' + list.length + ')';
    var running = list.filter(function(i) { return i.status == 'running'; });

    cs.innerHTML = '<option value="">— select a running instance —</option>';
    list.forEach(function(i) { var mn = i.model || '?'; cs.innerHTML += '<option value="' + i.port + '"' + (chatPort == i.port ? ' selected' : '') + '>' + i.port + ' - ' + (mn.length > 35 ? mn.slice(0, 35) + '…' : escHtml(mn)) + '</option>'; });
    if (!list.length) {
      document.getElementById('chatPanel').style.display = 'none';
      document.getElementById('chatEmpty').style.display = 'flex';
      c.innerHTML = '<div class="empty-state"><div class="icon">🚀</div><div class="title">No running instances</div><p>Launch one from the Quick Launch section above.</p></div>';
      return;
    }
    document.getElementById('chatPanel').style.display = chatPort ? 'block' : 'none';

    c.setAttribute('data-list', JSON.stringify(list));
    c.innerHTML = list.map(function(i) {
      var cls = i.status == 'running' ? '' : ' stopped';
      var bc = i.status == 'running' ? 'badge-green' : 'badge-red';
      var mn = i.model || '?';
      var tps = i.tokens_per_sec ? '<span style="color: var(--green); font-variant-numeric: tabular-nums">⚡ ' + i.tokens_per_sec.toFixed(1) + ' t/s</span>' : '';
      var uptime = i.started_at ? (function() { var s = Math.floor((Date.now() - new Date(i.started_at).getTime()) / 1000); return '<span title="Uptime">⏱ ' + (s > 86400 ? Math.floor(s/86400)+'d ' : '') + (s > 3600 ? Math.floor((s%86400)/3600)+'h ' : '') + Math.floor((s%3600)/60)+'m</span>'; })() : '';
      var idle = i.last_activity ? (function() { var s = Math.floor((Date.now() - new Date(i.last_activity).getTime()) / 1000); if (s < 60) return ''; return '<span title="Idle time">💤 ' + (s > 3600 ? Math.floor(s/3600)+'h ' : '') + Math.floor((s%3600)/60)+'m</span>'; })() : '';
      var tokens = i.total_tokens ? '<span title="Total tokens">🔤 ' + (i.total_tokens > 999 ? Math.round(i.total_tokens/1000) + 'K' : i.total_tokens) + '</span>' : '';
      var flags = i.flags && i.flags.length ? formatFlags(i.flags) : '';
      var flagsHtml = flags ? '<div style="font-size: 11px; color: var(--text-dim); margin-top: 8px; padding-top: 8px; border-top: 1px solid var(--border); word-break: break-all; font-family: var(--font-mono)">' + escHtml(flags) + '</div>' : '';
      var errDiv = i.status != 'running' ? '<div class="error-line" id="err-' + i.port + '"></div>' : '';
      return '<div class="inst-card' + cls + '"><div class="title">' + escHtml(mn.length > 40 ? mn.slice(0, 40) + '…' : mn) + '</div>' +
        '<div class="meta"><span>Port ' + i.port + '</span>' + tps + uptime + idle + tokens + '<span class="badge ' + bc + '">' + i.status + '</span></div>' +
        errDiv + flagsHtml +
        '<div class="actions"><button class="small danger" onclick="stopInstance(' + i.port + ')" aria-label="Stop instance on port ' + i.port + '">⏹ Stop</button>' +
        '<button class="small secondary" onclick="restartInstance(' + i.port + ')" aria-label="Restart instance on port ' + i.port + '">🔄 Restart</button>' +
        '<button class="small secondary" onclick="selectChatFor(' + i.port + ', \'' + escAttr(mn.replace(/'/g, '')) + '\')" aria-label="Chat with instance on port ' + i.port + '">💬 Chat</button>' +
        '<button class="small secondary" onclick="window.open(\'http://\' + location.hostname + \':' + i.port + '\', \'_blank\'); return false" aria-label="Open instance on port ' + i.port + '">🌐 Open</button>' +
        '<button class="small secondary" onclick="viewLogs(' + i.port + ')" aria-label="View logs for port ' + i.port + '">📋 Logs</button></div></div>';
    }).join('');
    list.forEach(function(i) { if (i.status != 'running') fetchErrorLog(i.port); });
  } catch (e) { /* retry on next poll */ }
}

async function launchInstance() {
  var btn = document.getElementById('launchBtn'), m = document.getElementById('modelSelect').value, p = parseInt(document.getElementById('portInput').value), f = [];
  document.querySelectorAll('#flagsContainer .flag-row').forEach(function(row) {
    var sel = row.querySelector('.flag-name'), valInput = row.querySelector('.flag-value'), customInput = row.querySelector('.flag-custom');
    var name = sel.value || customInput.value.trim(), val = valInput.value.trim();
    if (!name) return;
    f.push(name);
    if (val) f.push(val);
  });
  if (!m) { alert('Select a model'); return; }
  var orig = btn.textContent; btn.disabled = true; btn.textContent = 'Launching…';
  try {
    var r = await fetch('/api/v1/instances', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ model: m, port: p, flags: f, replace_flags: true }) });
    if (!r.ok) { var e = await r.text(); alert('Error: ' + e); return; }
    var inst = await r.json();
    document.getElementById('portInput').value = (inst.port || 0) + 1;
    loadInstances();
  } catch (e) { alert('Error: ' + e); }
  finally { btn.disabled = false; btn.textContent = orig; }
}

async function stopInstance(p) {
  if (!confirm('Stop instance on port ' + p + '?')) return;
  try {
    await fetch('/api/v1/instances/stop?port=' + p, { method: 'POST' });
    loadInstances();
    if (chatPort == p) { chatPort = 0; document.getElementById('chatPanel').style.display = 'none'; document.getElementById('chatEmpty').style.display = 'flex'; }
  } catch (e) { alert('Error: ' + e); }
}

async function restartInstance(port) {
  if (!confirm('Restart instance on port ' + port + '?')) return;
  var list = JSON.parse(document.querySelector('#instances').getAttribute('data-list') || '[]');
  var inst = null;
  for (var i = 0; i < list.length; i++) { if (list[i].port == port) { inst = list[i]; break; } }
  if (!inst) return;

  await fetch('/api/v1/instances/stop?port=' + port, { method: 'POST' });

  var userFlags = [];
  document.querySelectorAll('#flagsContainer .flag-row').forEach(function(row) {
    var sel = row.querySelector('.flag-name'), valInput = row.querySelector('.flag-value'), customInput = row.querySelector('.flag-custom');
    var name = sel.value || customInput.value.trim(), val = valInput.value.trim();
    if (!name) return;
    userFlags.push(name);
    if (val) userFlags.push(val);
  });

  try {
    var r = await fetch('/api/v1/instances', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ model: inst.model, port: inst.port, flags: userFlags, replace_flags: true }) });
    if (r.ok) { loadInstances(); }
  } catch (e) {}
}

// ── Error Log ──────────────────────────────────────────
async function fetchErrorLog(port) {
  try {
    var r = await fetch('/api/v1/instances/logs?port=' + port), d = await r.json();
    if (d.error || !d.lines || !d.lines.length) return;
    var el = document.getElementById('err-' + port);
    if (!el) return;
    for (var i = d.lines.length - 1; i >= 0; i--) {
      var l = d.lines[i].trim();
      if (l && !l.includes('\r')) { el.textContent = '⚠️ ' + l; break; }
    }
  } catch (e) {}
}

// ── Flags ─────────────────────────────────────────────
var commonFlags = ['--ctx-size','--flash-attn','--temp','--repeat-penalty','--top-k','--top-p','--min-p','--presence-penalty','--frequency-penalty','--mirostat','--seed','--n-gpu-layers','--host','--port','--mlock','--no-flash-attn','--no-kv-offload','--threads','--batch-size','--ubatch-size','--parallel','--keep','--predict','--no-mmap','--cache-type-k','--cache-type-v','--reasoning','--reasoning-budget'];
var standaloneFlags = {'--mlock':1,'--no-flash-attn':1,'--no-kv-offload':1};
var flagHints = {
  '--ctx-size': 'number of tokens, e.g. 4096',
  '--flash-attn': 'on, off, or auto',
  '--temp': '0.0–2.0 (default 0.7)',
  '--repeat-penalty': '1.0–1.5 (default 1.0)',
  '--top-k': 'e.g. 40 (0 = off)',
  '--top-p': '0.0–1.0 (default 0.95)',
  '--min-p': '0.0–1.0 (default 0.05)',
  '--presence-penalty': '0.0–2.0',
  '--frequency-penalty': '0.0–2.0',
  '--mirostat': '0=off, 1=MIROSTAT, 2=MIROSTAT 2.0',
  '--seed': 'RNG seed, -1 = random',
  '--n-gpu-layers': 'number or "all"',
  '--host': 'IP address (default 0.0.0.0)',
  '--port': 'port number (default 8080)',
  '--threads': 'CPU thread count',
  '--batch-size': 'max batch size (default 2048)',
  '--ubatch-size': 'physical max batch size (default 512)',
  '--parallel': 'number of server slots',
  '--keep': 'tokens to keep from prompt (-1 = all)',
  '--predict': 'tokens to predict (-1 = infinity)',
  '--cache-type-k': 'f32, f16, q8_0, q4_0, ...',
  '--cache-type-v': 'f32, f16, q8_0, q4_0, ...',
  '--reasoning': 'on, off, or auto',
  '--reasoning-budget': 'token budget for thinking (-1 = unlimited)',
};

function flagOptions(selected) {
  return commonFlags.map(function(f) { return '<option value="' + escAttr(f) + '"' + (f === selected ? ' selected' : '') + '>' + escHtml(f) + '</option>'; }).join('') +
    '<option value=""' + (!selected || commonFlags.indexOf(selected) === -1 ? ' selected' : '') + '>Custom…</option>';
}

function makeFlagRow(name, value) {
  var r = document.createElement('div'); r.className = 'flag-row';
  var isCustom = name && commonFlags.indexOf(name) === -1;
  var displayName = isCustom ? '' : (name || '');
  r.innerHTML = '<select class="flag-name" onchange="onFlagChange(this)">' + flagOptions(displayName) + '</select>' +
    '<input type="text" class="flag-custom" placeholder="Flag name" autocomplete="off" style="display:' + (isCustom ? '' : 'none') + '" value="' + (isCustom ? escAttr(name) : '') + '">' +
    '<input type="text" class="flag-value" placeholder="Value" autocomplete="off" value="' + escAttr(value || '') + '">' +
    '<button class="small danger" onclick="this.parentElement.remove()" aria-label="Remove flag">\u2715</button>';
  if (!isCustom && standaloneFlags[name]) r.querySelector('.flag-value').style.display = 'none';
  return r;
}

function addFlag() {
  document.getElementById('flagsContainer').appendChild(makeFlagRow('', ''));
}

function onFlagChange(sel) {
  var row = sel.parentElement, valInput = row.querySelector('.flag-value');
  row.querySelector('.flag-custom').style.display = sel.value === '' ? '' : 'none';
  var isStandalone = standaloneFlags[sel.value];
  valInput.style.display = isStandalone ? 'none' : '';
  valInput.placeholder = flagHints[sel.value] || 'Value';
}

// ── Presets ────────────────────────────────────────────
async function loadPresets() {
  try {
    var r = await fetch('/api/v1/presets'), presets = await r.json();
    var sel = document.getElementById('presetSelect');
    sel.innerHTML = '<option value="">— Presets —</option>';
    var hasPresets = false;
    for (var name in presets) { hasPresets = true; sel.innerHTML += '<option value="' + escAttr(name) + '">' + escHtml(name) + '</option>'; }
    document.getElementById('deletePresetBtn').style.display = hasPresets ? '' : 'none';
  } catch (e) {}
}

function applyPreset() {
  var sel = document.getElementById('presetSelect'), name = sel.value;
  if (!name) return;
  fetch('/api/v1/presets').then(function(r) { return r.json(); }).then(function(presets) {
    var flags = presets[name];
    if (!flags) return;
    var c = document.getElementById('flagsContainer'); c.innerHTML = '';
    for (var i = 0; i < flags.length; i += 2) {
      var fname = flags[i], fval = (i + 1 < flags.length && !flags[i + 1].startsWith('--')) ? flags[i + 1] : '';
      c.appendChild(makeFlagRow(fname, fval));
    }
    sel.value = '';
  });
}

async function savePreset() {
  var name = prompt('Preset name:');
  if (!name) return;
  var flags = [];
  document.querySelectorAll('#flagsContainer .flag-row').forEach(function(row) {
    var sel = row.querySelector('.flag-name'), valInput = row.querySelector('.flag-value'), customInput = row.querySelector('.flag-custom');
    var fname = sel.value || customInput.value.trim(), fval = valInput.value.trim();
    if (!fname) return;
    flags.push(fname);
    if (fval) flags.push(fval);
  });
  if (!flags.length) { alert('No flags to save'); return; }
  try {
    await fetch('/api/v1/presets', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name: name, flags: flags }) });
    loadPresets();
  } catch (e) { alert('Error: ' + e); }
}

async function deletePreset() {
  var sel = document.getElementById('presetSelect'), name = sel.value;
  if (!name) { alert('Select a preset first'); return; }
  if (!confirm('Delete preset "' + name + '"?')) return;
  try {
    await fetch('/api/v1/presets?name=' + encodeURIComponent(name), { method: 'DELETE' });
    loadPresets();
    sel.value = '';
  } catch (e) { alert('Error: ' + e); }
}

// ── Pull Model (streaming) ────────────────────────────
function pullModel() {
  var ref = document.getElementById('pullInput').value.trim();
  if (!ref) { alert('Enter a model reference'); return; }
  startPull(ref, document.getElementById('pullBtn'));
}

async function startPull(ref, btn) {
  document.getElementById('pullSuggestions').style.display = 'none';
  btn.disabled = true;
  document.getElementById('pullProgress').style.display = 'block';
  document.getElementById('pullBar').style.width = '0%';
  document.getElementById('pullPct').textContent = '0%';
  document.getElementById('pullSpeed').textContent = '';
  document.getElementById('pullStatus').textContent = 'Connecting…';

  try {
    var r = await fetch('/api/v1/models/pull/stream?model=' + encodeURIComponent(ref));
    if (!r.ok) {
      document.getElementById('pullStatus').textContent = 'HTTP ' + r.status;
      btn.disabled = false; btn.textContent = 'Pull';
      return;
    }
    var reader = r.body.getReader(), decoder = new TextDecoder(), buf = '', done = false;

    while (true) {
      var { done: streamDone, value } = await reader.read();
      if (streamDone) break;
      buf += decoder.decode(value, { stream: true });
      var events = buf.split('\n\n'); buf = events.pop() || '';
      for (var e of events) {
        var lines = e.split('\n');
        var dataLine = '';
        for (var l of lines) {
          if (l.startsWith('data: ')) dataLine += l.slice(6);
        }
        if (!dataLine) continue;
        try {
          var d = JSON.parse(dataLine);
          if (d.status === 'progress' || d.pct !== undefined) {
            document.getElementById('pullPct').textContent = (d.pct || 0).toFixed(1) + '%';
            document.getElementById('pullBar').style.width = (d.pct || 0) + '%';
            document.getElementById('pullSpeed').textContent = d.speed || '';
            document.getElementById('pullStatus').textContent = 'Downloading…';
          } else if (d.status === 'done') {
            done = true;
            document.getElementById('pullBar').style.width = '100%';
            document.getElementById('pullPct').textContent = '100%';
            document.getElementById('pullStatus').textContent = '\u2713 Done';
            loadModels();
            setTimeout(function() {
              document.getElementById('pullProgress').style.display = 'none';
              btn.disabled = false; btn.textContent = 'Pull';
            }, 2000);
          } else if (d.status === 'exists') {
            done = true;
            document.getElementById('pullStatus').textContent = 'Already exists';
            loadModels();
            setTimeout(function() {
              document.getElementById('pullProgress').style.display = 'none';
              btn.disabled = false; btn.textContent = 'Pull';
            }, 2000);
          } else if (d.status === 'error') {
            done = true;
            document.getElementById('pullStatus').textContent = 'Error: ' + d.error;
            btn.disabled = false; btn.textContent = 'Pull';
          }
        } catch (e) {}
      }
    }
  } catch (e) {
    document.getElementById('pullStatus').textContent = 'Error: ' + e.message;
    btn.disabled = false; btn.textContent = 'Retry';
  }
}

// ── Search-as-you-type ────────────────────────────────
var searchTimeout = null;

function onPullInputChange(val) {
  var sg = document.getElementById('pullSuggestions');
  if (searchTimeout) clearTimeout(searchTimeout);
  if (val.length < 2) { sg.style.display = 'none'; return; }
  searchTimeout = setTimeout(function() { doSearch(val); }, 300);
}

async function doSearch(q) {
  var sg = document.getElementById('pullSuggestions');
  try {
    var r = await fetch('/api/v1/models/search?q=' + encodeURIComponent(q));
    if (!r.ok) { sg.style.display = 'none'; return; }
    var results = await r.json();
    if (!results || !results.length) { sg.style.display = 'none'; return; }
    sg.innerHTML = results.slice(0, 8).map(function(m) {
      var id = m.id, label = id.replace(/-GGUF$/i, '');
      var size = m.size ? fmtSize(m.size) : '';
      var likes = m.likes > 0 ? '♥' + m.likes : '';
      var downloads = m.downloads > 0 ? (m.downloads > 999 ? (m.downloads/1000).toFixed(0) + 'K' : m.downloads) + ' dl' : '';
      var tag = m.pipeline_tag ? '<span class="badge badge-amber">' + escHtml(m.pipeline_tag) + '</span>' : '';
      var meta = [tag, size, likes, downloads].filter(Boolean).join(' · ');
      return '<div class="suggestion"><div><div class="name">' + escHtml(label) + '</div>' + (meta ? '<div class="meta">' + meta + '</div>' : '') + '</div><button class="small primary pull-btn" onclick="pullSuggestion(\'' + escAttr(id) + '\', this)">Pull</button></div>';
    }).join('');
    sg.style.display = 'block';
  } catch (e) { sg.style.display = 'none'; }
}

function pullSuggestion(id, btn) {
  document.getElementById('pullSuggestions').style.display = 'none';
  startPull(id, document.getElementById('pullBtn'));
}

// ── Default Flags ─────────────────────────────────────
async function loadDefaultFlags() {
  try {
    var r = await fetch('/api/v1/config/default-flags'), flags = await r.json();
    if (!flags || !flags.length) return;
    var c = document.getElementById('flagsContainer'); c.innerHTML = '';
    for (var i = 0; i < flags.length; i += 2) {
      var name = flags[i];
      var value = (i + 1 < flags.length && !flags[i + 1].startsWith('--')) ? flags[i + 1] : '';
      c.appendChild(makeFlagRow(name, value));
    }
  } catch (e) {}
}

// ── Chat ──────────────────────────────────────────────
function selectChatInstance() {
  var s = document.getElementById('chatInstanceSelect'); chatPort = parseInt(s.value) || 0;
  if (chatPort) { chatHistory = []; document.getElementById('chatPanel').innerHTML = ''; document.getElementById('chatPanel').style.display = 'block'; document.getElementById('chatEmpty').style.display = 'none'; addSystemMsg('Connected'); }
}
function selectChatFor(port, model) {
  chatPort = port; chatHistory = []; chatSessionId = null;
  document.getElementById('chatInstanceSelect').value = port;
  document.getElementById('chatPanel').innerHTML = '';
  document.getElementById('chatPanel').style.display = 'block';
  document.getElementById('chatEmpty').style.display = 'none';
  addSystemMsg('Chatting with ' + (model || 'port ' + port));
  if (currentView != 'chat') switchView('chat');
  setTimeout(function() { document.getElementById('chatInput').focus(); }, 100);
}
function addSystemMsg(t) { var c = document.getElementById('chatPanel'); c.innerHTML += '<div class="msg system">' + escHtml(t) + '</div>'; c.scrollTop = c.scrollHeight; }
function addMsg(r, t, re) {
  var c = document.getElementById('chatPanel');
  var el = document.createElement('div'); el.className = 'msg ' + r;
  el.textContent = t;
  if (r === 'assistant' && t) {
    var copyBtn = document.createElement('button');
    copyBtn.className = 'copy-btn'; copyBtn.textContent = '📋';
    copyBtn.title = 'Copy message';
    copyBtn.onclick = function() { navigator.clipboard.writeText(t).then(function() { copyBtn.textContent = '✓'; setTimeout(function() { copyBtn.textContent = '📋'; }, 1500); }); };
    el.appendChild(copyBtn);
  }
  if (re) { el.insertAdjacentHTML('beforebegin', '<div class="reasoning">' + escHtml(re) + '</div>'); }
  c.appendChild(el); c.scrollTop = c.scrollHeight; return el;
}
function autoGrow(el) { el.style.height = 'auto'; el.style.height = Math.min(el.scrollHeight, 200) + 'px'; }
function clearChat() { chatHistory = []; chatSessionId = null; var p = document.getElementById('chatPanel'); p.innerHTML = ''; addSystemMsg('Chat cleared'); }
function escHtml(s) { return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;'); }
function escAttr(s) { return s.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;'); }

async function sendChat() {
  var input = document.getElementById('chatInput'), msg = input.value.trim();
  if (!msg || !chatPort) return;
  input.value = ''; addMsg('user', msg); chatHistory.push({ role: 'user', content: msg });
  var content = '', reasoning = '', msgEl = null, reasoningEl = null, chatPanel = document.getElementById('chatPanel');
  try {
    var r = await fetch('/api/v1/chat?port=' + chatPort, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ model: 'default', messages: chatHistory.slice(-20), max_tokens: 4096, stream: true }) });
    var reader = r.body.getReader(), decoder = new TextDecoder(), buf = '';
    while (true) {
      var { done, value } = await reader.read();
      if (done) break;
      buf += decoder.decode(value, { stream: true });
      var lines = buf.split('\n'); buf = lines.pop() || '';
      for (var line of lines) {
        if (!line.startsWith('data: ')) continue;
        var data = line.slice(6);
        if (data === '[DONE]') continue;
        try {
          var chunk = JSON.parse(data), delta = chunk.choices && chunk.choices[0] && chunk.choices[0].delta || {};
          if (delta.reasoning_content) {
            reasoning += delta.reasoning_content;
            if (!reasoningEl) { reasoningEl = document.createElement('div'); reasoningEl.className = 'reasoning'; chatPanel.appendChild(reasoningEl); }
            reasoningEl.textContent = reasoning;
          }
          if (delta.content) {
            if (!msgEl) { msgEl = document.createElement('div'); msgEl.className = 'msg assistant'; chatPanel.appendChild(msgEl); }
            content += delta.content; msgEl.textContent = content;
          }
        } catch (e) {}
      }
      chatPanel.scrollTop = chatPanel.scrollHeight;
    }
    if (msgEl) { chatHistory.push({ role: 'assistant', content: content }); saveChatHistory(); }
  } catch (e) { if (msgEl) { msgEl.innerHTML = 'Error: ' + escHtml(e.message); msgEl.className = 'msg system'; } else { var em = addMsg('assistant', ''); em.innerHTML = 'Error: ' + escHtml(e.message); em.className = 'msg system'; } }
}

// ── Chat History ────────────────────────────────────────
async function saveChatHistory() {
  if (!chatPort || !chatHistory.length) return;
  var model = '';
  var list = JSON.parse(document.querySelector('#instances').getAttribute('data-list') || '[]');
  for (var i = 0; i < list.length; i++) { if (list[i].port == chatPort) { model = list[i].model || ''; break; } }
  try {
    var r = await fetch('/api/v1/chats/' + (chatSessionId || ''), { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: chatSessionId, model: model, messages: chatHistory }) });
    if (r.ok) {
      var d = await r.json();
      if (d.id) chatSessionId = d.id;
    }
  } catch (e) {}
}

async function showChatHistory() {
  document.getElementById('chatHistoryModal').style.display = 'block';
  var list = document.getElementById('chatHistoryList');
  try {
    var r = await fetch('/api/v1/chats'), chats = await r.json();
    if (!chats || !chats.length) { list.innerHTML = '<div class="empty-state"><div class="icon">💬</div><div class="title">No saved chats</div></div>'; return; }
    list.innerHTML = chats.map(function(c) {
      return '<div class="model-row" onclick="loadChat(\'' + c.id + '\')"><div><div class="name">' + escHtml(c.preview || '(empty)') + '</div><div class="info">' + c.msg_count + ' messages · ' + escHtml(c.model) + ' · ' + new Date(c.updated_at).toLocaleDateString() + '</div></div><button class="small danger" onclick="event.stopPropagation();deleteChat(\'' + c.id + '\')" aria-label="Delete chat">🗑</button></div>';
    }).join('');
  } catch (e) { list.innerHTML = '<div class="empty-state"><div class="title">Error loading chats</div></div>'; }
}

function closeChatHistory() { document.getElementById('chatHistoryModal').style.display = 'none'; }
document.getElementById('chatHistoryModal').addEventListener('click', closeChatHistory);

async function loadChat(id) {
  closeChatHistory();
  try {
    var r = await fetch('/api/v1/chats/' + id), session = await r.json();
    if (!session || !session.messages) return;
    chatHistory = session.messages;
    chatSessionId = session.id;
    document.getElementById('chatPanel').innerHTML = '';
    document.getElementById('chatPanel').style.display = 'block';
    document.getElementById('chatEmpty').style.display = 'none';
    var firstUser = '';
    for (var i = 0; i < session.messages.length; i++) {
      var m = session.messages[i];
      if (m.role === 'user') { firstUser = m.content; break; }
    }
    addSystemMsg('Loaded chat' + (firstUser ? ': ' + (firstUser.length > 50 ? firstUser.slice(0, 50) + '…' : firstUser) : ''));
    session.messages.forEach(function(m) { addMsg(m.role, m.content); });
    if (currentView != 'chat') switchView('chat');
  } catch (e) {}
}

async function deleteChat(id) {
  if (!confirm('Delete this chat?')) return;
  try {
    await fetch('/api/v1/chats/' + id, { method: 'DELETE' });
    showChatHistory();
  } catch (e) { alert('Error: ' + e); }
}

// ── Logs ──────────────────────────────────────────────
var logPollInterval = null;

async function viewLogs(port) {
  document.getElementById('logModal').style.display = 'block';
  if (logPollInterval) clearInterval(logPollInterval);
  var content = document.getElementById('logContent');
    var header = document.querySelector('#logModal h2');
    if (header) header.textContent = '\uD83D\uDCCB Logs (port ' + port + ')';
    async function refresh() {
    var r = await fetch('/api/v1/instances/logs?port=' + port), d = await r.json();
    if (d.error) { content.textContent = 'No logs'; if (logPollInterval) clearInterval(logPollInterval); }
    else { content.textContent = d.lines && d.lines.length ? d.lines.slice(-50).join('\n') : '(empty)'; content.scrollTop = content.scrollHeight; }
  }
  refresh();
  logPollInterval = setInterval(refresh, 3000);
}
function closeLogs() { document.getElementById('logModal').style.display = 'none'; if (logPollInterval) { clearInterval(logPollInterval); logPollInterval = null; } }
document.getElementById('logModal').addEventListener('click', closeLogs);

// ── Helpers ───────────────────────────────────────────
function formatFlags(flags) {
  var out = [], skip = 0;
  for (var j = 0; j < flags.length; j++) {
    if (skip > 0) { skip--; continue; }
    if (flags[j] === '-m') { skip = 1; continue; }
    out.push(flags[j]);
  }
  return out.join(' ');
}
function fmtSize(b) {
  if (!b) return '?';
  if (b > 1073741824) return (b / 1073741824).toFixed(1) + ' GB';
  if (b > 1048576) return (b / 1048576).toFixed(0) + ' MB';
  return (b / 1048576).toFixed(2) + ' MB';
}

// ── Sidebar ──────────────────────────────────────────
function toggleSidebar() {
  document.querySelector('.sidebar').classList.toggle('collapsed');
  var btn = document.querySelector('.sidebar .toggle');
  btn.textContent = document.querySelector('.sidebar').classList.contains('collapsed') ? '▶' : '◀';
  localStorage.setItem('gollama-sidebar', document.querySelector('.sidebar').classList.contains('collapsed') ? 'collapsed' : '');
}
(function() {
  if (localStorage.getItem('gollama-sidebar') === 'collapsed') {
    document.querySelector('.sidebar').classList.add('collapsed');
    document.querySelector('.sidebar .toggle').textContent = '▶';
  }
})();

// ── Theme ─────────────────────────────────────────────
function toggleTheme() {
  var b = document.body, h = document.documentElement;
  b.classList.toggle('light'); h.classList.toggle('light');
  document.getElementById('themeToggle').innerHTML = b.classList.contains('light') ? '<span>☀️</span><span class="label">Theme</span>' : '<span>🌙</span><span class="label">Theme</span>';
  localStorage.setItem('gollama-theme', b.classList.contains('light') ? 'light' : 'dark');
}
(function() {
  if (localStorage.getItem('gollama-theme') === 'light') { document.body.classList.add('light'); document.documentElement.classList.add('light'); document.getElementById('themeToggle').innerHTML = '<span>☀️</span><span class="label">Theme</span>'; }
})();

// ── Settings ────────────────────────────────────────────
async function loadIdleTTL() {
  try {
    var r = await fetch('/api/v1/config'), cfg = await r.json();
    document.getElementById('idleTtlInput').value = cfg.idle_ttl || 0;
  } catch (e) {}
}

async function saveIdleTTL() {
  var val = parseInt(document.getElementById('idleTtlInput').value) || 0;
  var st = document.getElementById('idleTtlStatus');
  try {
    var r = await fetch('/api/v1/config', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ idle_ttl: val }) });
    if (r.ok) { st.textContent = 'Saved'; setTimeout(function() { st.textContent = ''; }, 2000); }
    else { st.textContent = 'Error saving'; }
  } catch (e) { st.textContent = 'Error: ' + e.message; }
}

// ── Init (staggered, no pile-up) ─────────────────────
loadModels();
setTimeout(loadDefaultFlags, 50);
setTimeout(loadPresets, 50);
setTimeout(loadInstances, 100);
setTimeout(loadIdleTTL, 150);
</script>
</body>
</html>`
