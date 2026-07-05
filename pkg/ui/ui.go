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
  --bg: #08080f;
  --surface: #0f0f1a;
  --surface-2: #181828;
  --surface-3: #20203a;
  --border: #282845;
  --border-hover: #3a3a5c;
  --text: #e8e8f4;
  --text-muted: #8888b0;
  --text-dim: #5c5c88;
  --accent: #00e5bf;
  --accent-hover: #00ffd0;
  --accent-bg: rgba(0, 229, 191, 0.08);
  --accent-glow: rgba(0, 229, 191, 0.25);
  --accent-gradient: linear-gradient(135deg, #00e5bf, #00b8ff);
  --purple: #a855f7;
  --purple-bg: rgba(168, 85, 247, 0.1);
  --purple-glow: rgba(168, 85, 247, 0.2);
  --green: #00e5bf;
  --green-bg: rgba(0, 229, 191, 0.1);
  --red: #ff4060;
  --red-bg: rgba(255, 64, 96, 0.1);
  --amber: #f59e0b;
  --amber-bg: rgba(245, 158, 11, 0.1);
  --blue: #60a0ff;
  --blue-bg: rgba(96, 160, 255, 0.1);
  --sidebar-w: 220px;
  --radius: 12px;
  --radius-sm: 8px;
  --font: 'Plus Jakarta Sans', system-ui, sans-serif;
  --font-mono: 'JetBrains Mono', 'SF Mono', monospace;
  --shadow: 0 2px 8px rgba(0,0,0,.4), 0 1px 3px rgba(0,0,0,.3);
  --shadow-lg: 0 8px 32px rgba(0,0,0,.5), 0 2px 8px rgba(0,0,0,.3);
  --shadow-glow: 0 0 20px rgba(0, 229, 191, 0.08);
  --transition: 200ms cubic-bezier(0.4, 0, 0.2, 1);
}
.light {
  --bg: #f0f0f6;
  --surface: #ffffff;
  --surface-2: #f6f6fc;
  --surface-3: #eeeef6;
  --border: #d8d8e8;
  --border-hover: #b8b8d0;
  --text: #141420;
  --text-muted: #686888;
  --text-dim: #9898b8;
  --accent: #009977;
  --accent-hover: #00b388;
  --accent-bg: rgba(0, 153, 119, 0.06);
  --accent-glow: rgba(0, 153, 119, 0.12);
  --accent-gradient: linear-gradient(135deg, #009977, #0077b3);
  --purple: #7c3aed;
  --purple-bg: rgba(124, 58, 237, 0.06);
  --green: #009977;
  --green-bg: rgba(0, 153, 119, 0.08);
  --red: #cc3344;
  --red-bg: rgba(204, 51, 68, 0.06);
  --amber: #b87722;
  --amber-bg: rgba(184, 119, 34, 0.08);
  --blue: #3366cc;
  --blue-bg: rgba(51, 102, 204, 0.06);
  --shadow: 0 1px 4px rgba(0,0,0,.06), 0 1px 2px rgba(0,0,0,.04);
  --shadow-lg: 0 8px 24px rgba(0,0,0,.08), 0 2px 4px rgba(0,0,0,.04);
  --shadow-glow: 0 0 12px rgba(0, 153, 119, 0.06);
}
* { margin: 0; padding: 0; box-sizing: border-box; }
:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; border-radius: var(--radius-sm); }
html { color-scheme: dark; height: 100%; }
.light { color-scheme: light; }
::-webkit-scrollbar { width: 6px; height: 6px; }
::-webkit-scrollbar-track { background: transparent; }
::-webkit-scrollbar-thumb { background: var(--surface-3); border-radius: 3px; }
::-webkit-scrollbar-thumb:hover { background: var(--border-hover); }
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
  gap: 4px; color: var(--text); overflow: hidden; position: relative;
}
.sidebar .logo::after {
  content: ''; position: absolute; bottom: -1px; left: 20%; right: 20%;
  height: 1px; background: var(--accent-gradient); opacity: .4;
}
.sidebar .logo img { width: 100%; max-width: 72px; height: auto; flex-shrink: 0; filter: brightness(0) invert(1); clip-path: inset(2px); }
.light .sidebar .logo img { filter: none; }
.sidebar.collapsed .logo img { max-width: 26px; }
.sidebar nav { flex: 1; padding: 10px 6px; display: flex; flex-direction: column; gap: 2px; }
.sidebar .nav-item {
  display: flex; align-items: center; justify-content: flex-start; gap: 10px; padding: 10px 12px;
  border-radius: var(--radius-sm); color: var(--text-muted);
  text-decoration: none; font-size: 13px; font-weight: 600; cursor: pointer;
  transition: all var(--transition); white-space: nowrap; overflow: hidden;
  border: none; background: none; width: 100%; text-align: left;
  font-family: var(--font); position: relative;
}
.sidebar .nav-item:hover { background: var(--surface-2); color: var(--text); }
.sidebar .nav-item.active {
  background: var(--accent-bg); color: var(--accent);
  box-shadow: inset 3px 0 0 var(--accent);
}
.sidebar .nav-item .icon { font-size: 16px; width: 22px; text-align: center; flex-shrink: 0; }
.sidebar .nav-item .label { transition: opacity var(--transition); white-space: nowrap; }
.sidebar.collapsed .nav-item .label { opacity: 0; width: 0; overflow: hidden; }
.sidebar .bottom { padding: 6px; border-top: 1px solid var(--border); width: 100%; display: flex; position: relative; }
.sidebar .bottom::before {
  content: ''; position: absolute; top: -1px; left: 20%; right: 20%;
  height: 1px; background: var(--accent-gradient); opacity: .2;
}
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
  pointer-events: none; z-index: 0; opacity: 0.35;
}
.main::after {
  content: ''; position: fixed; top: -50%; left: -50%; width: 200%; height: 200%;
  background: radial-gradient(ellipse at 30% 20%, rgba(0, 229, 191, 0.03) 0%, transparent 50%),
              radial-gradient(ellipse at 70% 80%, rgba(168, 85, 247, 0.02) 0%, transparent 50%);
  pointer-events: none; z-index: 0;
}
.main > * { position: relative; z-index: 1; }
.view { display: none; animation: fadeIn 300ms cubic-bezier(0.16, 1, 0.3, 1); }
.view.active { display: block; }

@keyframes fadeIn { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }
@keyframes slideUp { from { opacity: 0; transform: translateY(12px); } to { opacity: 1; transform: translateY(0); } }
@keyframes pulse { 0%,100%{opacity:1} 50%{opacity:.4} }
@keyframes spin { to{transform:rotate(360deg)} }
@keyframes shimmer { 0%{background-position:-200% 0} 100%{background-position:200% 0} }

@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after { animation-duration: 0.01ms !important; transition-duration: 0.01ms !important; }
  .view { animation: none !important; }
}

/* ── Page Header ─────────────────────────────────────── */
.page-header { margin-bottom: 28px; }
.page-header h1 {
  font-size: 26px; font-weight: 800; letter-spacing: -.8px; text-wrap: balance;
  background: var(--accent-gradient); -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text;
}
.page-header p { color: var(--text-muted); font-size: 13px; margin-top: 4px; }

/* ── Metrics ──────────────────────────────────────────── */
.metrics { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 14px; margin-bottom: 32px; }
.metric-card {
  background: var(--surface); border: 1px solid var(--border);
  border-radius: var(--radius); padding: 18px 20px;
  transition: all var(--transition); position: relative; overflow: hidden;
  box-shadow: var(--shadow);
}
.metric-card::before {
  content: ''; position: absolute; top: 0; left: 0; right: 0; height: 2px;
  background: var(--accent-gradient); opacity: 0; transition: opacity var(--transition);
}
.metric-card:hover { border-color: var(--border-hover); box-shadow: var(--shadow-lg), var(--shadow-glow); transform: translateY(-1px); }
.metric-card:hover::before { opacity: 1; }
.metric-card .label { font-size: 11px; color: var(--text-muted); font-weight: 600; text-transform: uppercase; letter-spacing: .6px; margin-bottom: 6px; }
.metric-card .value { font-size: 28px; font-weight: 800; letter-spacing: -1px; font-variant-numeric: tabular-nums; background: var(--accent-gradient); -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text; }
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
.card { background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius); box-shadow: var(--shadow); transition: all var(--transition); }
.card:hover { border-color: var(--border-hover); box-shadow: var(--shadow-lg); }
.card-body { padding: 18px; }

/* ── Instance Card ────────────────────────────────────── */
.instance-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(380px, 1fr)); gap: 14px; }
.inst-card {
  background: linear-gradient(135deg, var(--surface), var(--surface-2)); border: 1px solid var(--border);
  border-radius: var(--radius); padding: 18px;
  transition: all var(--transition); position: relative; overflow: hidden;
  box-shadow: var(--shadow);
}
.inst-card::before {
  content: ''; position: absolute; top: 0; left: 0; width: 3px; height: 100%;
  background: var(--accent-gradient); border-radius: 0 2px 2px 0;
  transition: all var(--transition);
}
.inst-card::after {
  content: ''; position: absolute; top: 0; left: 0; right: 0; height: 1px;
  background: linear-gradient(90deg, var(--accent), transparent 60%);
  opacity: 0; transition: opacity var(--transition);
}
.inst-card:hover { border-color: var(--border-hover); box-shadow: var(--shadow-lg), var(--shadow-glow); transform: translateY(-2px); }
.inst-card:hover::after { opacity: .4; }
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
  width: 100%; padding: 10px 14px; background: var(--bg);
  border: 1px solid var(--border); border-radius: var(--radius-sm);
  color: var(--text); font-size: 13px; outline: none;
  transition: all var(--transition); font-family: var(--font);
}
select:focus, input:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-bg); }
select { cursor: pointer; appearance: none; background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' fill='%238888b0'%3E%3Cpath d='M2 4l4 4 4-4'/%3E%3C/svg%3E"); background-repeat: no-repeat; background-position: right 12px center; padding-right: 34px; }

/* ── Buttons ───────────────────────────────────────────── */
button {
  padding: 10px 20px; border-radius: var(--radius-sm); border: none;
  font-size: 13px; font-weight: 600; cursor: pointer;
  transition: all var(--transition); font-family: var(--font);
  display: inline-flex; align-items: center; justify-content: center; gap: 6px;
  white-space: nowrap; position: relative;
}
button.primary {
  background: var(--accent-gradient); color: #08080f; position: relative; overflow: hidden;
}
button.primary:hover { transform: translateY(-1px); box-shadow: 0 4px 16px var(--accent-glow); }
button.primary:active { transform: translateY(0); }
button.secondary {
  background: var(--surface-2); color: var(--text); border: 1px solid var(--border);
}
button.secondary:hover { background: var(--surface-3); border-color: var(--border-hover); }
button.danger { background: var(--red); color: #fff; }
button.danger:hover { box-shadow: 0 4px 16px rgba(255, 64, 96, 0.3); }
button.small { padding: 7px 14px; font-size: 11px; border-radius: 6px; }
button:disabled { opacity: .35; cursor: default; pointer-events: none; }
button.ghost { background: transparent; color: var(--text-muted); padding: 6px 10px; }
button.ghost:hover { background: var(--surface-2); color: var(--text); }

/* ── Badge ──────────────────────────────────────────────── */
.badge { display: inline-flex; align-items: center; padding: 3px 10px; border-radius: 9999px; font-size: 10px; font-weight: 700; letter-spacing: .3px; font-family: var(--font-mono); }
.badge-green { background: var(--green-bg); color: var(--green); border: 1px solid rgba(0, 229, 191, 0.15); }
.badge-red { background: var(--red-bg); color: var(--red); border: 1px solid rgba(255, 64, 96, 0.15); }
.badge-amber { background: var(--amber-bg); color: var(--amber); border: 1px solid rgba(245, 158, 11, 0.15); }
.badge-blue { background: var(--blue-bg); color: var(--blue); border: 1px solid rgba(96, 160, 255, 0.15); }
.badge-profile { background: var(--purple-bg); color: var(--purple); border: 1px solid rgba(168, 85, 247, 0.15); }

/* ── Advanced flags toggle ────────────────────────────── */
.advanced-flags summary { cursor: pointer; font-size: 12px; color: var(--text-muted); font-weight: 600; user-select: none; padding: 4px 0; }
.advanced-flags summary:hover { color: var(--text); }
.advanced-flags[open] summary { color: var(--accent); }
.advanced-flags[open] summary::after { content: ''; }

/* ── Flag rows ─────────────────────────────────────────── */
.flag-row { display: flex; gap: 6px; margin-bottom: 6px; align-items: center; }
.flag-row .flag-search-wrapper { position: relative; flex-shrink: 0; min-width: 160px; }
.flag-row .flag-search { width: 100%; padding: 9px 12px; background: var(--bg); border: 1px solid var(--border); border-radius: var(--radius-sm); color: var(--text); font-size: 13px; outline: none; transition: all var(--transition); font-family: var(--font); box-sizing: border-box; }
.flag-row .flag-search:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-bg); }
.flag-row .flag-search.open { border-radius: var(--radius-sm) var(--radius-sm) 0 0; }
.flag-row .flag-dropdown { position: absolute; top: 100%; left: 0; right: 0; max-height: 240px; overflow-y: auto; background: var(--surface); border: 1px solid var(--border); border-top: none; border-radius: 0 0 var(--radius-sm) var(--radius-sm); z-index: 100; box-shadow: 0 12px 32px rgba(0,0,0,.5); }
.flag-row .flag-dropdown-item { padding: 7px 12px; font-size: 12px; cursor: pointer; color: var(--text); font-family: var(--font-mono); transition: background var(--transition); }
.flag-row .flag-dropdown-item:hover, .flag-row .flag-dropdown-item.sel { background: var(--accent-bg); color: var(--accent); }
.flag-row .flag-dropdown-divider { border-top: 1px solid var(--border); margin: 0; }
.flag-row .flag-dropdown-item-custom { color: var(--text-muted); font-family: var(--font); }
.flag-row .flag-dropdown-item-custom:hover { color: var(--text); }
.flag-row .flag-dropdown-empty { color: var(--text-dim); cursor: default; font-family: var(--font); }
.flag-row input.flag-custom { flex: 1; }
.flag-row input.flag-value { flex: 1; }

/* ── Settings edit mode ───────────────────────────────── */
.settings-card .settings-readonly { display: block; }
.settings-card .settings-form { display: none; }
.settings-card.editing .settings-readonly { display: none; }
.settings-card.editing .settings-form { display: block; }
.settings-card.editing .settings-edit-btn { display: none; }

/* ── Detail rows (Settings info) ──────────────────────── */
.detail-row { display: flex; justify-content: space-between; align-items: center; padding: 6px 0; border-bottom: 1px solid var(--border); }
.detail-row:last-child { border-bottom: none; }
.detail-label { font-size: 12px; color: var(--text-muted); font-weight: 600; text-transform: uppercase; letter-spacing: .5px; display: flex; align-items: center; gap: 6px; }
.detail-value { font-size: 13px; color: var(--text); text-align: right; }

/* ── Empty state ───────────────────────────────────────── */
.empty-state { text-align: center; padding: 48px 20px; color: var(--text-dim); }
.instance-grid .empty-state { grid-column: 1 / -1; display: flex; flex-direction: column; align-items: center; justify-content: center; min-height: 200px; }
.empty-state .icon { font-size: 40px; margin-bottom: 12px; opacity: .5; }
.empty-state .title { font-size: 16px; font-weight: 600; color: var(--text-muted); margin-bottom: 4px; }
.empty-state p { font-size: 13px; }

/* ── Chat ──────────────────────────────────────────────── */
.chat-container { display: flex; flex-direction: column; height: calc(100vh - 64px); }
.chat-header { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding-bottom: 16px; margin-bottom: 16px; border-bottom: 1px solid var(--border); }
.chat-header .header-left { display: flex; align-items: center; gap: 12px; flex: 1; flex-wrap: wrap; }
.chat-header .header-left h1 { font-size: 22px; font-weight: 800; letter-spacing: -.5px; }
.chat-header select { width: auto; min-width: 220px; }
.chat-header .header-actions { display: flex; align-items: center; gap: 4px; flex-shrink: 0; }
.chat-header .header-actions .pill-btn { display: inline-flex; align-items: center; gap: 5px; padding: 5px 12px; font-size: 12px; font-weight: 500; border-radius: 9999px; border: none; cursor: pointer; transition: all var(--transition); white-space: nowrap; background: transparent; color: var(--text-muted); }
.chat-header .header-actions .pill-btn:hover { background: var(--surface-2); color: var(--text); }
.chat-header .header-actions .pill-btn.history:hover { background: var(--accent-bg); color: var(--accent); }
.chat-header .header-actions .pill-btn.clear:hover { background: var(--red-bg); color: var(--red); }
.chat-msgs { flex: 1; overflow-y: auto; padding: 6px 0; margin-bottom: 12px; }
.chat-msgs .msg { margin-bottom: 12px; padding: 12px 16px; border-radius: 12px; max-width: 80%; line-height: 1.7; font-size: 13.5px; animation: slideUp 200ms ease both; position: relative; }
.chat-msgs .user { background: var(--accent-bg); margin-left: auto; border-bottom-right-radius: 4px; color: var(--text); }
.chat-msgs .assistant { background: var(--surface); border: 1px solid var(--border); border-bottom-left-radius: 4px; padding-right: 40px; }
.chat-msgs .system { background: transparent; color: var(--text-dim); font-style: italic; font-size: 12px; text-align: center; max-width: 100%; }
.chat-msgs .assistant .copy-btn { position: absolute; top: 8px; right: 8px; font-size: 12px; background: none; border: none; cursor: pointer; opacity: 0; padding: 2px 4px; border-radius: 4px; transition: opacity var(--transition), background var(--transition); }
.chat-msgs .assistant:hover .copy-btn { opacity: 0.5; }
.chat-msgs .assistant .copy-btn:hover { opacity: 1; background: var(--surface-2); }
.chat-input-row { display: flex; flex-direction: column; gap: 8px; background: var(--surface); border-radius: var(--radius-sm); padding: 15px; border: 1px solid var(--border); transition: border-color var(--transition); }
.chat-input-row:focus-within { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-bg); }
.chat-input-row .input-wrap { display: flex; gap: 8px; align-items: flex-end; }
.chat-input-row .input-wrap textarea { flex: 1; max-height: 200px; }
.chat-loading { animation: pulse 1.2s infinite; display: inline-block; letter-spacing: 4px; font-size: 18px; line-height: 1; color: var(--text-dim); }
.reasoning { color: var(--text-dim); font-style: italic; font-size: 12px; border-left: 2px solid var(--accent); padding-left: 12px; margin-bottom: 8px; opacity: .85; }
.ctx-label { padding: 2px 8px; border-radius: 9999px; font-size: 10px; font-weight: 600; background: var(--accent-bg); color: var(--accent); font-family: var(--font-mono); border: 1px solid rgba(0, 229, 191, 0.15); }

/* ── Logs Modal ────────────────────────────────────────── */
.modal { display: none; position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,.75); z-index: 100; animation: fadeIn 150ms ease; backdrop-filter: blur(4px); }
.modal-content { background: var(--surface); margin: 6% auto; padding: 24px; width: 86%; max-width: 720px; max-height: 72vh; border-radius: var(--radius); overflow: auto; border: 1px solid var(--border); box-shadow: var(--shadow-lg); overscroll-behavior: contain; }
.modal-content h2 { font-size: 15px; font-weight: 700; }
.modal-content pre { background: var(--bg); padding: 14px; border-radius: var(--radius-sm); margin-top: 14px; font-size: 11.5px; line-height: 1.5; overflow: auto; max-height: 55vh; white-space: pre-wrap; color: var(--text-muted); font-family: var(--font-mono); }

/* ── Error line ───────────────────────────────────────── */
.error-line { font-size: 11px; color: var(--red); margin-top: 8px; padding: 6px 10px; background: var(--red-bg); border-radius: var(--radius-sm); word-break: break-all; font-family: var(--font-mono); border: 1px solid rgba(255, 64, 96, 0.15); }

/* ── Chat history ─────────────────────────────────────── */
.chat-history-item { display: flex; justify-content: space-between; align-items: center; padding: 10px 12px; border-radius: var(--radius-sm); transition: background var(--transition); cursor: pointer; }
.chat-history-item:hover { background: var(--surface-2); }
.chat-history-item + .chat-history-item { border-top: 1px solid var(--border); }
.chat-history-main { flex: 1; min-width: 0; }
.chat-history-title { font-size: 13px; font-weight: 500; word-break: break-all; }
.chat-history-meta { font-size: 10px; color: var(--text-dim); margin-top: 2px; }
.chat-history-actions { display: flex; gap: 4px; flex-shrink: 0; margin-left: 8px; }

/* ── Model list ────────────────────────────────────────── */
.model-row { display: flex; justify-content: space-between; align-items: center; padding: 12px 14px; border-radius: var(--radius-sm); transition: all var(--transition); cursor: pointer; }
.model-row:hover .name { color: var(--accent); }
.model-row .info-icon { font-size: 11px; opacity: 0; transition: opacity var(--transition); margin-left: 4px; }
.model-row:hover .info-icon { opacity: 0.6; }
.model-row:hover { background: var(--accent-bg); }
.model-row .name { font-size: 13px; font-weight: 500; word-break: break-all; }
.model-row .info { font-size: 11px; color: var(--text-dim); margin-top: 4px; display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }

/* ── Animations / Utilities ────────────────────────────── */
.spinner { display: inline-block; width: 14px; height: 14px; border: 2px solid var(--border); border-top-color: var(--accent); border-radius: 50%; animation: spin .6s linear infinite; vertical-align: middle; }
.refreshing { opacity: .5; pointer-events: none; transition: opacity var(--transition); }
.tag { display: inline-block; padding: 2px 8px; border-radius: 6px; font-size: 10px; font-weight: 600; background: var(--surface-2); color: var(--text-muted); font-family: var(--font-mono); border: 1px solid var(--border); }
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
  .sidebar nav .nav-item .label { display: none; }
  .sidebar nav .nav-item { justify-content: center; padding: 10px; }
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
      <div style="display:flex;align-items:center;gap:6px">
        <span class="badge badge-blue" id="instanceCount"></span>
        <button class="ghost small" onclick="loadInstances()" title="Refresh instances" style="font-size:13px;padding:2px 4px">🔄</button>
      </div>
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
    <p><span id="modelCount"><span class="spinner" id="modelCountSpinner"></span> Loading…</span> <button class="ghost small" onclick="loadModels()" title="Refresh models" style="font-size:12px;padding:1px 6px;vertical-align:middle">🔄</button></p>
  </div>
  <div id="modelList" class="card"><div class="card-body"><div class="empty-state"><span class="spinner"></span> Loading models…</div></div></div>
  <div class="section" style="margin-top:32px">
    <div class="section-header"><h2>Pull Model</h2></div>
    <div class="card"><div class="card-body">
      <div class="pull-row" style="position:relative">
        <input type="text" id="pullInput" placeholder="Search HuggingFace for a model…" autocomplete="off" oninput="onPullInputChange(this.value)">
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
      <div class="detail-row"><span class="detail-label">API Name</span><span class="detail-value" id="md-apiname" style="font-family:var(--font-mono);font-size:12px">—</span></div>
      <div class="detail-row"><span class="detail-label">Size</span><span class="detail-value" id="md-size">—</span></div>
      <div class="detail-row"><span class="detail-label">Path</span><span class="detail-value" id="md-path" style="word-break:break-all;font-family:var(--font-mono);font-size:11px">—</span></div>
    </div>
    <button class="primary" onclick="launchModelFromDetails()" id="md-launch-btn" style="margin-top:16px;width:100%">🚀 Launch</button>
  </div>
</div>

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
      <div class="header-left">
        <h1>Chat</h1>
        <select id="chatInstanceSelect" onchange="selectChatInstance()" aria-label="Select instance"><option value="">— select a running instance —</option></select>
      </div>
      <div class="header-actions">
        <button class="pill-btn history" onclick="showChatHistory()" aria-label="Chat history" title="Chat history">📋 History</button>
        <button class="pill-btn clear" onclick="clearChat()" aria-label="Clear chat" title="Clear chat">✕ Clear</button>
      </div>
    </div>
    <div id="chatPanel" class="chat-msgs" style="display: none" aria-live="polite"></div>
    <div id="chatEmpty" class="empty-state" style="flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center">
      <div class="icon">💬</div>
      <div class="title">No instance selected</div>
      <p>Launch an instance from the Dashboard to start chatting</p>
    </div>
    <div class="chat-input-row">
      <div id="contextMeter" style="display:none;height:3px;background:var(--border);border-radius:2px;overflow:hidden">
        <div id="contextBar" style="height:100%;width:0%;background:var(--accent);border-radius:2px;transition:width 300ms ease"></div>
      </div>
      <div class="input-wrap">
        <textarea id="chatInput" rows="1" placeholder="Type a message… (Enter to send)" onkeydown="if(event.key=='Enter'&&!event.shiftKey){event.preventDefault();sendChat()}" autocomplete="off" style="resize:none;padding:9px 12px;font-family:inherit;font-size:13px;line-height:1.5;background:var(--bg);border:1px solid var(--border);border-radius:var(--radius-sm);color:var(--text);outline:none;" oninput="autoGrow(this)"></textarea>
        <button class="primary" onclick="sendChat()">Send</button>
      </div>
      <div id="ctxLabelWrap" style="display:none;justify-content:center;font-size:10px;color:var(--text-dim)">
        <span id="ctxLabel" class="ctx-label"></span>
      </div>
    </div>
  </div>
</div>

<!-- ── Settings ──────────────────────────────────────── -->
<div id="view-settings" class="view" role="tabpanel" aria-label="Settings">
  <div class="page-header">
    <h1>Settings</h1>
    <p>Configuration and system information</p>
  </div>
  <div class="card"><div class="card-body">
    <div class="detail-row">
      <span class="detail-label">⚡ gollama</span>
      <span class="detail-value"><span class="tag" id="s-version">—</span></span>
    </div>
    <div class="detail-row">
      <span class="detail-label">🧠 llama-server</span>
      <span class="detail-value"><span class="tag" id="s-llama-version">—</span></span>
    </div>
    <div class="detail-row">
      <span class="detail-label">🔧 Backend</span>
      <span class="detail-value"><span class="tag" id="s-backend">—</span></span>
    </div>
    <div class="detail-row">
      <span class="detail-label">📁 Config</span>
      <span class="detail-value"><code style="font-size:12px;color:var(--text-dim);font-family:var(--font-mono)">~/.gollama/config.json</code></span>
    </div>
    <div class="detail-row">
      <span class="detail-label">📦 Models dir</span>
      <span class="detail-value"><code style="font-size:12px;color:var(--text-dim);font-family:var(--font-mono)">~/.gollama/models/</code></span>
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
  <div class="card settings-card" id="settings-ql" style="margin-top:16px"><div class="card-body">
    <div style="display:flex;align-items:flex-start;justify-content:space-between;margin-bottom:8px">
      <div>
        <div style="font-size:13px;font-weight:600">Quick Launch Defaults</div>
        <div style="font-size:11px;color:var(--text-dim);margin-top:2px">Pre-fills the Quick Launch form for manual launches</div>
      </div>
      <button class="ghost small settings-edit-btn" onclick="toggleEditSettings('ql')">✏️ Edit</button>
    </div>
    <div class="settings-readonly" id="ql-readonly">
      <div style="font-size:12px;color:var(--text-dim)">No custom flags set.</div>
    </div>
    <div class="settings-form">
      <div id="settingsFlagsContainer"></div>
      <button class="ghost small" onclick="addSettingsFlag()" style="margin-top:4px">＋ Add Flag</button>
      <div style="margin-top:10px">
        <button class="primary small" onclick="saveSettingsFlags()">Save</button>
        <button class="secondary small" onclick="cancelEditSettings('ql')">Cancel</button>
        <span id="settingsFlagsStatus" style="font-size:12px;color:var(--text-muted);margin-left:8px"></span>
      </div>
    </div>
  </div></div>
  <div class="card settings-card" id="settings-api" style="margin-top:16px"><div class="card-body">
    <div style="display:flex;align-items:flex-start;justify-content:space-between;margin-bottom:8px">
      <div>
        <div style="font-size:13px;font-weight:600">API Launch Defaults</div>
        <div style="font-size:11px;color:var(--text-dim);margin-top:2px">Used when auto-launching via API — acts as fallback when no profile matches</div>
      </div>
      <button class="ghost small settings-edit-btn" onclick="toggleEditSettings('api')">✏️ Edit</button>
    </div>
    <div class="settings-readonly" id="api-readonly">
      <div style="font-size:12px;color:var(--text-dim)">No custom flags set.</div>
    </div>
    <div class="settings-form">
      <div id="proxyFlagsContainer"></div>
      <button class="ghost small" onclick="addProxyFlag()" style="margin-top:4px">＋ Add Flag</button>
      <div style="margin-top:10px">
        <button class="primary small" onclick="saveProxyFlags()">Save</button>
        <button class="secondary small" onclick="cancelEditSettings('api')">Cancel</button>
        <span id="proxyFlagsStatus" style="font-size:12px;color:var(--text-muted);margin-left:8px"></span>
      </div>
    </div>
  </div></div>
  <div class="card settings-card" id="settings-profiles" style="margin-top:16px"><div class="card-body">
    <div style="display:flex;align-items:flex-start;justify-content:space-between;margin-bottom:8px">
      <div>
        <div style="font-size:13px;font-weight:600">Model Profiles</div>
        <div style="font-size:11px;color:var(--text-dim);margin-top:2px">Named flag sets for auto-launch routing. When a model profile's model matches the request, its flags override proxy defaults.</div>
      </div>
      <button class="ghost small settings-edit-btn" onclick="toggleEditSettings('profiles')">✏️ Edit</button>
    </div>
    <div class="settings-readonly" id="profiles-readonly">
      <div style="font-size:12px;color:var(--text-dim)">No model profiles configured.</div>
    </div>
    <div class="settings-form">
      <div id="profilesContainer"></div>
      <button class="ghost small" onclick="addProfile()" style="margin-top:4px">＋ Add Model Profile</button>
      <div style="margin-top:10px">
        <button class="primary small" onclick="saveProfiles()">Save</button>
        <button class="secondary small" onclick="cancelEditSettings('profiles')">Cancel</button>
        <span id="profilesStatus" style="font-size:12px;color:var(--text-muted);margin-left:8px"></span>
      </div>
    </div>
  </div></div>
  <div class="card" style="margin-top:16px"><div class="card-body" style="text-align:center">
    <button class="danger" onclick="restartGollama()">🔄 Restart gollama</button>
    <div style="font-size:11px;color:var(--text-dim);margin-top:6px">Applies config changes and picks up new version</div>
  </div></div>
</div>

<!-- ── Logs Modal ──────────────────────────────────── -->
<div class="modal" id="logModal" role="dialog" aria-modal="true" aria-label="Instance logs">
  <div class="modal-content" onclick="event.stopPropagation()">
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 4px">
      <h2>📋 Logs</h2>
      <div style="display:flex;gap:4px">
        <button class="small" onclick="copyLogs()" aria-label="Copy logs">📋 Copy</button>
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

// ── Navigation ──────────────────────────────────────
function switchView(name) {
  document.querySelectorAll('.view').forEach(function(v) { v.classList.remove('active'); });
  document.querySelectorAll('.nav-item').forEach(function(a) { a.classList.remove('active'); });
  var view = document.getElementById('view-' + name);
  if (view) view.classList.add('active');
  document.querySelector('.nav-item[onclick*="' + name + '"]').classList.add('active');
  currentView = name;
  if (name == 'dashboard') loadInstances();
  if (name == 'chat') {
    if (chatPort) {
      document.getElementById('chatPanel').style.display = 'block';
      document.getElementById('chatEmpty').style.display = 'none';
      updateContextMeter();
    }
  }
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
      var bc = i.status == 'running' ? (i.ready ? 'badge-green' : 'badge-blue') : 'badge-red';
      var statusLabel = i.status == 'running' ? (i.ready ? 'running' : 'starting…') : i.status;
      var mn = i.model || '?';
      var tps = i.tokens_per_sec ? '<span style="color: var(--green); font-variant-numeric: tabular-nums">⚡ ' + i.tokens_per_sec.toFixed(1) + ' t/s</span>' : '';
      var uptime = i.started_at ? (function() { var s = Math.floor((Date.now() - new Date(i.started_at).getTime()) / 1000); return '<span title="Uptime">⏱ ' + (s > 86400 ? Math.floor(s/86400)+'d ' : '') + (s > 3600 ? Math.floor((s%86400)/3600)+'h ' : '') + Math.floor((s%3600)/60)+'m</span>'; })() : '';
      var idle = i.last_activity ? (function() { var s = Math.floor((Date.now() - new Date(i.last_activity).getTime()) / 1000); if (s < 60) return ''; return '<span title="Idle time">💤 ' + (s > 3600 ? Math.floor(s/3600)+'h ' : '') + Math.floor((s%3600)/60)+'m</span>'; })() : '';
      var tokens = i.total_tokens ? '<span title="Total tokens">🔤 ' + (i.total_tokens > 999 ? Math.round(i.total_tokens/1000) + 'K' : i.total_tokens) + '</span>' : '';
      var metrics = (i.device_split ? '<span title="Model split">📊 ' + i.device_split + '</span>' : '') + (i.profile ? ' <span class="badge badge-profile" title="Active model profile">📋 ' + escHtml(i.profile) + '</span>' : '');
      var flags = i.flags && i.flags.length ? formatFlags(i.flags) : '';
      var flagsHtml = flags ? '<div style="font-size: 11px; color: var(--text-dim); margin-top: 8px; padding-top: 8px; border-top: 1px solid var(--border); word-break: break-all; font-family: var(--font-mono)">' + escHtml(flags) + '</div>' : '';
      var errDiv = i.status != 'running' ? '<div class="error-line" id="err-' + i.port + '"></div>' : '';
      return '<div class="inst-card' + cls + '"><div class="title">' + escHtml(mn.length > 40 ? mn.slice(0, 40) + '…' : mn) + '</div>' +
        '<div class="meta"><span>Port ' + i.port + '</span>' + tps + uptime + idle + tokens + metrics + '<span class="badge ' + bc + '">' + statusLabel + '</span></div>' +
        errDiv + flagsHtml +
        '<div class="actions"><button class="small danger" onclick="stopInstance(' + i.port + ')" aria-label="Stop instance on port ' + i.port + '">⏹ Stop</button>' +
        '<button class="small secondary" onclick="restartInstance(' + i.port + ')" aria-label="Restart instance on port ' + i.port + '">🔄 Restart</button>' +
        '<button class="small secondary" onclick="selectChatFor(' + i.port + ', \'' + escAttr(mn.replace(/'/g, '')) + '\')" aria-label="Chat with instance on port ' + i.port + '">💬 Chat</button>' +
        '<button class="small secondary" onclick="window.open(\'http://\' + location.hostname + \':' + i.port + '\', \'_blank\'); return false" aria-label="Open instance on port ' + i.port + '">🌐 Open</button>' +
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

  var userFlags = collectFlags(document.getElementById('flagsContainer'));

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
            var partLabel = (d.total_parts > 1) ? ' [' + d.part + '/' + d.total_parts + ']' : '';
            document.getElementById('pullStatus').textContent = 'Downloading\u2026' + partLabel;
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
  if (chatPort) { chatHistory = []; document.getElementById('chatPanel').innerHTML = ''; document.getElementById('chatPanel').style.display = 'block'; document.getElementById('chatEmpty').style.display = 'none'; addSystemMsg('Connected'); }
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
  document.getElementById('themeToggle').innerHTML = b.classList.contains('light') ? '<span>☀️</span><span class="label">Theme</span>' : '<span>🌙</span><span class="label">Theme</span>';
  localStorage.setItem('gollama-theme', b.classList.contains('light') ? 'light' : 'dark');
}
(function() {
  if (localStorage.getItem('gollama-theme') === 'light') { document.body.classList.add('light'); document.documentElement.classList.add('light'); document.getElementById('themeToggle').innerHTML = '<span>☀️</span><span class="label">Theme</span>'; }
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
  var container = document.getElementById('profiles-readonly');
  if (!profiles || Object.keys(profiles).length === 0) {
    container.innerHTML = '<div style="font-size:12px;color:var(--text-dim)">No model profiles configured.</div>';
    return;
  }
  var html = '<div style="display:flex;flex-direction:column;gap:8px">';
  for (var name in profiles) {
    var p = profiles[name];
    html += '<div style="background:var(--surface-2);border-radius:6px;padding:10px 12px;border:1px solid var(--border)">' +
      '<div style="font-weight:600;font-size:13px;margin-bottom:2px">' + escHtml(name) + '</div>';
    if (p.model) html += '<div style="font-size:11px;color:var(--text-dim);margin-bottom:2px">Model: <code style="font-size:11px">' + escHtml(p.model) + '</code></div>';
    if (p.description) html += '<div style="font-size:11px;color:var(--text-dim)">' + escHtml(p.description) + '</div>';
    if (p.flags && p.flags.length) {
      html += '<div style="display:flex;flex-wrap:wrap;gap:4px;margin-top:4px">';
      for (var i = 0; i < p.flags.length; i++) {
        html += '<span class="tag" style="font-family:var(--font-mono);font-size:10px">' + escHtml(p.flags[i]) + '</span>';
        if (i + 1 < p.flags.length && !standaloneFlags[p.flags[i]] && !p.flags[i + 1].startsWith('--')) {
          i++;
          html += '<span class="tag" style="font-family:var(--font-mono);font-size:10px;color:var(--text-dim)">' + escHtml(p.flags[i]) + '</span>';
        }
      }
      html += '</div>';
    }
    html += '</div>';
  }
  html += '</div>';
  container.innerHTML = html;
}

async function loadSettings() {
  try {
    var vr = await fetch('/api/v1/version'), vd = await vr.json();
    if (vd.version) document.getElementById('s-version').textContent = vd.version;
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
    pc.innerHTML = '';
    var nextPid = 0;
    if (cfg.profiles) {
      var i = 0;
      for (var name in cfg.profiles) {
        var p = cfg.profiles[name];
        renderProfile({ name: name, model: p.model || '', desc: p.description || '', flags: p.flags, strip_reasoning: p.strip_reasoning, env: p.env }, i);
        i++;
      }
      nextPid = i;
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
function addProfile() {
  var c = document.getElementById('profilesContainer');
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
      '<div style="flex:2;min-width:180px"><label style="font-size:11px;color:var(--text-dim);display:block;margin-bottom:2px">Description</label><input type="text" class="flag-custom" placeholder="What is this profile for?" style="width:100%" id="pd-' + i + '"></div>' +
    '</div>' +
    '<div style="font-size:11px;color:var(--text-dim);margin-bottom:6px">Flags</div>' +
    '<div style="display:flex;gap:4px;flex-wrap:wrap;margin-bottom:6px" id="pf-' + i + '"></div>' +
    '<button class="ghost small" onclick="addProfileFlag(' + i + ')" style="font-size:11px">＋ Add Flag</button>';
  d.id = 'pc-' + i;
  c.appendChild(d);
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
  var c = document.getElementById('profilesContainer');
  var d = document.createElement('div');
  d.id = 'pc-' + i;
  d.style = 'background:var(--surface-2);border-radius:8px;padding:14px;margin-bottom:12px;border:1px solid var(--border)';
  d.innerHTML =
    '<div style="display:flex;gap:8px;align-items:center;margin-bottom:10px">' +
      '<span style="font-size:15px">📋</span>' +
      '<input type="text" placeholder="Model profile name" style="flex:1;font-weight:600;font-size:14px;min-width:120px;padding:8px 10px;background:var(--bg);border:1px solid var(--border);border-radius:var(--radius-sm);color:var(--text);outline:none;transition:border-color var(--transition);font-family:var(--font)" id="pn-' + i + '" value="' + escAttr(p.name) + '">' +
      '<button class="small danger" onclick="document.getElementById(\'pc-' + i + '\').remove()" title="Remove profile">\u2715</button>' +
    '</div>' +
    '<div style="display:flex;gap:8px;margin-bottom:8px;flex-wrap:wrap">' +
      '<div style="flex:1;min-width:140px"><label style="font-size:11px;color:var(--text-dim);display:block;margin-bottom:2px">Model (optional)</label><input type="text" class="flag-custom" placeholder="e.g. qwen3-coder-next" style="width:100%" id="pm-' + i + '" value="' + escAttr(p.model || '') + '"></div>' +
      '<div style="flex:2;min-width:180px"><label style="font-size:11px;color:var(--text-dim);display:block;margin-bottom:2px">Description</label><input type="text" class="flag-custom" placeholder="What is this profile for?" style="width:100%" id="pd-' + i + '" value="' + escAttr(p.desc || '') + '"></div>' +
      '<div style="flex:0 0 auto;display:flex;align-items:end;padding-bottom:2px"><label style="font-size:11px;color:var(--text-dim);display:flex;align-items:center;gap:4px;white-space:nowrap;cursor:pointer"><input type="checkbox" id="ps-' + i + '" ' + (p.strip_reasoning ? 'checked' : '') + ' style="accent-color:var(--accent)"> Strip reasoning</label></div>' +
    '</div>' +
    '<div style="font-size:11px;color:var(--text-dim);margin-bottom:6px">Flags</div>' +
    '<div style="display:flex;gap:4px;flex-wrap:wrap;margin-bottom:6px" id="pf-' + i + '"></div>' +
    '<button class="ghost small" onclick="addProfileFlag(' + i + ')" style="font-size:11px">＋ Add Flag</button>' +
    '<div style="font-size:11px;color:var(--text-dim);margin-top:8px;margin-bottom:6px">Environment Variables</div>' +
    '<div style="display:flex;flex-direction:column;gap:4px;margin-bottom:4px" id="pe-' + i + '"></div>' +
    '<button class="ghost small" onclick="addProfileEnv(' + i + ')" style="font-size:11px">＋ Add Env Var</button>';
  c.appendChild(d);
  var fc = document.getElementById('pf-' + i);
  if (p.flags) {
    p.flags.forEach(function(f, fi) {
      if (fi % 2 == 0) fc.appendChild(makeFlagRow(f, p.flags[fi+1] || ''));
    });
  }
  var ec = document.getElementById('pe-' + i);
  if (p.env) {
    for (var ek in p.env) {
      addProfileEnvRow(i, ek, p.env[ek]);
    }
  }
}

async function saveProfiles() {
  var st = document.getElementById('profilesStatus');
  var c = document.getElementById('profilesContainer');
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
    var flags = [];
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
    var profileObj = { model: model ? model.value : '', description: desc ? desc.value : '', flags: flags };
    if (stripEl && stripEl.checked) profileObj.strip_reasoning = true;
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
  try {
    var r = await fetch('/api/v1/config', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ profiles: profiles }) });
    if (r.ok) { st.textContent = 'Saved'; setTimeout(function() { st.textContent = ''; }, 2000); loadSettings(); document.getElementById('settings-profiles').classList.remove('editing'); }
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

// ── Init (staggered, no pile-up) ─────────────────────
loadModels();
setTimeout(loadDefaultFlags, 50);
setTimeout(loadPresets, 50);
setTimeout(loadInstances, 100);
setTimeout(loadSettings, 150);
</script>
</body>
</html>`
