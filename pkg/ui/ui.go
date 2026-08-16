package ui

import _ "embed"

//go:embed logo.svg
var LogoSVG string

const Page = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta name="theme-color" content="#0f1115">
<link rel="icon" type="image/svg+xml" href="/logo.svg">
<title>gollama · GPU console</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Archivo:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
<style>
:root {
  --bg: #0f1115;
  --surface: #161a21;
  --surface-2: #1c212b;
  --surface-3: #252c38;
  --border: #29303c;
  --border-hover: #3b4453;
  --text: #e8ebf0;
  --text-muted: #9aa4b2;
  --text-dim: #626c7b;
  --accent: #45d483;
  --accent-hover: #5fe094;
  --accent-bg: rgba(69, 212, 131, 0.08);
  --accent-glow: rgba(69, 212, 131, 0.25);
  --accent-gradient: linear-gradient(135deg, #45d483, #2ea8ff);
  --green: #45d483;
  --green-bg: rgba(69, 212, 131, 0.1);
  --red: #ff5c5c;
  --red-bg: rgba(255, 92, 92, 0.08);
  --amber: #e3b341;
  --amber-bg: rgba(227, 179, 65, 0.1);
  --blue: #58a6ff;
  --blue-bg: rgba(88, 166, 255, 0.1);
  --purple: #bc8cff;
  --purple-bg: rgba(188, 140, 255, 0.1);
  --sidebar-w: 220px;
  --radius: 4px;
  --radius-sm: 3px;
  --font: 'Archivo', system-ui, 'Segoe UI', sans-serif;
  --font-mono: 'JetBrains Mono', 'SF Mono', 'Cascadia Mono', Consolas, monospace;
  --grid-line: rgba(255, 255, 255, 0.022);
  --shadow: 0 1px 2px rgba(0, 0, 0, 0.3);
  --shadow-lg: 0 12px 32px rgba(0, 0, 0, 0.5);
  --transition: 150ms ease;
}
.light {
  --bg: #f3f4f6;
  --surface: #ffffff;
  --surface-2: #f6f7f9;
  --surface-3: #e9ecf0;
  --border: #d8dce3;
  --border-hover: #b6bec9;
  --text: #1a1f27;
  --text-muted: #55606e;
  --text-dim: #8a93a1;
  --accent: #159048;
  --accent-hover: #18a854;
  --accent-bg: rgba(21, 144, 72, 0.07);
  --accent-glow: rgba(21, 144, 72, 0.18);
  --accent-gradient: linear-gradient(135deg, #159048, #0f6fb0);
  --green: #159048;
  --green-bg: rgba(21, 144, 72, 0.08);
  --red: #d13c3c;
  --red-bg: rgba(209, 60, 60, 0.07);
  --amber: #a06d10;
  --amber-bg: rgba(160, 109, 16, 0.09);
  --blue: #2f6fd0;
  --blue-bg: rgba(47, 111, 208, 0.08);
  --purple: #7040c8;
  --purple-bg: rgba(112, 64, 200, 0.08);
  --grid-line: rgba(15, 17, 21, 0.035);
  --shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
  --shadow-lg: 0 12px 28px rgba(0, 0, 0, 0.12);
}
* { margin: 0; padding: 0; box-sizing: border-box; }
::selection { background: var(--accent); color: #0b0f0d; }
.light ::selection { color: #fff; }
:focus-visible { outline: 2px solid var(--accent); outline-offset: 1px; }
html { color-scheme: dark; height: 100%; }
.light { color-scheme: light; }
::-webkit-scrollbar { width: 10px; height: 10px; }
::-webkit-scrollbar-track { background: transparent; }
::-webkit-scrollbar-thumb { background: var(--surface-3); border: 2px solid var(--bg); border-radius: 5px; }
::-webkit-scrollbar-thumb:hover { background: var(--border-hover); }
body {
  font-family: var(--font); background: var(--bg); color: var(--text);
  display: flex; font-size: 14px; line-height: 1.55; height: 100vh; overflow: hidden;
  -webkit-font-smoothing: antialiased;
}

/* ── Sidebar ─────────────────────────────────────────── */
.sidebar {
  width: var(--sidebar-w); background: var(--surface);
  border-right: 1px solid var(--border);
  display: flex; flex-direction: column; flex-shrink: 0;
  height: 100%; position: relative; z-index: 10;
  transition: width var(--transition); overflow: hidden;
}
.sidebar.collapsed { width: 60px; }
.sidebar .logo {
  padding: 18px 16px 12px; border-bottom: 1px solid var(--border);
  display: flex; flex-direction: column; align-items: center;
  gap: 5px; overflow: hidden;
}
.sidebar .logo img { width: 100%; max-width: 64px; height: auto; filter: brightness(0) invert(1); }
.light .sidebar .logo img { filter: none; }
.sidebar.collapsed .logo img { max-width: 30px; }
.sidebar .logo-sub {
  font-family: var(--font-mono); font-size: 8.5px; font-weight: 600;
  letter-spacing: 0.22em; color: var(--text-dim); white-space: nowrap;
  transition: opacity var(--transition);
}
.sidebar.collapsed .logo-sub { opacity: 0; height: 0; overflow: hidden; }
.sidebar .toggle {
  background: none; border: none; color: var(--text-dim);
  font-size: 10px; cursor: pointer; padding: 2px 8px;
  font-family: var(--font-mono); text-transform: none; letter-spacing: 0;
}
.sidebar .toggle:hover { color: var(--text); }
.sidebar nav { flex: 1; padding: 10px 8px; display: flex; flex-direction: column; gap: 2px; overflow-y: auto; }
.sidebar .nav-item {
  display: flex; align-items: center; gap: 10px; padding: 9px 12px;
  border-radius: var(--radius-sm); color: var(--text-muted);
  text-decoration: none; font-size: 12px; font-weight: 600; cursor: pointer;
  transition: background var(--transition), color var(--transition);
  white-space: nowrap; overflow: hidden;
  border: 1px solid transparent; background: none; width: 100%; text-align: left;
  font-family: var(--font); position: relative;
}
.sidebar .nav-item .icon, .sidebar .nav-item > span:not(.label) {
  font-size: 15px; width: 20px; text-align: center; flex-shrink: 0;
  filter: grayscale(1); opacity: 0.65;
  transition: filter var(--transition), opacity var(--transition);
}
.sidebar .nav-item .label {
  font-family: var(--font-mono); font-size: 11px; font-weight: 600;
  text-transform: uppercase; letter-spacing: 0.1em;
  transition: opacity var(--transition); white-space: nowrap;
}
.sidebar .nav-item:hover { background: var(--surface-2); color: var(--text); }
.sidebar .nav-item:hover .icon, .sidebar .nav-item:hover > span:not(.label) { filter: none; opacity: 1; }
.sidebar .nav-item.active { background: var(--accent-bg); color: var(--accent); box-shadow: inset 2px 0 0 var(--accent); }
.sidebar .nav-item.active .icon, .sidebar .nav-item.active > span:not(.label) { filter: none; opacity: 1; }
.sidebar.collapsed .nav-item { justify-content: center; padding: 9px 8px; }
.sidebar.collapsed .nav-item .label { opacity: 0; width: 0; overflow: hidden; }
.sidebar .bottom { padding: 8px; border-top: 1px solid var(--border); width: 100%; }
.sidebar .bottom .nav-item { width: 100%; }
.sidebar.collapsed .bottom .nav-item { justify-content: center; }

/* ── Content / faceplate ─────────────────────────────── */
.content { flex: 1; display: flex; flex-direction: column; min-width: 0; position: relative; }
.content::before {
  content: ''; position: absolute; inset: 0; z-index: 0; pointer-events: none;
  background-image:
    linear-gradient(var(--grid-line) 1px, transparent 1px),
    linear-gradient(90deg, var(--grid-line) 1px, transparent 1px);
  background-size: 28px 28px;
}
.faceplate {
  flex-shrink: 0; height: 46px; display: flex; align-items: center;
  justify-content: space-between; gap: 16px; padding: 0 20px;
  background: var(--surface); border-bottom: 1px solid var(--border);
  position: relative; z-index: 2;
}
.face-left { display: flex; align-items: center; gap: 10px; min-width: 0; }
.face-led { width: 8px; height: 8px; border-radius: 50%; background: var(--text-dim); flex-shrink: 0; }
.face-led.led-green { background: var(--green); box-shadow: 0 0 6px var(--accent-glow); }
.face-led.led-amber { background: var(--amber); box-shadow: 0 0 6px var(--amber); animation: pulse 1.4s ease-in-out infinite; }
.face-led.led-off { background: var(--text-dim); }
.face-brand { font-family: var(--font-mono); font-weight: 700; font-size: 13px; letter-spacing: 0.18em; }
.face-ver {
  font-family: var(--font-mono); font-size: 10px; color: var(--text-dim);
  border: 1px solid var(--border); padding: 1px 6px; border-radius: 2px;
  font-variant-numeric: tabular-nums;
}
.face-meters { display: flex; gap: 8px; }
.face-meter {
  font-family: var(--font-mono); font-size: 9.5px; letter-spacing: 0.08em;
  color: var(--text-dim); border: 1px solid var(--border); border-radius: 2px;
  padding: 3px 8px; display: inline-flex; gap: 6px; align-items: baseline; background: var(--bg);
}
.face-meter b { font-size: 11.5px; font-weight: 600; color: var(--text); font-variant-numeric: tabular-nums; }
.face-clock { font-family: var(--font-mono); font-size: 11px; color: var(--text-muted); font-variant-numeric: tabular-nums; }

/* ── Main / views ────────────────────────────────────── */
.main { flex: 1; overflow-y: auto; padding: 24px 28px 28px; position: relative; z-index: 1; min-height: 0; }
.view { display: none; height: 100%; }
.view.active { display: block; animation: fadeIn 240ms cubic-bezier(0.16, 1, 0.3, 1); }

@keyframes fadeIn { from { opacity: 0; transform: translateY(6px); } to { opacity: 1; transform: translateY(0); } }
@keyframes slideUp { from { opacity: 0; transform: translateY(10px); } to { opacity: 1; transform: translateY(0); } }
@keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.35; } }
@keyframes spin { to { transform: rotate(360deg); } }

@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after { animation-duration: 0.01ms !important; transition-duration: 0.01ms !important; }
  .view.active { animation: none !important; }
}

/* ── Page header ─────────────────────────────────────── */
.page-header { margin-bottom: 24px; }
.page-header h1 { font-size: 22px; font-weight: 700; letter-spacing: -0.01em; text-wrap: balance; }
.page-header p { font-family: var(--font-mono); font-size: 10.5px; color: var(--text-dim); margin-top: 5px; letter-spacing: 0.06em; }

/* ── Section ─────────────────────────────────────────── */
.section { margin-bottom: 28px; }
.section-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; gap: 12px; }
.section-header h2 {
  font-family: var(--font-mono); font-size: 10.5px; font-weight: 600; color: var(--text-muted);
  text-transform: uppercase; letter-spacing: 0.14em; display: flex; align-items: center; gap: 8px;
}
.section-header h2::before { content: ''; width: 5px; height: 5px; background: var(--accent); flex-shrink: 0; }
.section-tools { display: flex; align-items: center; gap: 8px; }

/* ── Cards ───────────────────────────────────────────── */
.card { background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius); box-shadow: var(--shadow); }
.card-body { padding: 16px; }

/* ── Instance card (rack unit) ───────────────────────── */
.instance-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(360px, 1fr)); gap: 12px; }
.inst-card {
  background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius);
  padding: 14px 16px; position: relative; transition: border-color var(--transition);
}
.inst-card::before {
  content: ''; position: absolute; left: 0; top: 0; bottom: 0; width: 2px;
  background: var(--green); border-radius: var(--radius) 0 0 var(--radius);
}
.inst-card.starting::before { background: var(--amber); }
.inst-card.stopped::before { background: var(--text-dim); }
.inst-card.stopped { opacity: 0.6; }
.inst-card.error::before { background: var(--red); }
.inst-card:hover { border-color: var(--border-hover); }
.inst-card .title { font-family: var(--font-mono); font-size: 12.5px; font-weight: 600; word-break: break-all; padding-left: 10px; }
.inst-card .meta { font-family: var(--font-mono); font-size: 11px; color: var(--text-dim); margin-top: 8px; display: flex; gap: 12px; flex-wrap: wrap; align-items: center; padding-left: 10px; }
.inst-card .actions { margin-top: 12px; display: flex; gap: 6px; flex-wrap: wrap; padding-left: 10px; }

/* ── Quick launch ────────────────────────────────────── */
.launch-row { display: grid; grid-template-columns: 1fr 110px auto; gap: 10px; align-items: end; margin-bottom: 14px; }
.launch-row .field { display: flex; flex-direction: column; gap: 5px; }
.launch-row .field label { font-family: var(--font-mono); font-size: 10px; color: var(--text-dim); text-transform: uppercase; letter-spacing: 0.1em; font-weight: 600; }
.flags-block { margin-top: 8px; }
.preset-row { display: flex; gap: 6px; flex-wrap: wrap; align-items: center; margin-top: 12px; }
.preset-row select { width: auto; min-width: 140px; flex: 1; }

/* ── Forms ───────────────────────────────────────────── */
select, input[type=text], input[type=number] {
  width: 100%; padding: 8px 12px; background: var(--bg);
  border: 1px solid var(--border); border-radius: var(--radius-sm);
  color: var(--text); font-size: 13px; outline: none;
  transition: border-color var(--transition), box-shadow var(--transition);
  font-family: var(--font);
}
select, input[type=number] { font-family: var(--font-mono); font-size: 12.5px; }
select:focus, input:focus, textarea:focus { border-color: var(--accent); box-shadow: 0 0 0 2px var(--accent-bg); }
select {
  cursor: pointer; appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' fill='%23626c7b'%3E%3Cpath d='M2 4l4 4 4-4'/%3E%3C/svg%3E");
  background-repeat: no-repeat; background-position: right 10px center; padding-right: 32px;
}
.light select {
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' fill='%238a93a1'%3E%3Cpath d='M2 4l4 4 4-4'/%3E%3C/svg%3E");
}
input[type=checkbox] { accent-color: var(--accent); }

/* ── Buttons ─────────────────────────────────────────── */
button {
  padding: 8px 16px; border-radius: var(--radius-sm); border: 1px solid transparent;
  font-size: 11.5px; font-weight: 600; cursor: pointer;
  transition: background var(--transition), border-color var(--transition), color var(--transition), opacity var(--transition);
  display: inline-flex; align-items: center; justify-content: center; gap: 6px;
  white-space: nowrap; position: relative;
  font-family: var(--font-mono); letter-spacing: 0.05em; text-transform: uppercase;
  background: var(--surface-2); color: var(--text);
}
button.primary { background: var(--accent); border-color: var(--accent); color: #0b0f0d; }
button.primary:hover { background: var(--accent-hover); border-color: var(--accent-hover); }
.light button.primary, .light .image-gen-btn { color: #fff; }
button.secondary { background: var(--surface-2); border-color: var(--border); color: var(--text); }
button.secondary:hover { background: var(--surface-3); border-color: var(--border-hover); }
button.danger { background: var(--red); border-color: var(--red); color: #fff; }
button.danger:hover { opacity: 0.9; }
button.small { padding: 5px 10px; font-size: 10.5px; }
button.ghost { background: transparent; border-color: transparent; color: var(--text-muted); text-transform: none; letter-spacing: 0; font-family: var(--font); font-weight: 500; }
button.ghost:hover { background: var(--surface-2); color: var(--text); }
button:disabled { opacity: 0.4; cursor: default; pointer-events: none; }

/* ── Badge / tag ─────────────────────────────────────── */
.badge {
  display: inline-flex; align-items: center; gap: 5px;
  padding: 2px 8px 2px 7px; border-radius: 2px;
  font-size: 9.5px; font-weight: 600; letter-spacing: 0.06em; text-transform: uppercase;
  font-family: var(--font-mono); border: 1px solid currentColor;
}
.badge::before { content: ''; width: 4px; height: 4px; border-radius: 50%; background: currentColor; flex-shrink: 0; }
.badge-green { background: var(--green-bg); color: var(--green); }
.badge-red { background: var(--red-bg); color: var(--red); }
.badge-amber { background: var(--amber-bg); color: var(--amber); }
.badge-blue { background: var(--blue-bg); color: var(--blue); }
.badge-profile { background: var(--purple-bg); color: var(--purple); }
.tag {
  display: inline-block; padding: 2px 7px; border-radius: 2px;
  font-size: 10px; font-family: var(--font-mono);
  background: var(--surface-2); color: var(--text-muted); border: 1px solid var(--border);
}

/* ── Advanced flags (details) ────────────────────────── */
.advanced-flags summary {
  cursor: pointer; font-family: var(--font-mono); font-size: 10.5px;
  color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.1em;
  user-select: none; padding: 4px 0; list-style: none;
}
.advanced-flags summary::-webkit-details-marker { display: none; }
.advanced-flags summary::before { content: '▸ '; }
.advanced-flags[open] summary::before { content: '▾ '; }
.advanced-flags summary:hover { color: var(--text); }

/* ── Flag rows ───────────────────────────────────────── */
.flag-row { display: flex; gap: 6px; margin-bottom: 6px; align-items: center; }
.flag-row .flag-search-wrapper { position: relative; flex-shrink: 0; min-width: 170px; }
.flag-row .flag-search {
  width: 100%; padding: 7px 10px; background: var(--bg);
  border: 1px solid var(--border); border-radius: var(--radius-sm);
  color: var(--text); font-size: 12px; outline: none;
  transition: border-color var(--transition); font-family: var(--font-mono); box-sizing: border-box;
}
.flag-row .flag-search:focus { border-color: var(--accent); }
.flag-row .flag-search.open { border-radius: var(--radius-sm) var(--radius-sm) 0 0; }
.flag-row .flag-dropdown {
  position: absolute; top: 100%; left: 0; right: 0; max-height: 240px; overflow-y: auto;
  background: var(--surface-2); border: 1px solid var(--border); border-top: none;
  border-radius: 0 0 var(--radius-sm) var(--radius-sm); z-index: 100; box-shadow: var(--shadow-lg);
}
.flag-row .flag-dropdown-item { padding: 6px 10px; font-size: 11.5px; cursor: pointer; color: var(--text); font-family: var(--font-mono); transition: background var(--transition); }
.flag-row .flag-dropdown-item:hover, .flag-row .flag-dropdown-item.sel { background: var(--accent-bg); color: var(--accent); }
.flag-row .flag-dropdown-divider { border-top: 1px solid var(--border); }
.flag-row .flag-dropdown-item-custom { color: var(--text-muted); font-family: var(--font); }
.flag-row .flag-dropdown-item-custom:hover { color: var(--text); }
.flag-row .flag-dropdown-empty { color: var(--text-dim); cursor: default; font-family: var(--font); }
.flag-row input.flag-custom, .flag-row input.flag-value { flex: 1; font-family: var(--font-mono); font-size: 12px; padding: 7px 10px; }

/* ── Settings edit mode ──────────────────────────────── */
.settings-card .settings-readonly { display: block; }
.settings-card .settings-form { display: none; }
.settings-card.editing .settings-readonly { display: none; }
.settings-card.editing .settings-form { display: block; }
.settings-card.editing .settings-edit-btn { display: none; }
.settings-card-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 10px; margin-bottom: 10px; }
.settings-card-title { font-size: 13px; font-weight: 700; }
.settings-card-sub { font-size: 11px; color: var(--text-dim); margin-top: 2px; line-height: 1.5; }
.settings-empty { font-size: 12px; color: var(--text-dim); }
.settings-form-actions { display: flex; align-items: center; gap: 8px; margin-top: 10px; }
.save-status { font-size: 11px; color: var(--text-muted); font-family: var(--font-mono); }

/* ── Detail rows ─────────────────────────────────────── */
.detail-row { display: flex; justify-content: space-between; align-items: center; gap: 12px; padding: 8px 0; border-bottom: 1px solid var(--border); }
.detail-row:last-child { border-bottom: none; }
.detail-label { font-family: var(--font-mono); font-size: 10.5px; color: var(--text-dim); text-transform: uppercase; letter-spacing: 0.1em; display: flex; align-items: center; gap: 6px; }
.detail-value { font-size: 13px; color: var(--text); text-align: right; }
code.path { font-size: 12px; color: var(--text-muted); font-family: var(--font-mono); }
#md-apiname { font-family: var(--font-mono); font-size: 12px; }
#md-path { word-break: break-all; font-family: var(--font-mono); font-size: 11px; }
#modelModal .modal-content { max-width: 500px; }
#chatHistoryModal .modal-content { max-width: 520px; }
#md-launch-btn { margin-top: 14px; width: 100%; }

/* ── Idle TTL row ────────────────────────────────────── */
.ttl-row { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; font-size: 13px; font-weight: 500; }
#idleTtlInput { width: 70px; text-align: center; }
.ttl-hint { font-family: var(--font-mono); font-size: 11px; color: var(--text-muted); }

/* ── Restart block ───────────────────────────────────── */
.restart-block { text-align: center; }
.restart-note { font-size: 11px; color: var(--text-dim); margin-top: 8px; font-family: var(--font-mono); }

/* ── Empty state ─────────────────────────────────────── */
.empty-state { text-align: center; padding: 40px 20px; color: var(--text-dim); }
.instance-grid .empty-state { grid-column: 1 / -1; display: flex; flex-direction: column; align-items: center; justify-content: center; min-height: 180px; }
.empty-state .icon { font-size: 28px; margin-bottom: 10px; opacity: 0.5; font-family: var(--font-mono); }
.empty-state .title { font-family: var(--font-mono); font-size: 11.5px; font-weight: 600; color: var(--text-muted); margin-bottom: 4px; text-transform: uppercase; letter-spacing: 0.08em; }
.empty-state p { font-size: 12.5px; }

/* ── Chat ────────────────────────────────────────────── */
.chat-container { display: flex; flex-direction: column; height: 100%; }
.chat-header, .image-header { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding-bottom: 12px; margin-bottom: 12px; border-bottom: 1px solid var(--border); flex-wrap: wrap; }
.chat-header .header-left, .image-header .header-left { display: flex; align-items: center; gap: 12px; flex: 1; flex-wrap: wrap; }
.chat-header h1, .image-header h1 { font-family: var(--font-mono); font-size: 14px; font-weight: 700; letter-spacing: 0.12em; text-transform: uppercase; }
.chat-header select { width: auto; min-width: 220px; }
.image-header select { width: auto; min-width: 200px; }
.chat-header .header-actions, .image-header .header-actions { display: flex; align-items: center; gap: 4px; flex-shrink: 0; }
.pill-btn {
  display: inline-flex; align-items: center; gap: 5px; padding: 4px 10px;
  font-family: var(--font-mono); font-size: 10px; font-weight: 600;
  letter-spacing: 0.06em; text-transform: uppercase;
  border: 1px solid transparent; cursor: pointer; transition: all var(--transition);
  white-space: nowrap; background: transparent; color: var(--text-muted); border-radius: 2px;
}
.pill-btn:hover { background: var(--surface-2); color: var(--text); border-color: var(--border); }
.pill-btn.history:hover { background: var(--accent-bg); color: var(--accent); border-color: transparent; }
.pill-btn.clear:hover { background: var(--red-bg); color: var(--red); border-color: transparent; }
.chat-msgs { flex: 1; overflow-y: auto; padding: 6px 2px; margin-bottom: 10px; min-height: 0; }
.chat-msgs .msg { margin-bottom: 10px; padding: 10px 14px; border-radius: var(--radius-sm); max-width: 82%; line-height: 1.65; font-size: 13.5px; animation: slideUp 200ms ease both; position: relative; }
.chat-msgs .user { background: var(--surface-2); border: 1px solid var(--border); border-left: 2px solid var(--accent); margin-left: auto; }
.chat-msgs .assistant { background: var(--surface); border: 1px solid var(--border); padding-right: 40px; }
.chat-msgs .system { background: transparent; border: none; color: var(--text-dim); font-family: var(--font-mono); font-size: 10.5px; text-align: center; max-width: 100%; letter-spacing: 0.04em; padding: 4px; }
.chat-msgs .assistant .copy-btn { position: absolute; top: 8px; right: 8px; font-size: 12px; background: none; border: none; cursor: pointer; opacity: 0; padding: 2px 4px; border-radius: 2px; transition: opacity var(--transition), background var(--transition); text-transform: none; }
.chat-msgs .assistant:hover .copy-btn { opacity: 0.5; }
.chat-msgs .assistant .copy-btn:hover { opacity: 1; background: var(--surface-2); }
.chat-input-row { display: flex; flex-direction: column; gap: 8px; background: var(--surface); border-radius: var(--radius-sm); padding: 12px; border: 1px solid var(--border); flex-shrink: 0; transition: border-color var(--transition), box-shadow var(--transition); }
.chat-input-row:focus-within { border-color: var(--accent); box-shadow: 0 0 0 2px var(--accent-bg); }
.chat-input-row .input-wrap { display: flex; gap: 8px; align-items: flex-end; }
.chat-input-row .input-wrap textarea { flex: 1; max-height: 200px; resize: none; padding: 9px 12px; font-family: var(--font); font-size: 13px; line-height: 1.5; background: var(--bg); border: 1px solid var(--border); border-radius: var(--radius-sm); color: var(--text); outline: none; }
.chat-input-row .input-wrap textarea:focus { border-color: var(--accent); }
#chatEmpty { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; }
#contextMeter { height: 3px; background: var(--surface-3); border-radius: 2px; overflow: hidden; }
#contextBar { height: 100%; width: 0%; background: var(--accent); border-radius: 2px; transition: width 300ms ease; }
#ctxLabelWrap { display: none; justify-content: center; }
.ctx-label { padding: 2px 8px; border-radius: 2px; font-size: 10px; font-weight: 600; background: var(--accent-bg); color: var(--accent); font-family: var(--font-mono); border: 1px solid var(--accent); }
.reasoning { color: var(--text-dim); font-family: var(--font-mono); font-size: 11.5px; border-left: 2px solid var(--amber); padding-left: 10px; margin-bottom: 8px; opacity: 0.9; line-height: 1.5; white-space: pre-wrap; }
.chat-loading { animation: pulse 1.2s infinite; display: inline-block; letter-spacing: 4px; font-size: 18px; line-height: 1; color: var(--text-dim); }

/* ── Image playground ────────────────────────────────── */
.image-container { display: flex; flex-direction: column; height: 100%; }
.image-body { flex: 1; display: flex; gap: 16px; min-height: 0; }
.image-prompt-area { flex: 0 0 330px; display: flex; flex-direction: column; gap: 12px; overflow-y: auto; min-height: 0; }
.image-results { flex: 1; min-height: 0; display: flex; flex-direction: column; gap: 12px; overflow-y: auto; }
.image-input-card { background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius); padding: 14px; }
.image-input-card label { display: block; font-family: var(--font-mono); font-size: 10px; font-weight: 600; color: var(--text-dim); margin-bottom: 6px; text-transform: uppercase; letter-spacing: 0.1em; }
.image-input-card textarea { width: 100%; min-height: 110px; max-height: 240px; resize: vertical; background: var(--bg); border: 1px solid var(--border); border-radius: var(--radius-sm); color: var(--text); font-family: var(--font); font-size: 13px; line-height: 1.6; padding: 10px 12px; outline: none; transition: border-color var(--transition); }
.image-input-card textarea:focus { border-color: var(--accent); }
.img-model-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; }
.img-model-head label { margin-bottom: 0; }
#imgModelSearchWrap { margin-top: 8px; }
#imgModelSearchResults { border: 1px solid var(--border); border-radius: var(--radius-sm); overflow: hidden; background: var(--surface-2); }
#imgModelList { margin-top: 6px; }
.image-gen-btn {
  width: 100%; padding: 10px 16px; font-size: 12px; font-weight: 700;
  border-radius: var(--radius-sm); border: none; cursor: pointer;
  background: var(--accent); color: #0b0f0d;
  font-family: var(--font-mono); text-transform: uppercase; letter-spacing: 0.08em;
  transition: background var(--transition);
}
.image-gen-btn:hover { background: var(--accent-hover); }
.image-gen-btn:active { transform: translateY(1px); }
.image-gen-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.img-adv { display: flex; flex-direction: column; gap: 10px; margin-top: 8px; }
.img-adv-item { display: flex; flex-direction: column; }
.img-adv-item label { font-family: var(--font-mono); font-size: 10px; color: var(--text-dim); display: block; margin-bottom: 4px; text-transform: uppercase; letter-spacing: 0.1em; }
.img-adv-item input, .img-adv-item select { width: 100%; }
#imageSizeCustom { margin-top: 4px; }
.img-history-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; font-family: var(--font-mono); font-size: 10px; text-transform: uppercase; letter-spacing: 0.1em; color: var(--text-dim); }
#imageHistoryCount { font-size: 10px; text-transform: none; letter-spacing: 0; }
.image-result-card { background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius); overflow: hidden; animation: slideUp 200ms ease both; }
.image-result-card .result-meta { padding: 10px 12px; border-bottom: 1px solid var(--border); display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.image-result-card .result-prompt { font-family: var(--font-mono); font-size: 11px; color: var(--text-muted); flex: 1; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.image-result-card .result-actions { display: flex; gap: 4px; flex-shrink: 0; }
.image-result-card .result-img { display: block; width: 100%; max-height: 68vh; object-fit: contain; cursor: zoom-in; transition: opacity var(--transition); background: #000; }
.image-result-card .result-img:hover { opacity: 0.92; }
.image-history-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(140px, 1fr)); gap: 8px; }
.image-thumb { border-radius: var(--radius-sm); overflow: hidden; border: 1px solid var(--border); cursor: pointer; transition: border-color var(--transition); position: relative; background: var(--surface); }
.image-thumb:hover { border-color: var(--accent); }
.image-thumb img { width: 100%; display: block; }
.image-thumb .thumb-overlay { position: absolute; inset: 0; background: linear-gradient(to top, rgba(0, 0, 0, 0.7), transparent); opacity: 0; transition: opacity var(--transition); display: flex; align-items: flex-end; padding: 8px; }
.image-thumb:hover .thumb-overlay { opacity: 1; }
.image-thumb .thumb-prompt { font-size: 11px; color: #fff; line-height: 1.4; overflow: hidden; display: -webkit-box; -webkit-line-clamp: 3; -webkit-box-orient: vertical; }
.image-empty { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 10px; color: var(--text-dim); text-align: center; padding: 24px; }
.image-empty .icon { font-size: 30px; opacity: 0.4; font-family: var(--font-mono); }
.image-empty .title { font-family: var(--font-mono); font-size: 11.5px; font-weight: 600; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.08em; }
.image-empty p { font-size: 12.5px; text-align: center; max-width: 300px; }
.image-spinner { display: inline-block; width: 14px; height: 14px; border: 2px solid var(--border); border-top-color: var(--accent); border-radius: 50%; animation: spin 600ms linear infinite; vertical-align: -2px; margin-right: 6px; }
.image-lightbox { position: fixed; inset: 0; z-index: 9999; background: rgba(5, 7, 10, 0.9); display: flex; align-items: center; justify-content: center; padding: 32px; cursor: zoom-out; }
.image-lightbox img { max-width: 100%; max-height: 100%; object-fit: contain; border: 1px solid var(--border); }
.image-lightbox .lb-close { position: absolute; top: 14px; right: 14px; background: var(--surface); border: 1px solid var(--border); border-radius: 2px; width: 34px; height: 34px; cursor: pointer; font-size: 14px; display: flex; align-items: center; justify-content: center; color: var(--text); text-transform: none; }
.image-lightbox .lb-close:hover { background: var(--surface-2); }
.image-loading-card { background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius); padding: 36px; text-align: center; color: var(--text-dim); font-family: var(--font-mono); font-size: 12px; }
.image-loading-card .spinner { display: block; width: 32px; height: 32px; border-width: 3px; margin: 0 auto 14px; }

/* ── Modal ───────────────────────────────────────────── */
.modal { display: none; position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(8, 10, 13, 0.7); z-index: 100; animation: fadeIn 120ms ease; }
.modal-content { background: var(--surface); margin: 8% auto; padding: 20px; width: 88%; max-width: 680px; max-height: 76vh; border-radius: var(--radius); overflow: auto; border: 1px solid var(--border); box-shadow: var(--shadow-lg); overscroll-behavior: contain; }
.modal-content h2 { font-family: var(--font-mono); font-size: 12px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.08em; }
.modal-head { display: flex; justify-content: space-between; align-items: center; gap: 10px; margin-bottom: 12px; }
.modal-head-actions { display: flex; gap: 6px; }
.modal-content pre { background: var(--bg); border: 1px solid var(--border); padding: 12px; border-radius: var(--radius-sm); margin-top: 12px; font-size: 11px; line-height: 1.55; overflow: auto; max-height: 55vh; white-space: pre-wrap; color: var(--text-muted); font-family: var(--font-mono); }

/* ── Error line ──────────────────────────────────────── */
.error-line { font-size: 10.5px; color: var(--red); margin-top: 8px; padding: 6px 10px; background: var(--red-bg); border-radius: 2px; word-break: break-all; font-family: var(--font-mono); border-left: 2px solid var(--red); }

/* ── Chat history ────────────────────────────────────── */
.chat-history-list { max-height: 50vh; overflow-y: auto; }
.chat-history-item { display: flex; justify-content: space-between; align-items: center; gap: 8px; padding: 10px 8px; transition: background var(--transition); cursor: pointer; }
.chat-history-item:hover { background: var(--surface-2); }
.chat-history-item + .chat-history-item { border-top: 1px solid var(--border); }
.chat-history-main { flex: 1; min-width: 0; }
.chat-history-title { font-size: 12.5px; font-weight: 500; word-break: break-all; }
.chat-history-meta { font-family: var(--font-mono); font-size: 10px; color: var(--text-dim); margin-top: 2px; }
.chat-history-actions { display: flex; gap: 4px; flex-shrink: 0; margin-left: 8px; }

/* ── Model list ──────────────────────────────────────── */
.model-row { display: flex; justify-content: space-between; align-items: center; gap: 10px; padding: 10px 12px; transition: background var(--transition); cursor: pointer; border-bottom: 1px solid var(--border); }
.model-row:last-child { border-bottom: none; }
.model-row:hover { background: var(--surface-2); }
.model-row:hover .name { color: var(--accent); }
.model-row .info-icon { font-size: 11px; opacity: 0; transition: opacity var(--transition); margin-left: 4px; }
.model-row:hover .info-icon { opacity: 0.7; }
.model-row .name { font-family: var(--font-mono); font-size: 12.5px; font-weight: 500; word-break: break-all; }
.model-row .info { font-family: var(--font-mono); font-size: 10.5px; color: var(--text-dim); margin-top: 4px; display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }

/* ── Pull model ──────────────────────────────────────── */
.pull-row { display: flex; gap: 8px; margin-bottom: 12px; position: relative; }
.pull-row input { flex: 1; }
#pullSuggestions { border: 1px solid var(--border); border-radius: var(--radius-sm); overflow: hidden; background: var(--surface); box-shadow: var(--shadow-lg); }
#pullProgress { margin-top: 10px; }
#pullModelName { font-size: 12px; font-weight: 600; font-family: var(--font-mono); color: var(--text); margin-bottom: 6px; word-break: break-all; }
.pull-status-row { display: flex; justify-content: space-between; align-items: center; gap: 8px; font-family: var(--font-mono); font-size: 11px; color: var(--text-muted); margin-bottom: 4px; }
.pull-track { height: 4px; background: var(--surface-3); border: 1px solid var(--border); border-radius: 2px; overflow: hidden; margin-bottom: 6px; }
#pullBar { height: 100%; width: 0%; background: var(--accent); transition: width 200ms ease; }
.suggestion { display: flex; justify-content: space-between; align-items: center; gap: 10px; padding: 9px 12px; cursor: pointer; transition: background var(--transition); }
.suggestion:hover { background: var(--surface-2); }
.suggestion + .suggestion { border-top: 1px solid var(--border); }
.suggestion .name { font-family: var(--font-mono); font-size: 12px; font-weight: 500; word-break: break-all; }
.suggestion .meta { font-family: var(--font-mono); font-size: 10px; color: var(--text-dim); margin-top: 2px; }
.suggestion .badge { font-size: 9px; }
.suggestion .pull-btn { font-size: 10px; padding: 3px 10px; flex-shrink: 0; margin-left: 8px; }

/* ── Utilities ───────────────────────────────────────── */
.spinner { display: inline-block; width: 13px; height: 13px; border: 2px solid var(--border); border-top-color: var(--accent); border-radius: 50%; animation: spin 0.7s linear infinite; vertical-align: -2px; }
.refreshing { opacity: 0.45; pointer-events: none; transition: opacity var(--transition); }
.tps { font-variant-numeric: tabular-nums; }

/* ── Responsive ──────────────────────────────────────── */
@media (max-width: 768px) {
  .sidebar { width: 60px !important; }
  .sidebar .logo { padding: 12px 8px; }
  .sidebar .logo img { max-width: 28px; }
  .sidebar .toggle, .sidebar .nav-item .label { display: none; }
  .sidebar .nav-item { justify-content: center; padding: 9px 8px; }
  .faceplate { padding: 0 12px; }
  .face-ver, .face-clock { display: none; }
  .face-meters .face-meter:nth-child(3) { display: none; }
  .main { padding: 16px 12px 20px; }
  .launch-row { grid-template-columns: 1fr; }
  .instance-grid { grid-template-columns: 1fr; }
  .chat-msgs .msg { max-width: 92%; }
  .chat-header select, .image-header select { min-width: 160px; flex: 1; }
  .chat-container, .image-container { height: auto; min-height: 100%; }
  .chat-msgs { max-height: 50vh; }
  .image-body { flex-direction: column; overflow-y: auto; }
  .image-prompt-area { flex: none; width: 100%; overflow: visible; }
  .image-results { overflow: visible; }
}
</style>
</head>
<body>

<!-- ─── Sidebar ──────────────────────────────────────── -->
<aside class="sidebar" role="navigation" aria-label="Main navigation">
  <div class="logo">
    <img src="/logo.svg" alt="gollama">
    <span class="logo-sub">GPU CONSOLE</span>
    <button class="toggle" onclick="toggleSidebar()" aria-label="Toggle sidebar">◀</button>
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
    <button class="nav-item" onclick="switchView('image')" aria-label="Image">
      <span class="icon">🎨</span><span class="label">Image</span>
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

<div class="content">

<!-- ─── Faceplate ────────────────────────────────────── -->
<header class="faceplate">
  <div class="face-left">
    <span class="face-led led-off" id="faceLed"></span>
    <span class="face-brand">GOLLAMA</span>
    <span class="face-ver" id="faceVersion">v—</span>
  </div>
  <div class="face-meters" aria-hidden="true">
    <span class="face-meter">INST<b id="faceInst">0</b></span>
    <span class="face-meter">THR<b id="faceTps">—</b></span>
    <span class="face-meter">MOD<b id="faceModels">0</b></span>
  </div>
  <div class="face-right">
    <span class="face-clock" id="faceClock">--:--:--</span>
  </div>
</header>

<!-- ─── Main ──────────────────────────────────────────── -->
<main class="main" id="main">

<!-- ── Dashboard ────────────────────────────────────── -->
<div id="view-dashboard" class="view active" role="tabpanel" aria-label="Dashboard">
  <div class="page-header">
    <h1>Dashboard</h1>
    <p>MONITOR &amp; MANAGE LLAMA.CPP INSTANCES</p>
  </div>

  <div class="section" id="launch-section">
    <div class="section-header"><h2>Quick Launch</h2></div>
    <div class="card"><div class="card-body">
      <div class="launch-row">
        <div class="field"><label for="modelSelect">Model</label><select id="modelSelect" autocomplete="off"><option value="">Loading…</option></select></div>
        <div class="field"><label for="portInput">Port</label><input type="number" id="portInput" value="8081" min="8081" max="8099" autocomplete="off"></div>
        <div class="field"><button class="primary" onclick="launchInstance()" id="launchBtn">Launch</button></div>
      </div>
      <details class="advanced-flags">
        <summary>Advanced flags</summary>
        <div class="flags-block">
          <div id="flagsContainer"></div>
          <button class="ghost small" onclick="addFlag()">+ Add Flag</button>
        </div>
      </details>
      <div class="preset-row">
        <select id="presetSelect" onchange="applyPreset()">
          <option value="">Presets</option>
        </select>
        <button class="small secondary" onclick="savePreset()" id="savePresetBtn" title="Save current flags as preset">Save</button>
        <button class="small danger" onclick="deletePreset()" id="deletePresetBtn" style="display:none" title="Delete preset">Delete</button>
      </div>
    </div></div>
  </div>

  <div class="section">
    <div class="section-header">
      <h2>Running Instances</h2>
      <div class="section-tools">
        <span class="badge badge-blue" id="instanceCount"></span>
        <button class="ghost small" onclick="loadInstances()" title="Refresh instances" style="padding:2px 6px;font-size:13px">↻</button>
      </div>
    </div>
    <div id="instances" class="instance-grid">
      <div class="empty-state">
        <div class="icon">▣</div>
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
    <p><span id="modelCount"><span class="spinner" id="modelCountSpinner"></span> Loading…</span> <button class="ghost small" onclick="loadModels()" title="Refresh models" style="padding:2px 6px;vertical-align:middle">↻</button></p>
  </div>
  <div id="modelList" class="card"><div class="card-body"><div class="empty-state"><span class="spinner"></span> Loading models…</div></div></div>
  <div class="section">
    <div class="section-header"><h2>Pull Model</h2></div>
    <div class="card"><div class="card-body">
      <div class="pull-row">
        <input type="text" id="pullInput" placeholder="Search HuggingFace for a model…" autocomplete="off" oninput="onPullInputChange(this.value)">
        <button class="primary" onclick="pullModel()" id="pullBtn">Pull</button>
      </div>
      <div id="pullSuggestions" style="display:none;margin-top:4px"></div>
      <div id="pullProgress" style="display:none">
        <div id="pullModelName"></div>
        <div class="pull-status-row">
          <span id="pullPct">0%</span>
          <span id="pullSpeed"></span>
        </div>
        <div class="pull-track"><div id="pullBar"></div></div>
        <div class="pull-status-row">
          <span id="pullStatus"></span>
          <button class="small danger" id="pullCancelBtn" onclick="cancelPull()">Cancel</button>
        </div>
      </div>
    </div></div>
  </div>
</div>

<!-- ── Model Details Modal ──────────────────────────── -->
<div class="modal" id="modelModal" role="dialog" aria-modal="true" aria-label="Model details">
  <div class="modal-content" onclick="event.stopPropagation()">
    <div class="modal-head">
      <h2 id="modelModalTitle">Model</h2>
      <button class="small danger" onclick="closeModelDetails()" aria-label="Close">Close</button>
    </div>
    <div id="modelModalBody">
      <div class="detail-row"><span class="detail-label">Architecture</span><span class="detail-value" id="md-arch">—</span></div>
      <div class="detail-row"><span class="detail-label">Quantization</span><span class="detail-value" id="md-quant">—</span></div>
      <div class="detail-row"><span class="detail-label">Context Length</span><span class="detail-value" id="md-ctx">—</span></div>
      <div class="detail-row"><span class="detail-label">API Name</span><span class="detail-value" id="md-apiname">—</span></div>
      <div class="detail-row"><span class="detail-label">Size</span><span class="detail-value" id="md-size">—</span></div>
      <div class="detail-row"><span class="detail-label">Path</span><span class="detail-value" id="md-path">—</span></div>
    </div>
    <button class="primary" onclick="launchModelFromDetails()" id="md-launch-btn">Launch</button>
  </div>
</div>

<!-- ── Chat History Modal ──────────────────────────── -->
<div class="modal" id="chatHistoryModal" role="dialog" aria-modal="true" aria-label="Chat history">
  <div class="modal-content" onclick="event.stopPropagation()">
    <div class="modal-head">
      <h2>Chat History</h2>
      <button class="small danger" onclick="closeChatHistory()" aria-label="Close">Close</button>
    </div>
    <div id="chatHistoryList" class="chat-history-list">
      <div class="empty-state"><span class="spinner"></span> Loading…</div>
    </div>
  </div>
</div>

<!-- ── Chat ──────────────────────────────────────────── -->
<div id="view-chat" class="view" role="tabpanel" aria-label="Chat">
  <div class="chat-container">
    <div class="chat-header">
      <div class="header-left">
        <h1>Chat</h1>
        <select id="chatInstanceSelect" onchange="selectChatInstance()" aria-label="Select instance"><option value="">— select a running instance —</option></select>
      </div>
      <div class="header-actions">
        <button class="pill-btn history" onclick="showChatHistory()" aria-label="Chat history" title="Chat history">History</button>
        <button class="pill-btn clear" onclick="clearChat()" aria-label="Clear chat" title="Clear chat">Clear</button>
      </div>
    </div>
    <div id="chatPanel" class="chat-msgs" style="display: none" aria-live="polite"></div>
    <div id="chatEmpty" class="empty-state">
      <div class="icon">▚</div>
      <div class="title">No instance selected</div>
      <p>Launch an instance from the Dashboard to start chatting</p>
    </div>
    <div class="chat-input-row">
      <div id="contextMeter" style="display:none">
        <div id="contextBar"></div>
      </div>
      <div class="input-wrap">
        <textarea id="chatInput" rows="1" placeholder="Type a message… (Enter to send)" onkeydown="if(event.key=='Enter'&&!event.shiftKey){event.preventDefault();sendChat()}" autocomplete="off" oninput="autoGrow(this)"></textarea>
        <button class="primary" onclick="sendChat()">Send</button>
      </div>
      <div id="ctxLabelWrap" style="display:none">
        <span id="ctxLabel" class="ctx-label"></span>
      </div>
    </div>
  </div>
</div>

<!-- ── Image Playground ──────────────────────────────── -->
<div id="view-image" class="view" role="tabpanel" aria-label="Image Generation">
  <div class="image-container">
    <div class="image-header">
      <div class="header-left">
        <h1>Image Playground</h1>
      </div>
      <div class="header-actions">
        <button class="pill-btn" onclick="clearImageHistory()" title="Clear generation history">Clear History</button>
      </div>
    </div>
    <div class="image-body">
      <div class="image-prompt-area">
        <div class="image-input-card">
          <div class="img-model-head">
            <label>Model</label>
            <button class="ghost small" onclick="toggleImageModelSearch()" id="imgModelBrowseBtn">Browse</button>
          </div>
          <select id="imageProfileSelect" aria-label="Select profile" onchange="onImageProfileChange()">
            <option value="">Loading profiles…</option>
          </select>
          <div id="imgModelSearchWrap" style="display:none">
            <input type="text" id="imgModelSearchInput" placeholder="Search image models on HF…" autocomplete="off" oninput="onImageModelSearch(this.value)">
            <div id="imgModelSearchResults" style="display:none;margin-top:4px"></div>
          </div>
          <div id="imgModelList" style="display:none"></div>
        </div>
        <div class="image-input-card">
          <label>Prompt</label>
          <textarea id="imagePromptInput" placeholder="Describe the image you want to generate…" onkeydown="if(event.key=='Enter'&&!event.shiftKey){event.preventDefault();generateImage()}" oninput="autoGrow(this)"></textarea>
        </div>
        <button class="image-gen-btn" onclick="generateImage()" id="imageGenBtn">Generate</button>
        <details class="advanced-flags">
          <summary>Advanced settings</summary>
          <div class="img-adv">
            <div class="img-adv-item"><label>Images</label><input type="number" id="imageNInput" min="1" max="8" placeholder="profile default"></div>
            <div class="img-adv-item"><label>Size</label>
              <select id="imageSizeSelect"><option value="">Profile default</option><option value="512x512">512×512</option><option value="768x768">768×768</option><option value="1024x1024">1024×1024</option><option value="1280x720">1280×720</option><option value="1024x768">1024×768</option><option value="custom">Custom…</option></select>
              <input type="text" id="imageSizeCustom" placeholder="WIDTHxHEIGHT" style="display:none">
            </div>
            <div class="img-adv-item"><label>Steps</label><input type="number" id="imageStepsInput" min="1" max="100" placeholder="profile default"></div>
            <div class="img-adv-item"><label>Guidance</label><input type="number" id="imageGuidanceInput" min="0" max="30" step="0.5" placeholder="profile default"></div>
            <div class="img-adv-item"><label>Seed</label><input type="number" id="imageSeedInput" placeholder="random"></div>
          </div>
        </details>
        <div id="imageHistorySection" style="display:none">
          <div class="img-history-head">
            <span>History</span>
            <span id="imageHistoryCount"></span>
          </div>
          <div id="imageHistoryGrid" class="image-history-grid"></div>
        </div>
      </div>
      <div class="image-results" id="imageResultsArea">
        <div class="image-empty" id="imageEmpty">
          <div class="icon">▦</div>
          <div class="title">No images generated yet</div>
          <p>Type a prompt and click Generate to create an image. Results appear here.</p>
        </div>
      </div>
    </div>
  </div>
</div>

<!-- ── Settings ──────────────────────────────────────── -->
<div id="view-settings" class="view" role="tabpanel" aria-label="Settings">
  <div class="page-header">
    <h1>Settings</h1>
    <p>CONFIGURATION &amp; SYSTEM INFORMATION</p>
  </div>
  <div class="card"><div class="card-body">
    <div class="detail-row">
      <span class="detail-label">gollama</span>
      <span class="detail-value"><span class="tag" id="s-version">—</span></span>
    </div>
    <div class="detail-row">
      <span class="detail-label">llama-server</span>
      <span class="detail-value"><span class="tag" id="s-llama-version">—</span></span>
    </div>
    <div class="detail-row">
      <span class="detail-label">Backend</span>
      <span class="detail-value"><span class="tag" id="s-backend">—</span></span>
    </div>
    <div class="detail-row">
      <span class="detail-label">Config</span>
      <span class="detail-value"><code class="path">~/.gollama/config.json</code></span>
    </div>
    <div class="detail-row">
      <span class="detail-label">Models dir</span>
      <span class="detail-value"><code class="path">~/.gollama/models/</code></span>
    </div>
  </div></div>
  <div class="card" style="margin-top:12px"><div class="card-body">
    <div class="ttl-row">
      <label for="idleTtlInput">Auto-stop idle instances after</label>
      <input type="number" id="idleTtlInput" value="30" min="0" max="1440">
      <span class="ttl-hint">minutes (0 = disable)</span>
      <button class="primary small" onclick="saveIdleTTL()">Save</button>
      <span id="idleTtlStatus" class="save-status"></span>
    </div>
  </div></div>
  <div class="card settings-card" id="settings-ql" style="margin-top:12px"><div class="card-body">
    <div class="settings-card-head">
      <div>
        <div class="settings-card-title">Quick Launch Defaults</div>
        <div class="settings-card-sub">Pre-fills the Quick Launch form for manual launches</div>
      </div>
      <button class="ghost small settings-edit-btn" onclick="toggleEditSettings('ql')">Edit</button>
    </div>
    <div class="settings-readonly" id="ql-readonly">
      <div class="settings-empty">No custom flags set.</div>
    </div>
    <div class="settings-form">
      <div id="settingsFlagsContainer"></div>
      <button class="ghost small" onclick="addSettingsFlag()">+ Add Flag</button>
      <div class="settings-form-actions">
        <button class="primary small" onclick="saveSettingsFlags()">Save</button>
        <button class="secondary small" onclick="cancelEditSettings('ql')">Cancel</button>
        <span id="settingsFlagsStatus" class="save-status"></span>
      </div>
    </div>
  </div></div>
  <div class="card settings-card" id="settings-api" style="margin-top:12px"><div class="card-body">
    <div class="settings-card-head">
      <div>
        <div class="settings-card-title">API Launch Defaults</div>
        <div class="settings-card-sub">Used when auto-launching via API — acts as fallback when no profile matches</div>
      </div>
      <button class="ghost small settings-edit-btn" onclick="toggleEditSettings('api')">Edit</button>
    </div>
    <div class="settings-readonly" id="api-readonly">
      <div class="settings-empty">No custom flags set.</div>
    </div>
    <div class="settings-form">
      <div id="proxyFlagsContainer"></div>
      <button class="ghost small" onclick="addProxyFlag()">+ Add Flag</button>
      <div class="settings-form-actions">
        <button class="primary small" onclick="saveProxyFlags()">Save</button>
        <button class="secondary small" onclick="cancelEditSettings('api')">Cancel</button>
        <span id="proxyFlagsStatus" class="save-status"></span>
      </div>
    </div>
  </div></div>
  <div class="card settings-card" id="settings-profiles" style="margin-top:12px"><div class="card-body">
    <div class="settings-card-head">
      <div>
        <div class="settings-card-title">Model Profiles</div>
        <div class="settings-card-sub">Named flag sets for auto-launch routing. When a model profile's model matches the request, its flags override proxy defaults.</div>
      </div>
      <button class="ghost small settings-edit-btn" onclick="toggleEditSettings('profiles')">Edit</button>
    </div>
    <div class="settings-readonly" id="profiles-readonly">
      <div class="settings-empty">No model profiles configured.</div>
    </div>
    <div class="settings-form">
      <div id="profilesContainer"></div>
      <button class="ghost small" onclick="addProfile('text')">+ Add Text Profile</button>
      <div class="settings-form-actions">
        <button class="primary small" onclick="saveProfiles()">Save</button>
        <button class="secondary small" onclick="cancelEditSettings('profiles')">Cancel</button>
        <span id="profilesStatus" class="save-status"></span>
      </div>
    </div>
  </div></div>
  <div class="card settings-card" id="settings-image-profiles" style="margin-top:12px"><div class="card-body">
    <div class="settings-card-head">
      <div>
        <div class="settings-card-title">Image Profiles</div>
        <div class="settings-card-sub">Image generation model settings. No llama.cpp flags apply.</div>
      </div>
      <button class="ghost small settings-edit-btn" onclick="toggleEditSettings('image-profiles')">Edit</button>
    </div>
    <div class="settings-readonly" id="image-profiles-readonly">
      <div class="settings-empty">No image profiles configured.</div>
    </div>
    <div class="settings-form">
      <div id="imageProfilesContainer"></div>
      <button class="ghost small" onclick="addProfile('image')">+ Add Image Profile</button>
      <div class="settings-form-actions">
        <button class="primary small" onclick="saveProfiles()">Save</button>
        <button class="secondary small" onclick="cancelEditSettings('image-profiles')">Cancel</button>
        <span id="imageProfilesStatus" class="save-status"></span>
      </div>
    </div>
  </div></div>
  <div class="card" style="margin-top:12px"><div class="card-body restart-block">
    <button class="danger" onclick="restartGollama()">Restart gollama</button>
    <div class="restart-note">Applies config changes and picks up new version</div>
  </div></div>
</div>

</main>
</div>

<!-- ── Logs Modal ──────────────────────────────────── -->
<div class="modal" id="logModal" role="dialog" aria-modal="true" aria-label="Instance logs">
  <div class="modal-content" onclick="event.stopPropagation()">
    <div class="modal-head">
      <h2>Logs</h2>
      <div class="modal-head-actions">
        <button class="small secondary" onclick="copyLogs()" aria-label="Copy logs">Copy</button>
        <button class="small danger" onclick="closeLogs()" aria-label="Close logs">Close</button>
      </div>
    </div>
    <pre id="logContent" aria-live="polite"></pre>
  </div>
</div>
<script>
var chatPort = 0, chatHistory = [], chatSessionId = null;
var currentView = 'dashboard';
var cachedModelCount = 0;
var cachedModels = [];
var imageHistory = [];
var imageGenerating = false;

// ── Navigation ──────────────────────────────────────
function switchView(name) {
  document.querySelectorAll('.view').forEach(function(v) { v.classList.remove('active'); });
  document.querySelectorAll('.nav-item').forEach(function(a) { a.classList.remove('active'); });
  var view = document.getElementById('view-' + name);
  if (view) view.classList.add('active');
  document.querySelector('.nav-item[onclick*="' + name + '"]').classList.add('active');
  currentView = name;
  if (name == 'dashboard') { startDashPoll(); loadInstances(); } else { stopDashPoll(); }
  if (name == 'chat') {
    if (chatPort) {
      document.getElementById('chatPanel').style.display = 'block';
      document.getElementById('chatEmpty').style.display = 'none';
      updateContextMeter();
    }
  }
  if (name == 'image') loadImageProfiles();
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
    var fmc = document.getElementById('faceModels'); if (fmc) fmc.textContent = String(m.length);
    mc.innerHTML = m.length + ' downloaded';

    s.innerHTML = '<option value="">— Select model —</option>';
    if (!m || !m.length) {
      s.innerHTML += '<option value="" disabled>No models found. Use gollama pull.</option>';
      ml.innerHTML = '<div class="card-body"><div class="empty-state"><div class="icon">📦</div><div class="title">No models yet</div><p>Pull a model from HuggingFace to get started.</p></div></div>';
      return;
    }

    cachedModels = m;
    var names = [], seen = {};
    m.forEach(function(x) { var n = x.short_name || x.name || '?'; if (!seen[n]) { seen[n] = 1; names.push(n); } });
    names.forEach(function(n, i) { s.innerHTML += '<option value="' + escAttr(n) + '"' + (names.length === 1 ? ' selected' : '') + '>' + escHtml(n) + '</option>'; });
    ml.innerHTML = '<div class="card-body">' + m.map(function(x) {
      var name = x.name || '?', shortName = x.short_name || '', size = x.size ? fmtSize(x.size) : '?';
      var arch = x.architecture || '', quant = x.quantization || '', ctx = x.context_length || 0, badges = [];
      if (shortName) badges.push('<span class="badge badge-green" style="cursor:help" title="API name">' + escHtml(shortName) + '</span>');
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
  document.getElementById('md-apiname').textContent = m.short_name || '—';
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
var instancesPollTimer = null;
var dashPoll = null;

function startDashPoll() { if (dashPoll) return; dashPoll = setInterval(function() { loadInstances(); }, 5000); }
function stopDashPoll() { if (dashPoll) { clearInterval(dashPoll); dashPoll = null; } }

function updateFaceplate(list) {
  var led = document.getElementById('faceLed');
  var n = 0, tps = 0, starting = 0;
  (list || []).forEach(function(i) { if (i.status == 'running') { n++; tps += i.tokens_per_sec || 0; if (!i.ready) starting++; } });
  var fInst = document.getElementById('faceInst'); if (fInst) fInst.textContent = String(n);
  var fTps = document.getElementById('faceTps'); if (fTps) fTps.textContent = tps > 0 ? tps.toFixed(1) : '—';
  if (led) led.className = 'face-led' + (starting ? ' led-amber' : n ? ' led-green' : ' led-off');
}

async function loadInstances() {
  var ic = document.getElementById('instanceCount'), c = document.getElementById('instances'), cs = document.getElementById('chatInstanceSelect');
  try {
    var r = await fetch('/api/v1/instances'), list = await r.json();
    ic.textContent = '(' + list.length + ')';
    updateFaceplate(list);
    var running = list.filter(function(i) { return i.status == 'running'; });

    cs.innerHTML = '<option value="">— select a running instance —</option>';
    list.forEach(function(i) { if (i.type == 'image') return; var mn = i.model || '?'; cs.innerHTML += '<option value="' + i.port + '"' + (chatPort == i.port ? ' selected' : '') + '>' + i.port + ' - ' + (escHtml(mn.length > 35 ? mn.slice(0, 35) + '…' : mn)) + '</option>'; });
    if (!list.length) {
      document.getElementById('chatPanel').style.display = 'none';
      document.getElementById('chatEmpty').style.display = 'flex';
      c.innerHTML = '<div class="empty-state"><div class="icon">🚀</div><div class="title">No running instances</div><p>Launch one from the Quick Launch section above.</p></div>';
      return;
    }
    document.getElementById('chatPanel').style.display = chatPort ? 'block' : 'none';

    c.setAttribute('data-list', JSON.stringify(list));
    c.innerHTML = list.map(function(i) {
      var cls = i.status == 'running' ? (i.ready ? '' : ' starting') : ' stopped';
      var bc = i.status == 'running' ? (i.ready ? 'badge-green' : 'badge-blue') : 'badge-red';
      var statusLabel = i.status == 'running' ? (i.ready ? 'running' : 'starting…') : i.status;
      var mn = i.model || '?';
      var tps = i.tokens_per_sec ? '<span style="color: var(--green); font-variant-numeric: tabular-nums">⚡ ' + i.tokens_per_sec.toFixed(1) + ' t/s</span>' : '';
      var uptime = i.started_at ? (function() { var s = Math.floor((Date.now() - new Date(i.started_at).getTime()) / 1000); return '<span title="Uptime">⏱ ' + (s > 86400 ? Math.floor(s/86400)+'d ' : '') + (s > 3600 ? Math.floor((s%86400)/3600)+'h ' : '') + Math.floor((s%3600)/60)+'m</span>'; })() : '';
      var idle = i.last_activity ? (function() { var s = Math.floor((Date.now() - new Date(i.last_activity).getTime()) / 1000); if (s < 60) return ''; return '<span title="Idle time">💤 ' + (s > 3600 ? Math.floor(s/3600)+'h ' : '') + Math.floor((s%3600)/60)+'m</span>'; })() : '';
      var tokens = i.total_tokens ? '<span title="Total tokens">🔤 ' + (i.total_tokens > 999 ? Math.round(i.total_tokens/1000) + 'K' : i.total_tokens) + '</span>' : '';
      var typeLabel = i.type == 'image' ? '<span class="badge" style="background:var(--accent-bg);color:var(--accent)">🖼️ image</span>' : '<span class="badge" style="background:var(--surface-2);color:var(--text-dim)">💬 text</span>';
var metrics = typeLabel + (i.device_split ? '<span title="Model split">📊 ' + i.device_split + '</span>' : '') + (i.profile ? ' <span class="badge badge-profile" title="Active model profile">📋 ' + escHtml(i.profile) + '</span>' : '');
      var flags = i.flags && i.flags.length ? formatFlags(i.flags) : '';
      var flagsHtml = flags ? '<div style="font-size: 11px; color: var(--text-dim); margin-top: 8px; padding-top: 8px; border-top: 1px solid var(--border); word-break: break-all; font-family: var(--font-mono)">' + escHtml(flags) + '</div>' : '';
      var errDiv = i.status != 'running' ? '<div class="error-line" id="err-' + i.port + '"></div>' : '';
      return '<div class="inst-card' + cls + '"><div class="title">' + escHtml(mn.length > 40 ? mn.slice(0, 40) + '…' : mn) + '</div>' +
        '<div class="meta"><span>Port ' + i.port + '</span>' + tps + uptime + idle + tokens + metrics + '<span class="badge ' + bc + '">' + statusLabel + '</span></div>' +
        errDiv + flagsHtml +
        '<div class="actions"><button class="small danger" onclick="stopInstance(' + i.port + ')" aria-label="Stop instance on port ' + i.port + '">⏹ Stop</button>' +
        '<button class="small secondary" onclick="restartInstance(' + i.port + ')" aria-label="Restart instance on port ' + i.port + '">🔄 Restart</button>' +
        (i.type == 'image' ? '' : '<button class="small secondary" onclick="selectChatFor(' + i.port + ', \'' + escAttr(mn.replace(/'/g, '')) + '\')" aria-label="Chat with instance on port ' + i.port + '">💬 Chat</button>') +
        (i.type == 'image' ? '' : '<button class="small secondary" onclick="window.open(\'http://\' + location.hostname + \':' + i.port + '\', \'_blank\'); return false" aria-label="Open instance on port ' + i.port + '">🌐 Open</button>') +
        '<button class="small secondary" onclick="viewLogs(' + i.port + ')" aria-label="View logs for port ' + i.port + '">📋 Logs</button></div></div>';
    }).join('');
    list.forEach(function(i) { if (i.status != 'running') fetchErrorLog(i.port); });
    var hasStarting = list.some(function(i) { return i.status == 'running' && !i.ready; });
    if (hasStarting) {
      if (instancesPollTimer) clearTimeout(instancesPollTimer);
      instancesPollTimer = setTimeout(loadInstances, 1000);
    } else {
      if (instancesPollTimer) { clearTimeout(instancesPollTimer); instancesPollTimer = null; }
    }
  } catch (e) {
    if (instancesPollTimer) clearTimeout(instancesPollTimer);
    instancesPollTimer = setTimeout(loadInstances, 2000);
  }
}

async function launchInstance() {
  var btn = document.getElementById('launchBtn'), m = document.getElementById('modelSelect').value, p = parseInt(document.getElementById('portInput').value), f = collectFlags(document.getElementById('flagsContainer'));
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

  var userFlags = (inst.flags || []).slice();
  for (var k = 0; k < userFlags.length; k++) { if (userFlags[k] === '-m' || userFlags[k] === '--port') { userFlags.splice(k, 2); k--; } }

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
var commonFlags = [
  '--adaptive-decay','--adaptive-target','--agent',
  '--alias','--api-key','--api-key-file','--api-prefix',
  '--backend-sampling','--batch-size',
  '--cache-idle-slots','--cache-prompt','--cache-ram','--cache-reuse','--cache-type-k','--cache-type-k-draft','--cache-type-v','--cache-type-v-draft','--chat-template','--chat-template-file','--chat-template-kwargs','--check-tensors','--checkpoint-min-step','--cont-batching','--context-shift','--control-vector','--control-vector-layer-range','--control-vector-scaled','--cpu-mask','--cpu-mask-batch','--cpu-mask-batch-draft','--cpu-mask-draft','--cpu-moe','--cpu-moe-draft','--cpu-range','--cpu-range-batch','--cpu-range-draft','--cpu-strict','--cpu-strict-batch','--cpu-strict-batch-draft','--cpu-strict-draft','--ctx-checkpoints','--ctx-size',
  '--defrag-thold','--device','--device-draft','--direct-io','--docker-repo','--dry-allowed-length','--dry-base','--dry-multiplier','--dry-penalty-last-n','--dry-sequence-breaker','--dynatemp-exp','--dynatemp-range',
  '--embd-normalize','--embedding','--escape',
  '--fit','--fit-ctx','--fit-target','--flash-attn','--frequency-penalty',
  '--grammar','--grammar-file',
  '--hf-file','--hf-file-v','--hf-repo','--hf-repo-v','--hf-token','--host',
  '--ignore-eos','--image-max-tokens','--image-min-tokens',
  '--jinja','--json-schema','--json-schema-file',
  '--keep','--kv-unified',
  '--list-devices','--log-colors','--log-disable','--log-file','--log-prefix','--log-prompts-dir','--log-timestamps','--log-verbose','--log-verbosity','--logit-bias','--lookup-cache-dynamic','--lookup-cache-static','--lora','--lora-init-without-apply','--lora-scaled',
  '--main-gpu','--media-path','--metrics','--min-p','--mirostat','--mirostat-ent','--mirostat-lr','--mlock','--mmproj','--mmproj-auto','--mmproj-offload','--mmproj-url','--model','--model-url','--model-vocoder','--models-autoload','--models-dir','--models-max','--models-preset','--mtmd-batch-max-tokens',
  '--n-cpu-moe','--n-cpu-moe-draft','--n-gpu-layers','--n-gpu-layers-draft','--no-agent','--no-cache-idle-slots','--no-cache-prompt','--no-cont-batching','--no-context-shift','--no-direct-io','--no-escape','--no-flash-attn','--no-host','--no-jinja','--no-kv-offload','--no-kv-unified','--no-log-prefix','--no-log-timestamps','--no-mmap','--no-mmproj','--no-mmproj-auto','--no-mmproj-offload','--no-models-autoload','--no-op-offload','--no-perf','--no-prefill-assistant','--no-repack','--no-skip-chat-parsing','--no-slots','--no-spec-draft-backend-sampling','--no-ui','--no-ui-mcp-proxy','--no-warmup','--no-webui','--no-webui-mcp-proxy','--numa',
  '--offline','--op-offload','--override-kv','--override-tensor','--override-tensor-draft',
  '--parallel','--path','--perf','--poll','--poll-batch','--poll-batch-draft','--poll-draft','--pooling','--port','--predict','--prefill-assistant','--presence-penalty','--prio','--prio-batch','--prio-batch-draft','--prio-draft','--props',
  '--reasoning','--reasoning-budget','--reasoning-budget-message','--reasoning-format','--repeat-last-n','--repeat-penalty','--repack','--rerank','--reuse-port','--reverse-prompt','--rope-freq-base','--rope-freq-scale','--rope-scale','--rope-scaling',
  '--samplers','--sampling-seq','--seed','--skip-chat-parsing','--sleep-idle-seconds','--slot-prompt-similarity','--slot-save-path','--special','--spec-draft-backend-sampling','--no-spec-draft-backend-sampling','--spec-draft-cpu-moe','--spec-draft-device','--spec-draft-hf','--spec-draft-model','--spec-draft-n-cpu-moe','--spec-draft-n-max','--spec-draft-n-min','--spec-draft-ngl','--spec-draft-p-min','--spec-draft-p-split','--spec-draft-threads','--spec-draft-threads-batch','--spec-draft-type-k','--spec-draft-type-v','--spec-ngram-map-k-min-hits','--spec-ngram-map-k-size-m','--spec-ngram-map-k-size-n','--spec-ngram-map-k4v-min-hits','--spec-ngram-map-k4v-size-m','--spec-ngram-map-k4v-size-n','--spec-ngram-mod-n-match','--spec-ngram-mod-n-max','--spec-ngram-mod-n-min','--spec-ngram-simple-min-hits','--spec-ngram-simple-size-m','--spec-ngram-simple-size-n','--spec-type','--spm-infill','--split-mode','--sse-ping-interval','--ssl-cert-file','--ssl-key-file','--swa-checkpoints','--swa-full',
  '--tags','--temp','--tensor-split','--threads','--threads-batch','--threads-batch-draft','--threads-draft','--threads-http','--timeout','--tools','--top-k','--top-n-sigma','--top-p','--tts-use-guide-tokens','--typical-p',
  '--ubatch-size','--ui','--ui-config','--ui-config-file','--ui-mcp-proxy','--no-ui-mcp-proxy',
  '--verbose','--verbosity',
  '--xtc-probability','--xtc-threshold',
  '--yarn-attn-factor','--yarn-beta-fast','--yarn-beta-slow','--yarn-ext-factor','--yarn-orig-ctx',
];
var standaloneFlags = {
  '--agent':1,
  '--backend-sampling':1,
  '--cache-idle-slots':1,'--cache-prompt':1,'--check-tensors':1,'--cont-batching':1,'--context-shift':1,'--cpu-moe':1,'--cpu-moe-draft':1,
  '--direct-io':1,
  '--embedding':1,'--escape':1,
  '--ignore-eos':1,
  '--jinja':1,
  '--kv-unified':1,
  '--list-devices':1,'--log-disable':1,'--log-prefix':1,'--no-log-prefix':1,'--log-timestamps':1,'--no-log-timestamps':1,'--log-verbose':1,'--lora-init-without-apply':1,
  '--metrics':1,'--mlock':1,'--mmproj-auto':1,'--models-autoload':1,
  '--no-agent':1,'--no-cache-idle-slots':1,'--no-cache-prompt':1,'--no-cont-batching':1,'--no-context-shift':1,'--no-direct-io':1,'--no-escape':1,'--no-flash-attn':1,'--no-host':1,'--no-jinja':1,'--no-kv-offload':1,'--no-kv-unified':1,'--no-log-prefix':1,'--no-log-timestamps':1,'--no-mmap':1,'--no-mmproj':1,'--no-mmproj-auto':1,'--no-mmproj-offload':1,'--no-models-autoload':1,'--no-op-offload':1,'--no-perf':1,'--no-prefill-assistant':1,'--no-repack':1,'--no-skip-chat-parsing':1,'--no-slots':1,'--no-spec-draft-backend-sampling':1,'--no-ui':1,'--no-ui-mcp-proxy':1,'--no-warmup':1,'--no-webui':1,'--no-webui-mcp-proxy':1,
  '--offline':1,'--op-offload':1,
  '--perf':1,'--prefill-assistant':1,'--props':1,
  '--repack':1,'--rerank':1,'--reuse-port':1,
  '--skip-chat-parsing':1,'--spec-draft-backend-sampling':1,'--spec-draft-cpu-moe':1,'--special':1,'--spm-infill':1,'--swa-full':1,
  '--tts-use-guide-tokens':1,
  '--ui':1,'--ui-mcp-proxy':1,
  '--verbose':1,
};
var flagHints = {
  '--adaptive-decay': 'adaptive-p decay rate (0.0-0.99, default 0.90)',
  '--adaptive-target': 'adaptive-p target probability (-1 = disabled)',
  '--agent': 'enable CORS proxy and all built-in tools',
  '--alias': 'model name alias for API',
  '--api-key': 'API key for authentication',
  '--api-key-file': 'path to API key file',
  '--api-prefix': 'API path prefix',
  '--backend-sampling': 'enable backend sampling (experimental)',
  '--batch-size': 'logical max batch size (default 2048)',
  '--cache-idle-slots': 'save idle slots to prompt cache',
  '--cache-prompt': 'enable prompt caching',
  '--cache-ram': 'max cache size in MiB (default 8192)',
  '--cache-reuse': 'min chunk size to reuse from cache',
  '--cache-type-k': 'f32, f16, bf16, q8_0, q4_0, ...',
  '--cache-type-k-draft': 'KV cache type K for draft model',
  '--cache-type-v': 'f32, f16, bf16, q8_0, q4_0, ...',
  '--cache-type-v-draft': 'KV cache type V for draft model',
  '--check-tensors': 'check model tensor data for invalid values',
  '--checkpoint-min-step': 'minimum spacing between context checkpoints (default 256)',
  '--chat-template': 'jinja chat template name',
  '--chat-template-file': 'path to jinja chat template file',
  '--chat-template-kwargs': 'extra JSON params for template parser',
  '--cont-batching': 'enable continuous batching',
  '--context-shift': 'enable context shift for infinite text gen',
  '--control-vector': 'path to control vector',
  '--control-vector-layer-range': 'START END layer range for control vector',
  '--control-vector-scaled': 'FNAME:SCALE control vector with scaling',
  '--cpu-mask': 'CPU affinity mask (hex)',
  '--cpu-mask-batch': 'CPU affinity mask for batch processing (hex)',
  '--cpu-moe': 'keep MoE weights in CPU',
  '--cpu-range': 'CPU range for affinity (lo-hi)',
  '--cpu-range-batch': 'CPU range for batch affinity (lo-hi)',
  '--cpu-strict': '<0|1> strict CPU placement',
  '--cpu-strict-batch': '<0|1> strict CPU placement for batch',
  '--cpu-strict-batch-draft': '<0|1> strict CPU placement for batch draft',
  '--cpu-strict-draft': '<0|1> strict CPU placement for draft model',
  '--cpu-mask-batch-draft': 'CPU affinity mask for batch draft (hex)',
  '--cpu-mask-draft': 'CPU affinity mask for draft model (hex)',
  '--cpu-moe-draft': 'keep MoE weights for draft in CPU',
  '--cpu-range-draft': 'CPU range for draft affinity (lo-hi)',
  '--device-draft': 'comma-separated devices for draft model',
  '--ctx-checkpoints': 'max context checkpoints per slot (default 32)',
  '--ctx-size': 'context size in tokens',
  '--defrag-thold': 'KV cache defragmentation threshold (deprecated)',
  '--device': 'comma-separated devices for offloading',
  '--direct-io': 'use DirectIO if available',
  '--dry-allowed-length': 'DRY allowed length (default 2)',
  '--dry-base': 'DRY base value (default 1.75)',
  '--dry-multiplier': 'DRY multiplier (0.0 = disabled)',
  '--dry-penalty-last-n': 'DRY penalty last N tokens (-1 = ctx)',
  '--dry-sequence-breaker': 'DRY sequence breaker string',
  '--dynatemp-exp': 'dynamic temperature exponent (default 1.0)',
  '--dynatemp-range': 'dynamic temperature range (default 0.0)',
  '--embedding': 'embedding mode only',
  '--embd-normalize': 'normalization: -1=none, 0=max, 1=taxicab, 2=euclidean',
  '--escape': 'process escape sequences in prompt',
  '--fit': 'on/off — auto-adjust args to fit device memory',
  '--fit-ctx': 'minimum ctx size set by --fit (default 4096)',
  '--fit-target': 'target margin per device for --fit (MiB)',
  '--flash-attn': 'on, off, or auto',
  '--frequency-penalty': '0.0–2.0',
  '--grammar': 'BNF-like grammar string',
  '--grammar-file': 'path to grammar file',
  '--hf-file': 'Hugging Face model file name',
  '--hf-file-v': 'Hugging Face vocoder model file',
  '--hf-repo': '<user>/<model>[:quant] Hugging Face repo',
  '--hf-repo-v': '<user>/<model>[:quant] vocoder HF repo',
  '--hf-token': 'Hugging Face access token',
  '--host': 'IP address (default 0.0.0.0)',
  '--ignore-eos': 'ignore EOS token and continue generating',
  '--image-max-tokens': 'max tokens per image',
  '--image-min-tokens': 'min tokens per image',
  '--jinja': 'use jinja template engine for chat',
  '--json-schema': 'JSON schema string',
  '--json-schema-file': 'path to JSON schema file',
  '--keep': 'tokens to keep from prompt (-1 = all)',
  '--kv-unified': 'shared KV buffer across sequences',
  '--list-devices': 'print list of available devices and exit',
  '--log-colors': 'on/off/auto — colored logging',
  '--log-file': 'path to log file',
  '--log-prefix': 'enable prefix in log messages',
  '--no-log-prefix': 'disable prefix in log messages',
  '--log-timestamps': 'enable timestamps in log messages',
  '--no-log-timestamps': 'disable timestamps in log messages',
  '--log-verbose': 'set verbosity level to infinity',
  '--logit-bias': 'TOKEN_ID(+/-)BIAS',
  '--lookup-cache-dynamic': 'path to dynamic lookup cache',
  '--lookup-cache-static': 'path to static lookup cache',
  '--lora': 'path to LoRA adapter(s)',
  '--lora-scaled': 'FNAME:SCALE LoRA with scaling',
  '--main-gpu': 'GPU index for main GPU (default 0)',
  '--metrics': 'enable Prometheus metrics endpoint',
  '--min-p': '0.0–1.0 (default 0.05)',
  '--mirostat': '0=off, 1=MIROSTAT, 2=MIROSTAT 2.0',
  '--mirostat-ent': 'target entropy (default 5.0)',
  '--mirostat-lr': 'learning rate (default 0.1)',
  '--mlock': 'lock model in RAM',
  '--mmproj': 'path to multimodal projector file',
  '--mmproj-auto': 'auto-download multimodal projector from HF',
  '--no-mmproj-auto': 'disable auto mmproj download',
  '--mmproj-offload': 'offload multimodal projector to GPU',
  '--mmproj-url': 'URL to multimodal projector file',
  '--model': 'path to model file',
  '--model-url': 'model download URL',
  '--mtmd-batch-max-tokens': 'max image tokens per batch (default 1024)',
  '--n-cpu-moe': 'keep MoE of first N layers in CPU',
  '--n-gpu-layers': 'number or "all"',
  '--no-agent': 'disable CORS proxy and built-in tools',
  '--no-cache-idle-slots': 'disable idle slot caching',
  '--no-cache-prompt': 'disable prompt caching',
  '--no-cont-batching': 'disable continuous batching',
  '--no-context-shift': 'disable context shift',
  '--no-direct-io': 'disable DirectIO',
  '--no-escape': 'do not process escape sequences',
  '--no-flash-attn': 'disable flash attention',
  '--no-host': 'disable host buffer',
  '--no-jinja': 'disable jinja template engine',
  '--no-kv-offload': 'disable KV cache offloading',
  '--no-kv-unified': 'separate KV buffers per sequence',
  '--no-mmap': 'disable memory-mapping',
  '--no-mmproj': 'disable multimodal projector',
  '--no-mmproj-offload': 'keep projector in CPU',
  '--no-op-offload': 'disable host tensor op offloading',
  '--no-perf': 'disable performance timings',
  '--no-prefill-assistant': 'disable assistant prefill',
  '--no-repack': 'disable weight repacking',
  '--no-spec-draft-backend-sampling': 'disable backend sampling for draft',
  '--no-ui': 'disable web UI',
  '--no-ui-mcp-proxy': 'disable MCP CORS proxy',
  '--no-warmup': 'skip warmup run',
  '--numa': 'NUMA optimization: distribute, isolate, numactl',
  '--offline': 'offline mode — no network access',
  '--op-offload': 'offload host tensor operations to device',
  '--override-kv': 'KEY=TYPE:VALUE — override model metadata',
  '--override-tensor': '<tensor>=<type> override tensor buffer type',
  '--parallel': 'number of server slots',
  '--path': 'path to serve static files',
  '--perf': 'enable performance timings',
  '--pooling': 'none, mean, cls, last, rank',
  '--port': 'port number (default 8080)',
  '--predict': 'tokens to predict (-1 = infinity)',
  '--prefill-assistant': 'prefill assistant response',
  '--presence-penalty': '0.0–2.0',
  '--prio': 'thread priority: 0=normal, 1=medium, 2=high, 3=realtime',
  '--props': 'enable changing properties via POST /props',
  '--poll': '<0-100> polling level to wait for work (default 50)',
  '--poll-batch': '<0|1> use polling for batch work',
  '--prio-batch': 'batch thread priority: -1=low, 0=normal, 1=medium, 2=high, 3=realtime',
  '--reasoning': 'on, off, or auto',
  '--reasoning-budget': 'token budget for thinking (-1 = unlimited)',
  '--reasoning-budget-message': 'message when budget exhausted',
  '--reasoning-format': 'none, deepseek, deepseek-legacy',
  '--repeat-last-n': 'last N tokens for penalty (0=off, -1=ctx)',
  '--repeat-penalty': '1.0–1.5 (default 1.0)',
  '--repack': 'enable weight repacking',
  '--rerank': 'enable reranking endpoint',
  '--reuse-port': 'allow multiple sockets on same port',
  '--reverse-prompt': 'halt generation at PROMPT',
  '--rope-freq-base': 'RoPE base frequency',
  '--rope-freq-scale': 'RoPE frequency scaling (1/N)',
  '--rope-scale': 'RoPE context scaling factor',
  '--rope-scaling': 'none, linear, yarn',
  '--samplers': 'semicolon-separated sampler list',
  '--sampling-seq': 'simplified sampler sequence (edskypmxt)',
  '--seed': 'RNG seed, -1 = random',
  '--sleep-idle-seconds': 'auto-sleep after N seconds idle (-1=off)',
  '--slot-prompt-similarity': 'prompt match threshold (default 0.1)',
  '--slot-save-path': 'path to save slot KV cache',
  '--special': 'enable special tokens output',
  '--spec-draft-backend-sampling': 'offload draft sampling to backend',
  '--no-spec-draft-backend-sampling': 'disable backend sampling for draft',
  '--spec-draft-cpu-moe': 'keep MoE weights for draft in CPU',
  '--spec-draft-device': 'devices for draft model offloading',
  '--spec-draft-hf': '<user>/<model>[:quant] HF repo for draft',
  '--spec-draft-model': 'path to draft model',
  '--spec-draft-n-cpu-moe': 'keep first N MoE layers in CPU for draft',
  '--spec-draft-n-max': 'draft tokens max (default 3)',
  '--spec-draft-n-min': 'draft tokens min (default 0)',
  '--spec-draft-ngl': 'GPU layers for draft model',
  '--spec-draft-p-min': 'minimum speculative decoding probability (default 0.00)',
  '--spec-draft-p-split': 'speculative split probability (default 0.1)',
  '--spec-draft-threads': 'CPU threads for draft model',
  '--spec-draft-type-k': 'KV cache data type K for draft (f16, q8_0, ...)',
  '--spec-draft-type-v': 'KV cache data type V for draft (f16, q8_0, ...)',
  '--spec-ngram-map-k-min-hits': 'min hits for ngram-map-k (default 1)',
  '--spec-ngram-map-k-size-m': 'draft m-gram size for ngram-map-k (default 48)',
  '--spec-ngram-map-k-size-n': 'lookup n-gram size for ngram-map-k (default 12)',
  '--spec-ngram-map-k4v-min-hits': 'min hits for ngram-map-k4v (default 1)',
  '--spec-ngram-map-k4v-size-m': 'draft m-gram size for ngram-map-k4v (default 48)',
  '--spec-ngram-map-k4v-size-n': 'lookup n-gram size for ngram-map-k4v (default 12)',
  '--spec-ngram-mod-n-match': 'ngram-mod lookup length (default 24)',
  '--spec-ngram-mod-n-max': 'max ngram tokens for ngram-mod (default 64)',
  '--spec-ngram-mod-n-min': 'min ngram tokens for ngram-mod (default 48)',
  '--spec-ngram-simple-min-hits': 'min hits for ngram-simple (default 1)',
  '--spec-ngram-simple-size-m': 'draft m-gram size for ngram-simple (default 48)',
  '--spec-ngram-simple-size-n': 'lookup n-gram size for ngram-simple (default 12)',
  '--spec-type': 'none, draft-simple, draft-mtp, ngram-cache, ...',
  '--spm-infill': 'Suffix/Prefix/Middle infill pattern',
  '--split-mode': 'none, layer, row, tensor',
  '--sse-ping-interval': 'SSE ping interval in seconds (default 30)',
  '--ssl-cert-file': 'path to SSL cert file (PEM)',
  '--ssl-key-file': 'path to SSL key file (PEM)',
  '--swa-full': 'use full-size SWA cache',
  '--tags': 'comma-separated model tags',
  '--temp': '0.0–2.0 (default 0.7)',
  '--tensor-split': 'GPU split proportions, e.g. 3,1',
  '--threads': 'CPU thread count',
  '--threads-batch': 'batch/prompt processing threads',
  '--threads-http': 'HTTP request threads',
  '--timeout': 'server read/write timeout in seconds (default 3600)',
  '--tools': 'comma-separated built-in tools (or "all")',
  '--top-k': 'e.g. 40 (0 = off)',
  '--top-n-sigma': 'top-n-sigma sampling (-1 = disabled)',
  '--top-p': '0.0–1.0 (default 0.95)',
  '--typical-p': 'locally typical sampling (1.0 = disabled)',
  '--ubatch-size': 'physical max batch size (default 512)',
  '--ui': 'enable/disable the web UI',
  '--ui-config': 'JSON string for default UI settings',
  '--ui-config-file': 'path to JSON file for default UI settings',
  '--ui-mcp-proxy': 'enable MCP CORS proxy (experimental)',
  '--no-ui-mcp-proxy': 'disable MCP CORS proxy',
  '--verbose': 'set verbosity level to infinity',
  '--verbosity': 'set verbosity threshold (0-5, default 3)',
  '--xtc-probability': 'XTC probability (0.0 = disabled)',
  '--xtc-threshold': 'XTC threshold (default 0.1)',
  '--yarn-attn-factor': 'YaRN attention magnitude scale',
  '--yarn-beta-fast': 'YaRN low correction dim',
  '--yarn-beta-slow': 'YaRN high correction dim',
  '--yarn-ext-factor': 'YaRN extrapolation mix factor',
  '--yarn-orig-ctx': 'YaRN original context size',

  '--docker-repo': '<repo>/<model>[:quant] Docker Hub repo',
  '--log-disable': 'disable all logging',
  '--log-prompts-dir': 'directory to log prompts (debug)',
  '--log-verbosity': 'verbosity threshold 0-5 (default 3)',
  '--lora-init-without-apply': 'load LoRA adapters without applying',
  '--media-path': 'directory for local media files',
  '--model-vocoder': 'path to vocoder model file',
  '--models-autoload': 'auto-load models for router server',
  '--models-dir': 'directory for router server models',
  '--models-max': 'max models to load simultaneously (default 4)',
  '--models-preset': 'path to router model presets INI file',
  '--n-cpu-moe-draft': 'first N MoE layers in CPU for draft',
  '--n-gpu-layers-draft': 'GPU layers for draft model',
  '--no-models-autoload': 'disable auto-load for router server',
  '--no-skip-chat-parsing': 'disable pure content parser',
  '--no-slots': 'disable slots monitoring endpoint',
  '--no-webui': 'disable web UI',
  '--no-webui-mcp-proxy': 'disable WebUI MCP CORS proxy',
  '--override-tensor-draft': 'override tensor buffer type for draft',
  '--poll-batch-draft': '<0|1> use polling for batch draft work',
  '--poll-draft': '<0|1> use polling for draft work (default same as --poll)',
  '--prio-batch-draft': 'draft batch thread priority',
  '--prio-draft': 'draft thread priority',
  '--skip-chat-parsing': 'force pure content parser for chat',
  '--spec-draft-threads-batch': 'batch threads for draft model',
  '--swa-checkpoints': 'max SWA context checkpoints per slot (default 32)',
  '--threads-batch-draft': 'batch processing threads for draft',
  '--threads-draft': 'CPU threads for draft model generation',
  '--tts-use-guide-tokens': 'use guide tokens to improve TTS word recall',
};

function makeFlagRow(name, value) {
  var r = document.createElement('div'); r.className = 'flag-row';
  var isCustom = name && commonFlags.indexOf(name) === -1;
  var displayName = isCustom ? '' : (name || '');
  r.innerHTML =
    '<div class="flag-search-wrapper">' +
      '<input type="text" class="flag-search" placeholder="Search or type flag\u2026" autocomplete="off" value="' + escAttr(displayName) + '">' +
      '<div class="flag-dropdown" style="display:none"></div>' +
    '</div>' +
    '<input type="text" class="flag-custom" placeholder="Flag name" autocomplete="off" style="display:' + (isCustom ? '' : 'none') + '" value="' + (isCustom ? escAttr(name) : '') + '">' +
    '<input type="text" class="flag-value" placeholder="Value" autocomplete="off" value="' + escAttr(value || '') + '">' +
    '<button class="small danger" onclick="this.parentElement.remove()" aria-label="Remove flag">\u2715</button>';
  if (!isCustom && standaloneFlags[name]) r.querySelector('.flag-value').style.display = 'none';
  setupFlagDropdown(r);
  return r;
}

function addFlag() {
  document.getElementById('flagsContainer').appendChild(makeFlagRow('', ''));
}

function setupFlagDropdown(row) {
  var wrapper = row.querySelector('.flag-search-wrapper');
  var input = wrapper.querySelector('.flag-search');
  var dropdown = wrapper.querySelector('.flag-dropdown');
  var customInput = row.querySelector('.flag-custom');
  var valInput = row.querySelector('.flag-value');

  function buildItems(query) {
    var q = query.toLowerCase();
    var html = '';
    var found = false;
    commonFlags.forEach(function(f) {
      if (f.indexOf(q) !== -1) {
        html += '<div class="flag-dropdown-item" data-value="' + escAttr(f) + '">' + escHtml(f) + '</div>';
        found = true;
      }
    });
    if (!found && q) {
      html += '<div class="flag-dropdown-item flag-dropdown-empty">No flags match\u2026</div>';
    }
    html += '<div class="flag-dropdown-divider"></div>';
    html += '<div class="flag-dropdown-item flag-dropdown-item-custom" data-value="">Custom\u2026</div>';
    dropdown.innerHTML = html;
  }

  function selectValue(val) {
    if (val) {
      input.value = val;
      customInput.style.display = 'none';
      customInput.value = '';
      var isStandalone = standaloneFlags[val];
      valInput.style.display = isStandalone ? 'none' : '';
      valInput.placeholder = flagHints[val] || 'Value';
    } else {
      input.value = '';
      customInput.style.display = '';
      customInput.focus();
    }
    dropdown.style.display = 'none';
    input.classList.remove('open');
  }

  input.addEventListener('focus', function() {
    buildItems(input.value);
    dropdown.style.display = '';
    input.classList.add('open');
  });

  input.addEventListener('input', function() {
    buildItems(input.value);
    dropdown.style.display = '';
    input.classList.add('open');
    customInput.style.display = 'none';
    valInput.style.display = '';
    valInput.placeholder = 'Value';
    customInput.value = '';
  });

  dropdown.addEventListener('mousedown', function(e) {
    var item = e.target.closest('.flag-dropdown-item');
    if (!item) return;
    e.preventDefault();
    var val = item.getAttribute('data-value');
    if (val === null) return;
    selectValue(val);
    input.focus();
  });

  input.addEventListener('keydown', function(e) {
    var items = dropdown.querySelectorAll('.flag-dropdown-item');
    var sel = dropdown.querySelector('.sel');
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (!sel) {
        if (items[0]) items[0].classList.add('sel');
      } else {
        sel.classList.remove('sel');
        var next = sel.nextElementSibling;
        if (next && next.classList.contains('flag-dropdown-item')) {
          next.classList.add('sel');
        } else if (items[0]) {
          items[0].classList.add('sel');
        }
      }
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      if (!sel) {
        var last = items[items.length - 1];
        if (last) last.classList.add('sel');
      } else {
        sel.classList.remove('sel');
        var prev = sel.previousElementSibling;
        if (prev && prev.classList.contains('flag-dropdown-item')) {
          prev.classList.add('sel');
        } else if (items[items.length - 1]) {
          items[items.length - 1].classList.add('sel');
        }
      }
    } else if (e.key === 'Enter') {
      e.preventDefault();
      if (sel) {
        var val = sel.getAttribute('data-value');
        if (val !== null) selectValue(val);
      } else {
        var first = items[0];
        if (first) {
          var val = first.getAttribute('data-value');
          if (val !== null) selectValue(val);
        }
      }
    } else if (e.key === 'Escape') {
      dropdown.style.display = 'none';
      input.classList.remove('open');
    }
  });

  input.addEventListener('blur', function() {
    setTimeout(function() {
      dropdown.style.display = 'none';
      input.classList.remove('open');
    }, 200);
  });
}

// Parse a flat flag array into rows, respecting standalone booleans
function renderFlags(container, flags) {
  container.innerHTML = '';
  if (!flags || !flags.length) return;
  for (var i = 0; i < flags.length; i++) {
    var name = flags[i];
    if (!name.startsWith('--')) continue;
    var value = '';
    if (i + 1 < flags.length && !standaloneFlags[name] && !flags[i + 1].startsWith('--')) {
      value = flags[i + 1];
      i++;
    }
    container.appendChild(makeFlagRow(name, value));
  }
}
// Serialize rows from a container into a flat flag array, skipping values for standalone booleans
function collectFlags(container) {
  var flags = [];
  container.querySelectorAll('.flag-row').forEach(function(row) {
    var searchInput = row.querySelector('.flag-search'), valInput = row.querySelector('.flag-value'), customInput = row.querySelector('.flag-custom');
    var name = (searchInput ? searchInput.value.trim() : '') || customInput.value.trim();
    var val = valInput.value.trim();
    if (!name) return;
    flags.push(name);
    if (val && !standaloneFlags[name]) flags.push(val);
  });
  return flags;
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
    renderFlags(document.getElementById('flagsContainer'), flags);
    sel.value = '';
  });
}

async function savePreset() {
  var name = prompt('Preset name:');
  if (!name) return;
  var flags = collectFlags(document.getElementById('flagsContainer'));
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

var _pullAbort = null;

function cancelPull() {
  if (_pullAbort) { _pullAbort.abort(); _pullAbort = null; }
  document.getElementById('pullStatus').textContent = 'Cancelled';
  document.getElementById('pullCancelBtn').style.display = 'none';
  document.getElementById('pullBar').style.width = '0%';
  document.getElementById('pullPct').textContent = '';
  document.getElementById('pullSpeed').textContent = '';
  setTimeout(function() {
    document.getElementById('pullProgress').style.display = 'none';
    document.getElementById('pullBtn').disabled = false;
    document.getElementById('pullBtn').textContent = 'Pull';
    loadModels();
  }, 500);
}

async function startPull(ref, btn) {
  document.getElementById('pullSuggestions').style.display = 'none';
  btn.disabled = true;
  document.getElementById('pullProgress').style.display = 'block';
  document.getElementById('pullModelName').textContent = ref;
  document.getElementById('pullBar').style.width = '0%';
  document.getElementById('pullPct').textContent = '0%';
  document.getElementById('pullSpeed').textContent = '';
  document.getElementById('pullStatus').textContent = 'Connecting…';
  document.getElementById('pullCancelBtn').style.display = '';
  _pullAbort = new AbortController();

  try {
    var r = await fetch('/api/v1/models/pull/stream?model=' + encodeURIComponent(ref), { signal: _pullAbort.signal });
    if (!r.ok) {
      if (_pullAbort) _pullAbort = null;
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
            var partLabel = (d.total_parts > 1) ? ' [' + d.part + '/' + d.total_parts + ']' : '';
            document.getElementById('pullStatus').textContent = 'Downloading\u2026' + partLabel;
          } else if (d.status === 'done') {
            done = true;
            document.getElementById('pullBar').style.width = '100%';
            document.getElementById('pullPct').textContent = '100%';
            document.getElementById('pullStatus').textContent = '\u2713 Done';
            document.getElementById('pullCancelBtn').style.display = 'none';
            loadModels();
            setTimeout(function() {
              document.getElementById('pullProgress').style.display = 'none';
              btn.disabled = false; btn.textContent = 'Pull';
            }, 2000);
          } else if (d.status === 'exists') {
            done = true;
            document.getElementById('pullStatus').textContent = 'Already exists';
            document.getElementById('pullCancelBtn').style.display = 'none';
            loadModels();
            setTimeout(function() {
              document.getElementById('pullProgress').style.display = 'none';
              btn.disabled = false; btn.textContent = 'Pull';
            }, 2000);
          } else if (d.status === 'error') {
            done = true;
            document.getElementById('pullStatus').textContent = 'Error: ' + d.error;
            document.getElementById('pullCancelBtn').style.display = 'none';
            btn.disabled = false; btn.textContent = 'Pull';
          }
        } catch (e) {}
      }
    }
    _pullAbort = null;
  } catch (e) {
    if (e.name === 'AbortError') {
      // Cancel handled by cancelPull() — do nothing
    } else {
      document.getElementById('pullStatus').textContent = 'Error: ' + e.message;
      btn.disabled = false; btn.textContent = 'Retry';
    }
    _pullAbort = null;
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
      var id = m.id, label = id;
      var size = m.size ? fmtSize(m.size) : '';
      var likes = m.likes > 0 ? '♥' + m.likes : '';
      var downloads = m.downloads > 0 ? (m.downloads > 999 ? (m.downloads/1000).toFixed(0) + 'K' : m.downloads) + ' dl' : '';
      var tag = m.pipeline_tag ? '<span class="badge badge-amber">' + escHtml(m.pipeline_tag) + '</span>' : '';
      var meta = [tag, size, likes, downloads].filter(Boolean).join(' · ');
      return '<div class="suggestion" onclick="showRepoFiles(\'' + escAttr(id) + '\')"><div><div class="name">' + escHtml(label) + '</div>' + (meta ? '<div class="meta">' + meta + '</div>' : '') + '</div><span class="badge badge-green" style="cursor:pointer">View quants →</span></div>';
    }).join('');
    sg.style.display = 'block';
  } catch (e) { sg.style.display = 'none'; }
}

var _cachedRepoFiles = {};
async function showRepoFiles(repo) {
  var sg = document.getElementById('pullSuggestions');
  sg.innerHTML = '<div style="padding:12px;text-align:center"><span class="spinner"></span> Loading quants…</div>';
  sg.style.display = 'block';
  try {
    var r = await fetch('/api/v1/models/repo-files?repo=' + encodeURIComponent(repo));
    if (!r.ok) { sg.innerHTML = '<div class="suggestion" style="cursor:pointer" onclick="doSearch(\'' + escAttr(document.getElementById('pullInput').value) + '\')">← Back to search. Error loading quants.</div>'; return; }
    var files = await r.json();
    _cachedRepoFiles[repo] = files;
    if (!files || !files.length) {
      sg.innerHTML = '<div class="suggestion" style="cursor:pointer" onclick="doSearch(\'' + escAttr(document.getElementById('pullInput').value) + '\')">← Back. No GGUF files found in this repo.</div>';
      return;
    }
    var html = '<div class="suggestion" style="font-weight:600;border-bottom:1px solid var(--border);cursor:pointer" onclick="doSearch(\'' + escAttr(document.getElementById('pullInput').value) + '\')">← ' + escHtml(repo) + '</div>';
    files.forEach(function(f) {
      var quant = f.quant ? '<span class="badge badge-blue">' + escHtml(f.quant) + '</span>' : '<span class="badge" style="background:var(--surface-2);color:var(--text-dim)">raw</span>';
      var size = f.size ? fmtSize(f.size) : '?';
      var parts = f.file_count > 1 ? ' · <span style="color:var(--text-dim)">' + f.file_count + ' files</span>' : '';
      html += '<div class="suggestion"><div><div class="name">' + quant + ' <span style="font-weight:400;font-size:12px;color:var(--text-dim)">' + size + '</span>' + parts + '</div></div><button class="small primary pull-btn" onclick="pullQuant(\'' + escAttr(repo) + '\', \'' + escAttr(f.quant) + '\', this)">Pull</button></div>';
    });
    sg.innerHTML = html;
  } catch (e) {
    sg.innerHTML = '<div class="suggestion" style="cursor:pointer" onclick="doSearch(\'' + escAttr(document.getElementById('pullInput').value) + '\')">← Back. Error: ' + escHtml(e.message) + '</div>';
  }
}

function pullQuant(repo, quant, btn) {
  var ref = quant ? 'hf.co/' + repo + ':' + quant : 'hf.co/' + repo;
  document.getElementById('pullSuggestions').style.display = 'none';
  startPull(ref, document.getElementById('pullBtn'));
}

// ── Default Flags ─────────────────────────────────────
async function loadDefaultFlags() {
  try {
    var r = await fetch('/api/v1/config/default-flags'), flags = await r.json();
    renderFlags(document.getElementById('flagsContainer'), flags);
  } catch (e) {}
}

// ── Chat ──────────────────────────────────────────────
function selectChatInstance() {
  var s = document.getElementById('chatInstanceSelect'); chatPort = parseInt(s.value) || 0;
  if (chatPort) { chatHistory = []; document.getElementById('chatPanel').innerHTML = ''; document.getElementById('chatPanel').style.display = 'block'; document.getElementById('chatEmpty').style.display = 'none'; resolveCtxLimit(); updateContextMeter(); addSystemMsg('Connected'); }
  else { document.getElementById('chatPanel').style.display = 'none'; document.getElementById('chatEmpty').style.display = 'flex'; }
}
function selectChatFor(port, model) {
  chatPort = port; chatHistory = []; chatSessionId = null;
  document.getElementById('chatInstanceSelect').value = port;
  document.getElementById('chatPanel').innerHTML = '';
  document.getElementById('chatPanel').style.display = 'block';
  document.getElementById('chatEmpty').style.display = 'none';
  resolveCtxLimit();
  updateContextMeter();
  addSystemMsg('Chatting with ' + (model || 'port ' + port));
  if (currentView != 'chat') switchView('chat');
  setTimeout(function() { document.getElementById('chatInput').focus(); }, 100);
}
function addSystemMsg(t) { var c = document.getElementById('chatPanel'); c.innerHTML += '<div class="msg system">' + escHtml(t) + '</div>'; c.scrollTop = c.scrollHeight; }
function addMsg(r, t, re) {
  var c = document.getElementById('chatPanel');
  if (!re && t) {
    var si = t.indexOf('<think>');
    if (si >= 0) {
      var ei = t.indexOf('</think>', si);
      if (ei >= 0) {
        re = t.slice(si + 7, ei);
        t = t.slice(0, si) + t.slice(ei + 8);
      }
    }
  }
  var el = document.createElement('div'); el.className = 'msg ' + r;
  el.textContent = t;
  if (r === 'assistant' && t) {
    var copyBtn = document.createElement('button');
    copyBtn.className = 'copy-btn'; copyBtn.textContent = '📋';
    copyBtn.title = 'Copy message';
    copyBtn.onclick = function() {
      var ok = function() { copyBtn.textContent = '✓'; setTimeout(function() { copyBtn.textContent = '📋'; }, 1500); };
      if (navigator.clipboard && navigator.clipboard.writeText) { navigator.clipboard.writeText(t).then(ok, function() { fallbackCopy(t, ok); }); }
      else { fallbackCopy(t, ok); }
    };
    el.appendChild(copyBtn);
  }
  if (re) { el.insertAdjacentHTML('beforebegin', '<div class="reasoning">' + escHtml(re) + '</div>'); }
  c.appendChild(el); c.scrollTop = c.scrollHeight; return el;
}
function autoGrow(el) { el.style.height = 'auto'; el.style.height = Math.min(el.scrollHeight, 200) + 'px'; }
function clearChat() { chatHistory = []; chatSessionId = null; var p = document.getElementById('chatPanel'); p.innerHTML = ''; addSystemMsg('Chat cleared'); }
function escHtml(s) { return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;'); }
function fallbackCopy(text, done) {
  var ta = document.createElement('textarea');
  ta.value = text; ta.style.position = 'fixed'; ta.style.left = '-9999px';
  document.body.appendChild(ta); ta.select();
  try { document.execCommand('copy'); if (done) done(); } catch (e) {}
  document.body.removeChild(ta);
}
function escAttr(s) { return s.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;'); }

// ── Context window ─────────────────────────────────────
var chatCtxLimit = 0;
var chatLastIdleUpdate = 0;

function estimateTokens(text) {
  // Rough estimate: ~4 chars per token for English, ~2 chars per token for code/CJK
  var len = text.length;
  // Count non-ASCII characters (roughly 2 bytes per char for CJK)
  var nonAscii = 0;
  for (var i = 0; i < len; i++) { if (text.charCodeAt(i) > 127) nonAscii++; }
  return Math.ceil((len - nonAscii) / 4 + nonAscii / 2);
}

function historyTokens() {
  var t = 0;
  for (var i = 0; i < chatHistory.length; i++) {
    t += estimateTokens(chatHistory[i].content || '');
    t += 4; // overhead per message
  }
  t += 20; // system prompt overhead
  return t;
}

function updateContextMeter() {
  var el = document.getElementById('ctxLabel');
  var wrap = document.getElementById('ctxLabelWrap');
  if (!chatCtxLimit || !chatHistory.length) {
    document.getElementById('contextMeter').style.display = 'none';
    wrap.style.display = 'none';
    el.textContent = '';
    return;
  }
  var used = historyTokens();
  var pct = Math.min(used / chatCtxLimit * 100, 100);
  var bar = document.getElementById('contextBar');
  bar.style.width = pct + '%';
  if (pct > 85) bar.style.background = 'var(--red)';
  else if (pct > 65) bar.style.background = 'var(--amber)';
  else bar.style.background = 'var(--accent)';
  document.getElementById('contextMeter').style.display = pct > 10 ? 'block' : 'none';
  wrap.style.display = pct > 10 ? 'flex' : 'none';
  el.textContent = (used > 999 ? Math.round(used/1000) + 'K' : used) + ' / ' + (chatCtxLimit > 999 ? Math.round(chatCtxLimit/1000) + 'K' : chatCtxLimit) + ' tokens (' + pct.toFixed(0) + '%)';
}

function trimHistory() {
  if (!chatCtxLimit) return;
  var target = Math.floor(chatCtxLimit * 0.65);
  while (historyTokens() > target && chatHistory.length > 4) {
    // Remove oldest user+assistant pair, keep system
    for (var i = 0; i < chatHistory.length; i++) {
      if (i > 0 && chatHistory[i].role !== 'system' && (chatHistory[i-1].role === 'user' || chatHistory[i-1].role === 'system')) {
        chatHistory.splice(i-1, 2);
        break;
      }
    }
  }
}

function resolveCtxLimit() {
  chatCtxLimit = 0;
  var list = JSON.parse(document.querySelector('#instances').getAttribute('data-list') || '[]');
  for (var i = 0; i < list.length; i++) { if (list[i].port == chatPort && list[i].model) { chatCtxLimit = 8192; break; } }
  // Try to find model in cachedModels for a more accurate context_length
  for (var i = 0; i < cachedModels.length; i++) {
    var cm = cachedModels[i];
    var list = JSON.parse(document.querySelector('#instances').getAttribute('data-list') || '[]');
    for (var j = 0; j < list.length; j++) {
      if (list[j].port == chatPort && list[j].model && list[j].model.indexOf(cm.name.split(':')[0]) >= 0) {
        if (cm.context_length) chatCtxLimit = cm.context_length;
        return;
      }
    }
  }
}

async function sendChat() {
  var input = document.getElementById('chatInput'), msg = input.value.trim();
  if (!msg || !chatPort) return;
  input.value = ''; addMsg('user', msg); chatHistory.push({ role: 'user', content: msg });
  resolveCtxLimit();
  trimHistory();
  updateContextMeter();
  var content = '', reasoning = '', msgEl = null, reasoningEl = null, thinking = false, chatPanel = document.getElementById('chatPanel');
  try {
    var r = await fetch('/api/v1/chat?port=' + chatPort, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ model: 'default', messages: chatHistory.slice(-20), max_tokens: 4096, stream: true }) });
    if (!r.ok) {
      var errText = await r.text();
      var msg;
      try { var errJson = JSON.parse(errText); msg = errJson.error || errText; } catch(e) { msg = errText; }
      addMsg('system', 'Server error (' + r.status + '): ' + msg);
      return;
    }
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
            var txt = delta.content;
            if (thinking) {
              reasoning += txt;
            } else {
              var ti = txt.indexOf('<think>');
              var ti2 = ti >= 0 ? -1 : txt.indexOf('<think');
              if (ti >= 0) {
                thinking = true;
                reasoning += txt.slice(ti + 7);
              } else if (ti2 >= 0) {
                thinking = true;
                reasoning += txt.slice(ti2);
              } else {
                content += txt;
                if (!msgEl) { msgEl = document.createElement('div'); msgEl.className = 'msg assistant'; chatPanel.appendChild(msgEl); }
                msgEl.textContent = content;
              }
            }
            if (thinking) {
              var endIdx = reasoning.indexOf('</think>');
              if (endIdx >= 0) {
                var after = reasoning.slice(endIdx + 8);
                reasoning = reasoning.slice(0, endIdx);
                if (reasoning.indexOf('<think>') === 0) reasoning = reasoning.slice(7);
                else if (reasoning.indexOf('<think') === 0) reasoning = reasoning.slice(6);
                thinking = false;
                if (after) {
                  content += after;
                  if (!msgEl) { msgEl = document.createElement('div'); msgEl.className = 'msg assistant'; chatPanel.appendChild(msgEl); }
                  msgEl.textContent = content;
                }
              }
              var displayR = reasoning;
              if (displayR.indexOf('<think>') === 0) displayR = displayR.slice(7);
              else if (displayR.indexOf('<think') === 0) displayR = displayR.slice(6);
              if (displayR) {
                if (!reasoningEl) { reasoningEl = document.createElement('div'); reasoningEl.className = 'reasoning'; chatPanel.appendChild(reasoningEl); }
                reasoningEl.textContent = displayR;
              }
            }
          }
        } catch (e) {}
      }
      chatPanel.scrollTop = chatPanel.scrollHeight;
    }
    if (thinking && reasoning) {
      // Model never closed </think> - flush reasoning as content
      var remaining = reasoning;
      if (remaining.indexOf('<think>') === 0) remaining = remaining.slice(7);
      else if (remaining.indexOf('<think') === 0) remaining = remaining.slice(6);
      content = remaining;
      reasoning = '';
      thinking = false;
      if (reasoningEl) { reasoningEl.remove(); reasoningEl = null; }
      if (!msgEl) { msgEl = document.createElement('div'); msgEl.className = 'msg assistant'; chatPanel.appendChild(msgEl); }
      msgEl.textContent = content;
    }
    if (msgEl) {
      var saveContent = reasoning ? '<think>' + reasoning + '</think>' + content : content;
      chatHistory.push({ role: 'assistant', content: saveContent });
      saveChatHistory();
    }
    updateContextMeter();
  } catch (e) { if (msgEl) { msgEl.innerHTML = 'Error: ' + escHtml(e.message); msgEl.className = 'msg system'; } else { var em = addMsg('assistant', ''); em.innerHTML = 'Error: ' + escHtml(e.message); em.className = 'msg system'; } }
}

// ── Chat History ────────────────────────────────────────
function generateChatTitle() {
  for (var i = 0; i < chatHistory.length; i++) {
    if (chatHistory[i].role === 'user' && chatHistory[i].content) {
      var t = chatHistory[i].content;
      return t.length > 60 ? t.slice(0, 60) + '…' : t;
    }
  }
  return '';
}

async function saveChatHistory() {
  if (!chatPort || !chatHistory.length) return;
  var model = '';
  var list = JSON.parse(document.querySelector('#instances').getAttribute('data-list') || '[]');
  for (var i = 0; i < list.length; i++) { if (list[i].port == chatPort) { model = list[i].model || ''; break; } }
  var title = generateChatTitle();
  try {
    var payload = { id: chatSessionId, model: model, messages: chatHistory };
    if (title) payload.title = title;
    var r = await fetch('/api/v1/chats/' + (chatSessionId || ''), { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload) });
    if (r.ok) {
      var d = await r.json();
      if (d.id) chatSessionId = d.id;
    }
  } catch (e) {}
}

async function renameChat(id, currentTitle) {
  var newTitle = prompt('Rename chat:', currentTitle);
  if (!newTitle || newTitle === currentTitle) return;
  try {
    var r = await fetch('/api/v1/chats/' + id);
    if (!r.ok) return;
    var session = await r.json();
    session.title = newTitle;
    await fetch('/api/v1/chats/' + id, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(session) });
    showChatHistory();
  } catch (e) {}
}

async function showChatHistory() {
  document.getElementById('chatHistoryModal').style.display = 'block';
  var list = document.getElementById('chatHistoryList');
  try {
    var r = await fetch('/api/v1/chats'), chats = await r.json();
    if (!chats || !chats.length) { list.innerHTML = '<div class="empty-state"><div class="icon">💬</div><div class="title">No saved chats</div></div>'; return; }
    list.innerHTML = chats.map(function(c) {
      var title = c.title || c.preview || '(empty)';
      var shortModel = c.model ? c.model.split('/').pop().split(':')[0].replace(/-GGUF$/i, '').slice(0, 25) : '';
      return '<div class="chat-history-item" data-id="' + c.id + '"><div class="chat-history-main" onclick="loadChat(\'' + c.id + '\')"><div class="chat-history-title">' + escHtml(title.length > 50 ? title.slice(0, 50) + '…' : title) + '</div><div class="chat-history-meta">' + c.msg_count + ' msgs · ' + escHtml(shortModel) + ' · ' + new Date(c.updated_at).toLocaleDateString() + '</div></div><div class="chat-history-actions"><button class="small ghost" onclick="event.stopPropagation();renameChat(\'' + c.id + '\', \'' + escAttr(title.replace(/'/g, '')) + '\')" title="Rename">✏️</button><button class="small danger" onclick="event.stopPropagation();deleteChat(\'' + c.id + '\')" title="Delete">🗑</button></div></div>';
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
    resolveCtxLimit();
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
    updateContextMeter();
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
function copyLogs() {
  var el = document.getElementById('logContent');
  if (!el || !el.textContent) return;
  if (navigator.clipboard) {
    navigator.clipboard.writeText(el.textContent);
  } else {
    var ta = document.createElement('textarea');
    ta.value = el.textContent;
    ta.style.position = 'fixed'; ta.style.left = '-9999px';
    document.body.appendChild(ta);
    ta.select();
    document.execCommand('copy');
    document.body.removeChild(ta);
  }
}
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
  var meta = document.querySelector('meta[name="theme-color"]'); if (meta) meta.content = b.classList.contains('light') ? '#f3f4f6' : '#0f1115';
  document.getElementById('themeToggle').innerHTML = b.classList.contains('light') ? '<span>☀️</span><span class="label">Theme</span>' : '<span>🌙</span><span class="label">Theme</span>';
  localStorage.setItem('gollama-theme', b.classList.contains('light') ? 'light' : 'dark');
}
(function() {
  if (localStorage.getItem('gollama-theme') === 'light') { document.body.classList.add('light'); document.documentElement.classList.add('light'); var meta = document.querySelector('meta[name="theme-color"]'); if (meta) meta.content = '#f3f4f6'; document.getElementById('themeToggle').innerHTML = '<span>☀️</span><span class="label">Theme</span>'; }
})();

// ── Settings ────────────────────────────────────────────

function toggleEditSettings(key) {
  document.getElementById('settings-' + key).classList.toggle('editing');
}

function cancelEditSettings(key) {
  document.getElementById('settings-' + key).classList.remove('editing');
  loadSettings();
}

function renderReadOnlyFlags(container, flags) {
  if (!flags || !flags.length) {
    container.innerHTML = '<div style="font-size:12px;color:var(--text-dim)">No custom flags set.</div>';
    return;
  }
  var html = '<div style="font-size:12px;font-family:var(--font-mono);display:flex;flex-wrap:wrap;gap:4px;align-items:center">';
  for (var i = 0; i < flags.length; i++) {
    var flag = flags[i];
    html += '<span class="tag">' + escHtml(flag) + '</span>';
    if (i + 1 < flags.length && !standaloneFlags[flag] && !flags[i + 1].startsWith('--')) {
      i++;
      html += '<span class="tag" style="color:var(--text-dim)">' + escHtml(flags[i]) + '</span>';
    }
  }
  html += '</div>';
  container.innerHTML = html;
}

function renderReadOnlyProfiles(profiles) {
  var textProfiles = [], imageProfiles = [];
  for (var name in profiles) {
    var p = profiles[name];
    p._name = name;
    if (p.type === 'image' || (p.type !== 'text' && (p.steps !== undefined || p.size))) { imageProfiles.push(p); } else { textProfiles.push(p); }
  }
  renderReadOnlyProfileList('profiles-readonly', textProfiles, 'No model profiles configured.', function(p) {
    var h = '<div style="font-weight:600;font-size:13px;margin-bottom:2px">' + escHtml(p._name) + '</div>';
    if (p.model) h += '<div style="font-size:11px;color:var(--text-dim);margin-bottom:2px">Model: <code style="font-size:11px">' + escHtml(p.model) + '</code></div>';
    if (p.binary_path) h += '<div style="font-size:11px;color:var(--text-dim);margin-bottom:2px">Binary: <code style="font-size:11px">' + escHtml(p.binary_path) + '</code></div>';
    if (p.description) h += '<div style="font-size:11px;color:var(--text-dim)">' + escHtml(p.description) + '</div>';
    if (p.flags && p.flags.length) {
      h += '<div style="display:flex;flex-wrap:wrap;gap:4px;margin-top:4px">';
      for (var i = 0; i < p.flags.length; i++) {
        h += '<span class="tag" style="font-family:var(--font-mono);font-size:10px">' + escHtml(p.flags[i]) + '</span>';
        if (i + 1 < p.flags.length && !standaloneFlags[p.flags[i]] && !p.flags[i + 1].startsWith('--')) {
          i++;
          h += '<span class="tag" style="font-family:var(--font-mono);font-size:10px;color:var(--text-dim)">' + escHtml(p.flags[i]) + '</span>';
        }
      }
      h += '</div>';
    }
    return h;
  });
  renderReadOnlyProfileList('image-profiles-readonly', imageProfiles, 'No image profiles configured.', function(p) {
    var h = '<div style="font-weight:600;font-size:13px;margin-bottom:2px">' + escHtml(p._name) + '</div>';
    if (p.model) h += '<div style="font-size:11px;color:var(--text-dim);margin-bottom:2px">Model: <code style="font-size:11px">' + escHtml(p.model) + '</code></div>';
    if (p.description) h += '<div style="font-size:11px;color:var(--text-dim)">' + escHtml(p.description) + '</div>';
    var parts = [];
    if (p.steps) parts.push(p.steps + ' steps');
    if (p.guidance !== undefined && p.guidance !== null) parts.push('CFG ' + p.guidance);
    if (p.size) parts.push(p.size);
    if (p.n) parts.push(p.n + '\u00d7');
    if (parts.length) h += '<div style="font-size:11px;color:var(--text-dim);margin-top:2px">' + parts.join(' \u00b7 ') + '</div>';
    return h;
  });
}

function renderReadOnlyProfileList(containerId, items, emptyMsg, renderFn) {
  var container = document.getElementById(containerId);
  if (!items.length) {
    container.innerHTML = '<div style="font-size:12px;color:var(--text-dim)">' + emptyMsg + '</div>';
    return;
  }
  var html = '<div style="display:flex;flex-direction:column;gap:8px">';
  for (var i = 0; i < items.length; i++) {
    html += '<div style="background:var(--surface-2);border-radius:6px;padding:10px 12px;border:1px solid var(--border)">' + renderFn(items[i]) + '</div>';
  }
  html += '</div>';
  container.innerHTML = html;
}

async function loadSettings() {
  try {
    var vr = await fetch('/api/v1/version'), vd = await vr.json();
    if (vd.version) { document.getElementById('s-version').textContent = vd.version; var fv = document.getElementById('faceVersion'); if (fv) fv.textContent = vd.version; }
    if (vd.llama_server) document.getElementById('s-llama-version').textContent = vd.llama_server;
    if (vd.backend) document.getElementById('s-backend').textContent = vd.backend;
  } catch (e) {}
  try {
    var r = await fetch('/api/v1/config'), cfg = await r.json();
    document.getElementById('idleTtlInput').value = cfg.idle_ttl || 0;
    renderFlags(document.getElementById('settingsFlagsContainer'), cfg.default_flags);
    renderFlags(document.getElementById('proxyFlagsContainer'), cfg.proxy_defaults);
    renderReadOnlyFlags(document.getElementById('ql-readonly'), cfg.default_flags);
    renderReadOnlyFlags(document.getElementById('api-readonly'), cfg.proxy_defaults);
    // Load model profiles into settings
    var pc = document.getElementById('profilesContainer');
    var ipc = document.getElementById('imageProfilesContainer');
    pc.innerHTML = '';
    if (ipc) ipc.innerHTML = '';
    var nextPid = 0;
    if (cfg.profiles) {
      var textIdx = 0, imageIdx = 0;
      for (var name in cfg.profiles) {
        var p = cfg.profiles[name];
        // Auto-set image defaults for profiles missing them
        if (p.type === 'image' || (p.type !== 'text' && (p.steps !== undefined || p.size))) {
          p.type = 'image';
          if (p.steps === undefined || p.steps === null) p.steps = 28;
          if (p.guidance === undefined || p.guidance === null) p.guidance = 3.5;
          if (!p.size) p.size = '1024x1024';
          if (!p.n) p.n = 1;
        }
        var profileData = { name: name, model: p.model || '', binary_path: p.binary_path || '', desc: p.description || '', flags: p.flags, strip_reasoning: p.strip_reasoning, merge_reasoning: p.merge_reasoning, env: p.env, type: p.type, steps: p.steps, guidance: p.guidance, size: p.size, n: p.n };
        if (p.type === 'image') {
          renderProfile(profileData, 'img-' + imageIdx);
          imageIdx++;
        } else {
          renderProfile(profileData, textIdx);
          textIdx++;
        }
        nextPid++;
      }
    }
    _pid = nextPid;
    renderReadOnlyProfiles(cfg.profiles);
  } catch (e) {}
}

function addSettingsFlag() {
  document.getElementById('settingsFlagsContainer').appendChild(makeFlagRow('', ''));
}

function addProxyFlag() {
  document.getElementById('proxyFlagsContainer').appendChild(makeFlagRow('', ''));
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

async function saveSettingsFlags() {
  var st = document.getElementById('settingsFlagsStatus');
  try {
    var r = await fetch('/api/v1/config', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ default_flags: collectFlags(document.getElementById('settingsFlagsContainer')) }) });
    if (r.ok) { st.textContent = 'Saved'; setTimeout(function() { st.textContent = ''; }, 2000); loadSettings(); document.getElementById('settings-ql').classList.remove('editing'); }
    else { st.textContent = 'Error saving'; }
  } catch (e) { st.textContent = 'Error: ' + e.message; }
}

async function saveProxyFlags() {
  var st = document.getElementById('proxyFlagsStatus');
  try {
    var r = await fetch('/api/v1/config', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ proxy_defaults: collectFlags(document.getElementById('proxyFlagsContainer')) }) });
    if (r.ok) { st.textContent = 'Saved'; setTimeout(function() { st.textContent = ''; }, 2000); loadSettings(); document.getElementById('settings-api').classList.remove('editing'); }
    else { st.textContent = 'Error saving'; }
  } catch (e) { st.textContent = 'Error: ' + e.message; }
}

// ── Model Profiles ─────────────────────────────────
var _pid = 0;
function addProfile(profileType) {
  profileType = profileType || 'text';
  var containerId = profileType === 'image' ? 'imageProfilesContainer' : 'profilesContainer';
  var c = document.getElementById(containerId);
  var i = _pid++;
  var d = document.createElement('div');
  d.style = 'background:var(--surface-2);border-radius:8px;padding:14px;margin-bottom:12px;border:1px solid var(--border)';
  d.innerHTML =
    '<div style="display:flex;gap:8px;align-items:center;margin-bottom:10px">' +
      '<span style="font-size:15px">📋</span>' +
      '<input type="text" placeholder="Model profile name" style="flex:1;font-weight:600;font-size:14px;min-width:120px;padding:8px 10px;background:var(--bg);border:1px solid var(--border);border-radius:var(--radius-sm);color:var(--text);outline:none;transition:border-color var(--transition);font-family:var(--font)" id="pn-' + i + '">' +
      '<button class="small danger" onclick="document.getElementById(\'pc-' + i + '\').remove()" title="Remove profile">\u2715</button>' +
    '</div>' +
    '<div style="display:flex;gap:8px;margin-bottom:8px;flex-wrap:wrap">' +
      '<div style="flex:1;min-width:140px"><label style="font-size:11px;color:var(--text-dim);display:block;margin-bottom:2px">Model (optional)</label><input type="text" class="flag-custom" placeholder="e.g. qwen3-coder-next" style="width:100%" id="pm-' + i + '"></div>' +
      '<div style="flex:1;min-width:140px"><label style="font-size:11px;color:var(--text-dim);display:block;margin-bottom:2px">Binary Path (optional)</label><input type="text" class="flag-custom" placeholder="e.g. /root/.gollama/bin/llama-server" style="width:100%" id="pb-' + i + '"></div>' +
      '<div style="flex:1;min-width:100px"><label style="font-size:11px;color:var(--text-dim);display:block;margin-bottom:2px">Type</label><select style="width:100%;padding:6px 8px;background:var(--bg);border:1px solid var(--border);border-radius:var(--radius-sm);color:var(--text);font-size:12px" id="pt-' + i + '"><option value="text"' + (profileType==='text'?' selected':'') + '>Text</option><option value="image"' + (profileType==='image'?' selected':'') + '>Image</option></select></div>' +
      '<div style="flex:2;min-width:180px"><label style="font-size:11px;color:var(--text-dim);display:block;margin-bottom:2px">Description</label><input type="text" class="flag-custom" placeholder="What is this profile for?" style="width:100%" id="pd-' + i + '"></div>' +
    '</div>' +
    '<div id="pimg-' + i + '" style="display:' + (profileType==='image' ? 'block' : 'none') + '">' +
      '<div style="display:flex;gap:8px;margin-bottom:8px;flex-wrap:wrap">' +
        '<div style="flex:0 0 60px"><label style="font-size:11px;color:var(--text-dim);display:block;margin-bottom:2px">Steps</label><input type="number" class="flag-custom" placeholder="28" style="width:100%" id="psteps-' + i + '"></div>' +
        '<div style="flex:0 0 70px"><label style="font-size:11px;color:var(--text-dim);display:block;margin-bottom:2px">Guidance</label><input type="number" class="flag-custom" placeholder="3.5" step="0.5" style="width:100%" id="pguidance-' + i + '"></div>' +
        '<div style="flex:0 0 150px"><label style="font-size:11px;color:var(--text-dim);display:block;margin-bottom:2px">Size</label><select style="width:100%;padding:6px 8px;background:var(--bg);border:1px solid var(--border);border-radius:var(--radius-sm);color:var(--text);font-size:12px" id="psize-' + i + '"><option value="">Default</option><option value="512x512">512×512</option><option value="768x768">768×768</option><option value="1024x1024">1024×1024</option><option value="1280x720">1280×720</option><option value="1024x768">1024×768</option><option value="__custom__">Custom…</option></select><input type="text" id="psize-custom-' + i + '" placeholder="WIDTHxHEIGHT" style="width:100%;display:none;margin-top:2px;padding:6px 8px;background:var(--bg);border:1px solid var(--border);border-radius:var(--radius-sm);color:var(--text);font-size:12px"></div>' +
        '<div style="flex:0 0 60px"><label style="font-size:11px;color:var(--text-dim);display:block;margin-bottom:2px">N</label><input type="number" class="flag-custom" placeholder="1" min="1" max="8" style="width:100%" id="pncount-' + i + '"></div>' +
      '</div>' +
    '</div>' +
    '<div id="pflags-section-' + i + '" style="display:' + (profileType==='image' ? 'none' : 'block') + '">' +
      '<div style="font-size:11px;color:var(--text-dim);margin-bottom:6px">Flags</div>' +
      '<div style="display:flex;gap:4px;flex-wrap:wrap;margin-bottom:6px" id="pf-' + i + '"></div>' +
      '<button class="ghost small" onclick="addProfileFlag(\'' + i + '\')" style="font-size:11px">＋ Add Flag</button>' +
    '</div>' +
    '<div style="font-size:11px;color:var(--text-dim);margin-top:8px;margin-bottom:6px">Environment Variables</div>' +
    '<div style="display:flex;flex-direction:column;gap:4px;margin-bottom:4px" id="pe-' + i + '"></div>' +
    '<button class="ghost small" onclick="addProfileEnv(\'' + i + '\')" style="font-size:11px">＋ Add Env Var</button>';
  d.id = 'pc-' + i;
  c.appendChild(d);
  document.getElementById('pt-' + i).addEventListener('change', function() {
    var isImage = this.value === 'image';
    document.getElementById('pimg-' + i).style.display = isImage ? 'block' : 'none';
    document.getElementById('pflags-section-' + i).style.display = isImage ? 'none' : 'block';
  });
  document.getElementById('psize-' + i).addEventListener('change', function() {
    document.getElementById('psize-custom-' + i).style.display = this.value === '__custom__' ? 'block' : 'none';
  });
}

function addProfileFlag(idx) {
  var fc = document.getElementById('pf-' + idx);
  fc.appendChild(makeFlagRow('', ''));
}

function addProfileEnv(idx) {
  addProfileEnvRow(idx, '', '');
}

function addProfileEnvRow(idx, key, val) {
  var ec = document.getElementById('pe-' + idx);
  var row = document.createElement('div');
  row.style = 'display:flex;gap:4px;align-items:center';
  row.innerHTML =
    '<input type="text" class="flag-custom" placeholder="KEY" style="width:160px;font-family:monospace;font-size:12px" value="' + escAttr(key) + '">' +
    '<span style="color:var(--text-dim)">=</span>' +
    '<input type="text" class="flag-custom" placeholder="value" style="flex:1;font-family:monospace;font-size:12px" value="' + escAttr(val) + '">' +
    '<button class="small ghost" onclick="this.parentElement.remove()" title="Remove" style="font-size:10px;padding:2px 6px">\u2715</button>';
  ec.appendChild(row);
}

function renderProfile(p, i) {
  var containerId = (p.type === 'image') ? 'imageProfilesContainer' : 'profilesContainer';
  var c = document.getElementById(containerId);
  var d = document.createElement('div');
  d.id = 'pc-' + i;
  d.style = 'background:var(--surface-2);border-radius:8px;padding:14px;margin-bottom:12px;border:1px solid var(--border)';
  var ptype = p.type || 'text';
  var isImage = ptype === 'image';
  d.innerHTML =
    '<div style="display:flex;gap:8px;align-items:center;margin-bottom:10px">' +
      '<span style="font-size:15px">📋</span>' +
      '<input type="text" placeholder="Model profile name" style="flex:1;font-weight:600;font-size:14px;min-width:120px;padding:8px 10px;background:var(--bg);border:1px solid var(--border);border-radius:var(--radius-sm);color:var(--text);outline:none;transition:border-color var(--transition);font-family:var(--font)" id="pn-' + i + '" value="' + escAttr(p.name) + '">' +
      '<button class="small danger" onclick="document.getElementById(\'pc-' + i + '\').remove()" title="Remove profile">\u2715</button>' +
    '</div>' +
    '<div style="display:flex;gap:8px;margin-bottom:8px;flex-wrap:wrap">' +
      '<div style="flex:1;min-width:140px"><label style="font-size:11px;color:var(--text-dim);display:block;margin-bottom:2px">Model (optional)</label><input type="text" class="flag-custom" placeholder="e.g. qwen3-coder-next" style="width:100%" id="pm-' + i + '" value="' + escAttr(p.model || '') + '"></div>' +
      '<div style="flex:1;min-width:140px"><label style="font-size:11px;color:var(--text-dim);display:block;margin-bottom:2px">Binary Path (optional)</label><input type="text" class="flag-custom" placeholder="e.g. /root/.gollama/bin/llama-server" style="width:100%" id="pb-' + i + '" value="' + escAttr(p.binary_path || '') + '"></div>' +
      '<div style="flex:1;min-width:100px"><label style="font-size:11px;color:var(--text-dim);display:block;margin-bottom:2px">Type</label><select style="width:100%;padding:6px 8px;background:var(--bg);border:1px solid var(--border);border-radius:var(--radius-sm);color:var(--text);font-size:12px" id="pt-' + i + '"><option value="text"' + (ptype==='text'?' selected':'') + '>Text</option><option value="image"' + (isImage?' selected':'') + '>Image</option></select></div>' +
      '<div style="flex:2;min-width:180px"><label style="font-size:11px;color:var(--text-dim);display:block;margin-bottom:2px">Description</label><input type="text" class="flag-custom" placeholder="What is this profile for?" style="width:100%" id="pd-' + i + '" value="' + escAttr(p.desc || '') + '"></div>' +
      '<div style="flex:0 0 auto;display:flex;align-items:end;padding-bottom:2px"><label style="font-size:11px;color:var(--text-dim);display:flex;align-items:center;gap:4px;white-space:nowrap;cursor:pointer"><input type="checkbox" id="ps-' + i + '" ' + (p.strip_reasoning ? 'checked' : '') + ' style="accent-color:var(--accent)"> Strip reasoning</label></div>' +
      '<div style="flex:0 0 auto;display:flex;align-items:end;padding-bottom:2px"><label style="font-size:11px;color:var(--text-dim);display:flex;align-items:center;gap:4px;white-space:nowrap;cursor:pointer"><input type="checkbox" id="pmr-' + i + '" ' + (p.merge_reasoning ? 'checked' : '') + ' style="accent-color:var(--accent)"> Merge reasoning</label></div>' +
    '</div>' +
    '<div id="pimg-' + i + '" style="display:' + (isImage ? 'block' : 'none') + '">' +
      '<div style="display:flex;gap:8px;margin-bottom:8px;flex-wrap:wrap">' +
        '<div style="flex:0 0 60px"><label style="font-size:11px;color:var(--text-dim);display:block;margin-bottom:2px">Steps</label><input type="number" class="flag-custom" placeholder="28" style="width:100%" id="psteps-' + i + '" value="' + (p.steps || '') + '"></div>' +
        '<div style="flex:0 0 70px"><label style="font-size:11px;color:var(--text-dim);display:block;margin-bottom:2px">Guidance</label><input type="number" class="flag-custom" placeholder="3.5" step="0.5" style="width:100%" id="pguidance-' + i + '" value="' + (p.guidance !== undefined && p.guidance !== null ? p.guidance : '') + '"></div>' +
        '<div style="flex:0 0 150px"><label style="font-size:11px;color:var(--text-dim);display:block;margin-bottom:2px">Size</label><select style="width:100%;padding:6px 8px;background:var(--bg);border:1px solid var(--border);border-radius:var(--radius-sm);color:var(--text);font-size:12px" id="psize-' + i + '"><option value="">Default</option><option value="512x512">512×512</option><option value="768x768">768×768</option><option value="1024x1024">1024×1024</option><option value="1280x720">1280×720</option><option value="1024x768">1024×768</option><option value="__custom__">Custom…</option></select><input type="text" id="psize-custom-' + i + '" placeholder="WIDTHxHEIGHT" style="width:100%;display:none;margin-top:2px;padding:6px 8px;background:var(--bg);border:1px solid var(--border);border-radius:var(--radius-sm);color:var(--text);font-size:12px"></div>' +
        '<div style="flex:0 0 60px"><label style="font-size:11px;color:var(--text-dim);display:block;margin-bottom:2px">N</label><input type="number" class="flag-custom" placeholder="1" min="1" max="8" style="width:100%" id="pncount-' + i + '" value="' + (p.n || '') + '"></div>' +
      '</div>' +
    '</div>' +
    '<div id="pflags-section-' + i + '" style="display:' + (isImage ? 'none' : 'block') + '">' +
      '<div style="font-size:11px;color:var(--text-dim);margin-bottom:6px">Flags</div>' +
      '<div style="display:flex;gap:4px;flex-wrap:wrap;margin-bottom:6px" id="pf-' + i + '"></div>' +
      '<button class="ghost small" onclick="addProfileFlag(\'' + i + '\')" style="font-size:11px">＋ Add Flag</button>' +
    '</div>' +
    '<div style="font-size:11px;color:var(--text-dim);margin-top:8px;margin-bottom:6px">Environment Variables</div>' +
    '<div style="display:flex;flex-direction:column;gap:4px;margin-bottom:4px" id="pe-' + i + '"></div>' +
    '<button class="ghost small" onclick="addProfileEnv(\'' + i + '\')" style="font-size:11px">＋ Add Env Var</button>';
  c.appendChild(d);
  var fc = document.getElementById('pf-' + i);
  if (p.flags) {
    for (var fi = 0; fi < p.flags.length; fi++) {
      var f = p.flags[fi];
      var isStandalone = standaloneFlags[f];
      var val = (!isStandalone && fi + 1 < p.flags.length) ? p.flags[fi + 1] : '';
      fc.appendChild(makeFlagRow(f, val));
      if (!isStandalone && val) fi++;
    }
  }
  var ec = document.getElementById('pe-' + i);
  if (p.env) {
    for (var ek in p.env) {
      addProfileEnvRow(i, ek, p.env[ek]);
    }
  }
  // Toggle image-specific and flags sections when type changes
  document.getElementById('pt-' + i).addEventListener('change', function() {
    var isImage = this.value === 'image';
    document.getElementById('pimg-' + i).style.display = isImage ? 'block' : 'none';
    document.getElementById('pflags-section-' + i).style.display = isImage ? 'none' : 'block';
  });
  // Set size dropdown value and show custom input if needed
  var sizeSel = document.getElementById('psize-' + i);
  if (p.size) {
    var opt = sizeSel.querySelector('option[value="' + p.size + '"]');
    if (opt) {
      sizeSel.value = p.size;
    } else {
      sizeSel.value = '__custom__';
      document.getElementById('psize-custom-' + i).value = p.size;
      document.getElementById('psize-custom-' + i).style.display = 'block';
    }
  }
  sizeSel.addEventListener('change', function() {
    document.getElementById('psize-custom-' + i).style.display = this.value === '__custom__' ? 'block' : 'none';
  });
}

function collectProfilesFromContainer(containerId) {
  var c = document.getElementById(containerId);
  if (!c) return {};
  var profiles = {};
  for (var ci = 0; ci < c.children.length; ci++) {
    var card = c.children[ci];
    if (!card.id) continue;
    var idx = card.id.replace('pc-', '');
    var name = document.getElementById('pn-' + idx);
    var model = document.getElementById('pm-' + idx);
    var desc = document.getElementById('pd-' + idx);
    var stripEl = document.getElementById('ps-' + idx);
    if (!name || !name.value) continue;
    var typeEl = document.getElementById('pt-' + idx);
    var isImage = typeEl && typeEl.value === 'image';
    var flags = [];
    if (!isImage) {
      var fc = document.getElementById('pf-' + idx);
      if (fc) {
        for (var j = 0; j < fc.children.length; j++) {
          var searchInput = fc.children[j].querySelector('.flag-search');
          var custom = fc.children[j].querySelector('.flag-custom');
          var valInput = fc.children[j].querySelector('.flag-value');
          var fn = (searchInput ? searchInput.value.trim() : '') || (custom ? custom.value : '');
          var fv = valInput ? valInput.value : '';
          if (fn) { flags.push(fn); if (!standaloneFlags[fn] && fv) flags.push(fv); }
        }
      }
    }
    var profileObj = { model: model ? model.value : '', description: desc ? desc.value : '', flags: flags };
    var bpEl = document.getElementById('pb-' + idx);
    if (bpEl && bpEl.value) profileObj.binary_path = bpEl.value;
    if (isImage) profileObj.type = 'image';
    if (stripEl && stripEl.checked) profileObj.strip_reasoning = true;
    var mergeEl = document.getElementById('pmr-' + idx);
    if (mergeEl && mergeEl.checked) profileObj.merge_reasoning = true;
    var stepsEl = document.getElementById('psteps-' + idx);
    if (stepsEl && stepsEl.value) profileObj.steps = parseInt(stepsEl.value);
    var guidanceEl = document.getElementById('pguidance-' + idx);
    if (guidanceEl && guidanceEl.value) profileObj.guidance = parseFloat(guidanceEl.value);
    var sizeEl = document.getElementById('psize-' + idx);
    if (sizeEl && sizeEl.value) {
      if (sizeEl.value === '__custom__') {
        var customSize = document.getElementById('psize-custom-' + idx);
        if (customSize && customSize.value) profileObj.size = customSize.value;
      } else {
        profileObj.size = sizeEl.value;
      }
    }
    var ncountEl = document.getElementById('pncount-' + idx);
    if (ncountEl && ncountEl.value) profileObj.n = parseInt(ncountEl.value);
    var envObj = {};
    var ec = document.getElementById('pe-' + idx);
    if (ec) {
      for (var j = 0; j < ec.children.length; j++) {
        var inputs = ec.children[j].querySelectorAll('input');
        if (inputs.length >= 2 && inputs[0].value) {
          envObj[inputs[0].value] = inputs[1].value;
        }
      }
    }
    if (Object.keys(envObj).length > 0) profileObj.env = envObj;
    profiles[name.value] = profileObj;
  }
  return profiles;
}

async function saveProfiles() {
  var textProfiles = collectProfilesFromContainer('profilesContainer');
  var imageProfiles = collectProfilesFromContainer('imageProfilesContainer');
  var profiles = Object.assign({}, textProfiles, imageProfiles);
  var st = document.getElementById('profilesStatus');
  try {
    var r = await fetch('/api/v1/config', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ profiles: profiles }) });
    if (r.ok) { st.textContent = 'Saved'; setTimeout(function() { st.textContent = ''; }, 2000); loadSettings(); document.getElementById('settings-profiles').classList.remove('editing'); document.getElementById('settings-image-profiles').classList.remove('editing'); }
    else { st.textContent = 'Error saving'; }
  } catch (e) { st.textContent = 'Error: ' + e.message; }
}

async function restartGollama() {
  if (!confirm('Restart gollama? The web UI will be unavailable for a few seconds.')) return;
  try {
    await fetch('/api/v1/restart', { method: 'POST' });
    setTimeout(function() { location.reload(); }, 2000);
  } catch (e) { alert('Restart failed: ' + e.message); }
}

// ── Image Generation ──────────────────────────────────
var _cachedImageProfiles = {};

async function loadImageProfiles() {
  var sel = document.getElementById('imageProfileSelect');
  if (!sel) return;
  try {
    var r = await fetch('/api/v1/config'), cfg = await r.json();
    _cachedImageProfiles = {};
    for (var name in cfg.profiles) {
      if (cfg.profiles[name].type === 'image') {
        _cachedImageProfiles[name] = cfg.profiles[name];
      }
    }
    sel.innerHTML = '';
    var names = Object.keys(_cachedImageProfiles);
    if (names.length === 0) {
      sel.innerHTML = '<option value="">No image profiles configured</option>';
      sel.disabled = true;
    } else {
      sel.disabled = false;
      sel.innerHTML = '<option value="">Select profile…</option>';
      names.forEach(function(name) {
        var p = _cachedImageProfiles[name];
        var desc = p.description || p.model || name;
        sel.innerHTML += '<option value="' + escAttr(name) + '">' + escHtml(name) + (desc ? ' — ' + escHtml(desc.slice(0, 40)) : '') + '</option>';
      });
      if (imageHistory.length > 0) {
        var last = imageHistory[imageHistory.length - 1];
        if (last.profile) sel.value = last.profile;
      }
    }
    onImageProfileChange();
    renderImageHistory();
    loadImageModels();
  } catch (e) {
    sel.innerHTML = '<option value="">Error loading profiles</option>';
  }
}

function onImageProfileChange() {
  var sel = document.getElementById('imageProfileSelect');
  var name = sel ? sel.value : '';
  var p = name ? _cachedImageProfiles[name] : null;
  var sizeSel = document.getElementById('imageSizeSelect');
  if (p) {
    document.getElementById('imageStepsInput').value = p.steps || '';
    document.getElementById('imageGuidanceInput').value = (p.guidance !== undefined && p.guidance !== null) ? p.guidance : '';
    document.getElementById('imageNInput').value = p.n || '';
    if (p.size && sizeSel) {
      var opt = sizeSel.querySelector('option[value="' + p.size + '"]');
      if (opt) {
        sizeSel.value = p.size;
        document.getElementById('imageSizeCustom').style.display = 'none';
      } else {
        sizeSel.value = 'custom';
        document.getElementById('imageSizeCustom').value = p.size;
        document.getElementById('imageSizeCustom').style.display = 'block';
      }
    } else {
      if (sizeSel) sizeSel.value = '';
    }
  } else {
    document.getElementById('imageStepsInput').value = '';
    document.getElementById('imageGuidanceInput').value = '';
    document.getElementById('imageNInput').value = '';
    if (sizeSel) sizeSel.value = '';
  }
}

async function loadImageModels() {
  var list = document.getElementById('imgModelList');
  if (!list) return;
  try {
    var r = await fetch('/api/v1/image-models'), models = await r.json();
    if (!models || !models.length) { list.style.display = 'none'; return; }
    list.style.display = 'block';
    list.innerHTML = models.map(function(m) {
      var status = m.cached
        ? '<span class="badge badge-green" style="font-size:9px">cached ' + (m.size ? fmtSize(m.size) : '') + '</span>'
        : '<span class="badge" style="background:var(--surface-2);color:var(--text-dim);font-size:9px">not cached</span>';
      return '<div style="font-size:11px;padding:6px 0;border-bottom:1px solid var(--border);display:flex;justify-content:space-between;align-items:center">' +
        '<div style="min-width:0"><div style="font-weight:500;white-space:nowrap;overflow:hidden;text-overflow:ellipsis">' + escHtml(m.name) + '</div><div style="font-size:10px;color:var(--text-dim);white-space:nowrap;overflow:hidden;text-overflow:ellipsis">' + escHtml(m.model_id) + '</div></div>' +
        status +
        '</div>';
    }).join('');
  } catch (e) { list.style.display = 'none'; }
}

var _imgModelSearchTimer = null;

function toggleImageModelSearch() {
  var wrap = document.getElementById('imgModelSearchWrap');
  if (wrap.style.display === 'block') { wrap.style.display = 'none'; return; }
  wrap.style.display = 'block';
  document.getElementById('imgModelSearchInput').focus();
}

function onImageModelSearch(query) {
  if (_imgModelSearchTimer) clearTimeout(_imgModelSearchTimer);
  var results = document.getElementById('imgModelSearchResults');
  if (!query || query.length < 2) { results.style.display = 'none'; return; }
  results.innerHTML = '<div style="padding:10px;text-align:center;color:var(--text-dim)"><span class="spinner"></span> Searching…</div>';
  results.style.display = 'block';
  _imgModelSearchTimer = setTimeout(async function() {
    try {
      var r = await fetch('/api/v1/image-models/search?q=' + encodeURIComponent(query));
      if (!r.ok) { results.innerHTML = '<div style="padding:10px;text-align:center;color:var(--text-dim)">Search failed</div>'; return; }
      var items = await r.json();
      if (!items || !items.length) { results.innerHTML = '<div style="padding:10px;text-align:center;color:var(--text-dim)">No models found</div>'; return; }
      results.innerHTML = items.map(function(m) {
        var gated = m.gated ? ' <span class="badge" style="background:var(--amber-bg);color:var(--amber);font-size:9px">gated</span>' : '';
        var size = m.size ? ' · ' + fmtSize(m.size) : '';
        var tag = m.pipeline_tag === 'text-to-image' ? '🖼️' : '🎨';
        return '<div style="padding:8px 10px;cursor:pointer;border-bottom:1px solid var(--border);transition:background var(--transition)" onmouseover="this.style.background=\'var(--surface-2)\'" onmouseout="this.style.background=\'transparent\'">' +
          '<div style="display:flex;justify-content:space-between;align-items:center">' +
          '<div style="min-width:0;flex:1">' +
          '<div style="font-size:12px;font-weight:500;white-space:nowrap;overflow:hidden;text-overflow:ellipsis">' + tag + ' ' + escHtml(m.id) + gated + '</div>' +
          '<div style="font-size:10px;color:var(--text-dim);margin-top:1px">⭐ ' + m.likes + ' · ⬇ ' + m.downloads + size + '</div>' +
          '</div>' +
          '<button class="small primary" onclick="event.stopPropagation();installImageModel(\'' + escAttr(m.id) + '\')" style="font-size:10px;white-space:nowrap;flex-shrink:0">Add</button>' +
          '</div>' +
          '</div>';
      }).join('');
    } catch (e) { results.innerHTML = '<div style="padding:10px;text-align:center;color:var(--text-dim)">Error: ' + escHtml(e.message) + '</div>'; }
  }, 300);
}

async function installImageModel(modelId) {
  var name = prompt('Profile name for "' + modelId.split('/').pop() + '":', modelId.split('/').pop().replace(/[^a-zA-Z0-9_-]/g, '').slice(0, 30));
  if (!name) return;
  try {
    var r = await fetch('/api/v1/image-models/install', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name: name, model_id: modelId }) });
    if (!r.ok) { var e = await r.text(); try { var j = JSON.parse(e); alert(j.error || e); } catch (x) { alert(e); } return; }
    document.getElementById('imgModelSearchWrap').style.display = 'none';
    document.getElementById('imgModelSearchResults').style.display = 'none';
    loadImageProfiles();
    alert('Profile "' + name + '" added for ' + modelId + '. Select it from the Model dropdown and generate!');
  } catch (e) { alert('Error: ' + e.message); }
}

async function generateImage() {
  if (imageGenerating) return;
  var prompt = document.getElementById('imagePromptInput').value.trim();
  if (!prompt) { alert('Enter a prompt first.'); return; }
  var profile = document.getElementById('imageProfileSelect').value;
  if (!profile) { alert('Select an image profile first.'); return; }

  var pr = _cachedImageProfiles[profile] || {};
  var n = parseInt(document.getElementById('imageNInput').value) || pr.n || 1;
  var sizeEl = document.getElementById('imageSizeSelect');
  var size = sizeEl.value === 'custom' ? document.getElementById('imageSizeCustom').value.trim() : (sizeEl.value || pr.size || '1024x1024');
  var steps = parseInt(document.getElementById('imageStepsInput').value) || pr.steps || 4;
  var guidance = parseFloat(document.getElementById('imageGuidanceInput').value);
  if (!guidance && pr.guidance !== undefined && pr.guidance !== null) guidance = pr.guidance;
  var seedInput = document.getElementById('imageSeedInput').value.trim();
  var seed = seedInput ? parseInt(seedInput) || undefined : undefined;

  var params = { n: n, size: size, steps: steps, guidance: guidance, seed: seed };

  imageGenerating = true;
  var btn = document.getElementById('imageGenBtn');
  btn.disabled = true;
  btn.innerHTML = '<span class="image-spinner"></span> Generating…';
  var resultsArea = document.getElementById('imageResultsArea');
  var empty = document.getElementById('imageEmpty');
  if (empty) empty.style.display = 'none';

  var loadingCard = document.createElement('div');
  loadingCard.className = 'image-loading-card';
  loadingCard.innerHTML = '<div class="spinner"></div><div>Generating image…</div>';
  resultsArea.insertBefore(loadingCard, resultsArea.firstChild);

  var body = { prompt: prompt, profile: profile, n: n, size: size, steps: steps };
  if (guidance > 0) body.guidance = guidance;
  if (seed !== undefined) body.seed = seed;
  try {
    var r = await fetch('/v1/images/generations', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
    var retries = 0;
    while (r.status == 503 && retries < 20) {
      var retryAfter = parseInt(r.headers.get('Retry-After')) || 5;
      loadingCard.innerHTML = '<div class="spinner"></div><div>Loading model… (attempt ' + (retries + 1) + ')</div>';
      await new Promise(function(res) { setTimeout(res, retryAfter * 1000); });
      r = await fetch('/v1/images/generations', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
      retries++;
    }
    loadingCard.remove();
    if (!r.ok) {
      var errText = await r.text();
      var errMsg = errText;
      try { var j = JSON.parse(errText); errMsg = j.error || errText; } catch (e) {}
      var errCard = document.createElement('div');
      errCard.className = 'image-result-card';
      errCard.innerHTML = '<div style="padding:20px;text-align:center;color:var(--red)"><div style="font-size:24px;margin-bottom:8px">⚠️</div><div style="font-weight:600;margin-bottom:4px">Generation Failed</div><div style="font-size:12px;color:var(--text-dim);white-space:pre-wrap">' + escHtml(errMsg) + '</div></div>';
      resultsArea.insertBefore(errCard, resultsArea.firstChild);
      return;
    }
    var data = await r.json();
    var items = data.data || [];
    if (!items.length) {
      var emptyCard = document.createElement('div');
      emptyCard.className = 'image-result-card';
      emptyCard.innerHTML = '<div style="padding:20px;text-align:center;color:var(--text-dim)">No images returned</div>';
      resultsArea.insertBefore(emptyCard, resultsArea.firstChild);
      return;
    }
    items.forEach(function(item) {
      var b64 = item.b64_json || item.url || '';
      if (b64 && item.b64_json && b64.indexOf('data:') !== 0) {
        b64 = 'data:image/png;base64,' + b64;
      }
      var timestamp = new Date().toLocaleString();
      var idx = imageHistory.length;
      var entry = { prompt: prompt, profile: profile, data: b64, time: timestamp, params: params, seed: seed };
      imageHistory.push(entry);
      try { localStorage.setItem('gollama_image_history', JSON.stringify(imageHistory.slice(-20))); } catch (e) {}
      renderImageResultCard(resultsArea, idx, false);
    });
    renderImageHistory();
  } catch (e) {
    loadingCard.remove();
    var errCard = document.createElement('div');
    errCard.className = 'image-result-card';
    errCard.innerHTML = '<div style="padding:20px;text-align:center;color:var(--red)"><div style="font-size:24px;margin-bottom:8px">⚠️</div><div style="font-weight:600;margin-bottom:4px">Request Failed</div><div style="font-size:12px;color:var(--text-dim)">' + escHtml(e.message) + '</div></div>';
    resultsArea.insertBefore(errCard, resultsArea.firstChild);
  } finally {
    imageGenerating = false;
    btn.disabled = false;
    btn.textContent = 'Generate';
  }
}

function renderImageResultCard(container, idx, isHistory) {
  var h = imageHistory[idx];
  if (!h) return;
  var card = document.createElement('div');
  card.className = 'image-result-card';
  var promptShort = h.prompt.length > 80 ? h.prompt.slice(0, 80) + '…' : h.prompt;
  var removeAttr = isHistory ? 'removeHistoryImage(' + idx + ', this)' : 'removeResultCard(this)';

  var metaParts = [];
  if (h.params) {
    var p = h.params;
    if (p.n && p.n > 1) metaParts.push(p.n + '×');
    if (p.size) metaParts.push(p.size);
    if (p.steps) metaParts.push(p.steps + ' steps');
    if (p.guidance > 0) metaParts.push('CFG ' + p.guidance);
  }
  if (h.seed !== undefined && h.seed !== null) {
    metaParts.push('seed ' + h.seed);
  }
  var metaHtml = metaParts.length ? '<div style="font-size:10px;color:var(--text-dim);margin-top:6px;font-family:var(--font-mono)">' + escHtml(metaParts.join(' · ')) + '</div>' : '';

  card.innerHTML =
    '<div class="result-meta">' +
      '<div style="min-width:0">' +
        '<span class="result-prompt" title="' + escAttr(h.prompt) + '">' + escHtml(promptShort) + '</span>' +
        metaHtml +
      '</div>' +
      '<div class="result-actions">' +
        '<button class="small ghost" onclick="regenerateFromHistory(' + idx + ')" title="Re-generate">🔄</button>' +
        '<button class="small ghost" onclick="downloadImageByIndex(' + idx + ')" title="Download">⬇</button>' +
        '<button class="small danger" onclick="' + removeAttr + '" title="Remove">✕</button>' +
      '</div>' +
    '</div>';
  var img = document.createElement('img');
  img.className = 'result-img';
  img.src = h.data;
  img.alt = h.prompt;
  img.loading = 'lazy';
  var dataRef = h.data;
  img.onclick = function() { openLightbox(dataRef); };
  card.appendChild(img);
  container.insertBefore(card, container.firstChild);
}

function regenerateFromHistory(idx) {
  var h = imageHistory[idx];
  if (!h) return;
  document.getElementById('imagePromptInput').value = h.prompt;
  if (h.params) {
    var p = h.params;
    if (p.n) document.getElementById('imageNInput').value = p.n;
    if (p.size) {
      var sel = document.getElementById('imageSizeSelect');
      var opt = Array.from(sel.options).find(function(o) { return o.value === p.size; });
      if (opt) { sel.value = p.size; document.getElementById('imageSizeCustom').style.display = 'none'; }
      else { sel.value = 'custom'; document.getElementById('imageSizeCustom').value = p.size; document.getElementById('imageSizeCustom').style.display = 'block'; }
    }
    if (p.steps) document.getElementById('imageStepsInput').value = p.steps;
    if (p.guidance) document.getElementById('imageGuidanceInput').value = p.guidance;
  }
  document.getElementById('imageSeedInput').value = h.seed !== undefined && h.seed !== null ? h.seed : '';
  switchView('image');
  document.getElementById('imagePromptInput').focus();
}

function downloadImageByIndex(idx) {
  var h = imageHistory[idx];
  if (!h) return;
  var link = document.createElement('a');
  link.download = 'gollama-' + (h.time || Date.now()).replace(/[/:]/g, '-') + '.png';
  link.href = h.data;
  link.click();
}

function removeResultCard(btn) {
  var card = btn.closest('.image-result-card');
  if (card) card.remove();
}

function openLightbox(src) {
  var lb = document.createElement('div');
  lb.className = 'image-lightbox';
  lb.innerHTML = '<button class="lb-close" onclick="this.parentElement.remove()">✕</button><img src="' + src + '" alt="Full size">';
  lb.onclick = function(e) { if (e.target === lb) lb.remove(); };
  document.body.appendChild(lb);
}

function clearImageHistory() {
  if (!confirm('Clear all generation history?')) return;
  imageHistory = [];
  try { localStorage.removeItem('gollama_image_history'); } catch (e) {}
  renderImageHistory();
  var resultsArea = document.getElementById('imageResultsArea');
  resultsArea.innerHTML = '<div class="image-empty" id="imageEmpty"><div class="icon">🎨</div><div class="title">No images generated yet</div><p>Type a prompt and click Generate to create an image. Results appear here.</p></div>';
}

function renderImageHistory() {
  var section = document.getElementById('imageHistorySection');
  var grid = document.getElementById('imageHistoryGrid');
  var count = document.getElementById('imageHistoryCount');
  if (!section || !grid) return;
  if (imageHistory.length === 0) {
    section.style.display = 'none';
    return;
  }
  section.style.display = 'block';
  count.textContent = imageHistory.length + ' images';
  grid.innerHTML = imageHistory.slice().reverse().map(function(h, i) {
    var realIdx = imageHistory.length - 1 - i;
    var promptShort = h.prompt.length > 60 ? h.prompt.slice(0, 60) + '…' : h.prompt;
    return '<div class="image-thumb" onclick="showHistoryImage(' + realIdx + ')"><img src="' + escAttr(h.data) + '" loading="lazy"><div class="thumb-overlay"><div class="thumb-prompt">' + escHtml(promptShort) + '</div></div></div>';
  }).join('');
}

function showHistoryImage(idx) {
  var h = imageHistory[idx];
  if (!h) return;
  var resultsArea = document.getElementById('imageResultsArea');
  var empty = document.getElementById('imageEmpty');
  if (empty) empty.style.display = 'none';
  renderImageResultCard(resultsArea, idx, true);
}

function removeHistoryImage(idx, btn) {
  imageHistory.splice(idx, 1);
  try { localStorage.setItem('gollama_image_history', JSON.stringify(imageHistory.slice(-20))); } catch (e) {}
  var card = btn.closest('.image-result-card');
  if (card) card.remove();
  renderImageHistory();
  if (imageHistory.length === 0) {
    document.getElementById('imageResultsArea').innerHTML = '<div class="image-empty" id="imageEmpty"><div class="icon">🎨</div><div class="title">No images generated yet</div><p>Type a prompt and click Generate to create an image. Results appear here.</p></div>';
  }
}

// ── Init (staggered, no pile-up) ─────────────────────
document.addEventListener('keydown', function(e) {
  if (e.key !== 'Escape') return;
  var lb = document.querySelector('.image-lightbox'); if (lb) lb.remove();
  if (document.getElementById('logModal').style.display === 'block') closeLogs();
  if (document.getElementById('modelModal').style.display === 'block') closeModelDetails();
  if (document.getElementById('chatHistoryModal').style.display === 'block') closeChatHistory();
});
setInterval(function() {
  var c = document.getElementById('faceClock');
  if (c) c.textContent = new Date().toLocaleTimeString([], { hour12: false });
}, 1000);
loadModels();
setTimeout(loadDefaultFlags, 50);
setTimeout(loadPresets, 50);
setTimeout(loadInstances, 100);
setTimeout(loadSettings, 150);
try { var saved = localStorage.getItem('gollama_image_history'); if (saved) { imageHistory = JSON.parse(saved); } } catch (e) {}
document.getElementById('imageSizeSelect').addEventListener('change', function() {
  document.getElementById('imageSizeCustom').style.display = this.value === 'custom' ? 'block' : 'none';
});
</script>
</body>
</html>`
