package ui

const Page = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>gollama</title>
<style>
:root {
  --bg:#0e0e12; --surface:#16161c; --card:#1c1c24; --border:#262630;
  --border-hover:#34344a; --text:#e8e8ed; --muted:#8888a0; --dim:#5c5c72;
  --accent:#4a8eff; --accent-hover:#6ba3ff; --accent-bg:#1a2744;
  --green:#2ed573; --green-bg:#0a2e1a; --red:#ff4757; --red-bg:#2e0a0f;
  --amber:#ffa502; --amber-bg:#2e1a06; --blue:#4a8eff; --blue-bg:#1a2744;
  --sidebar-w:220px; --radius:8px; --radius-sm:5px;
  --font: -apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;
}
.light {
  --bg:#f5f5f7; --surface:#ffffff; --card:#fafafc; --border:#d4d4dc;
  --border-hover:#b0b0be; --text:#1a1a24; --muted:#6b6b80; --dim:#8e8ea0;
  --accent:#4a8eff; --accent-hover:#3578e0; --accent-bg:#e8f0ff;
  --green:#1a9d52; --green-bg:#e6f7ed; --red:#d63031; --red-bg:#ffe8e8;
  --amber:#d4890a; --amber-bg:#fff3e0; --blue:#3578e0; --blue-bg:#e8f0ff;
}
* { margin:0; padding:0; box-sizing:border-box; }
html,body { height:100%; }
body {
  font-family:var(--font); background:var(--bg); color:var(--text);
  display:flex; font-size:14px; line-height:1.5;
  -webkit-font-smoothing:antialiased;
}

/* ── Sidebar ─────────────────────────────────────────── */
.sidebar {
  width:var(--sidebar-w); background:var(--surface); border-right:1px solid var(--border);
  display:flex; flex-direction:column; flex-shrink:0; height:100vh; position:sticky; top:0;
  transition:width .25s ease;
  overflow:hidden;
}
.sidebar.collapsed { width:56px; }
.sidebar .logo {
  padding:16px 18px; font-size:16px; font-weight:700; letter-spacing:-.3px;
  border-bottom:1px solid var(--border); white-space:nowrap; overflow:hidden;
  display:flex; align-items:center; gap:10px; min-height:52px;
}
.sidebar .logo span { color:var(--accent); }
.sidebar .logo .toggle {
  font-size:14px; cursor:pointer; background:none; border:none; color:var(--muted);
  padding:4px; border-radius:4px; flex-shrink:0; line-height:1;
}
.sidebar .logo .toggle:hover { color:var(--text); }
.sidebar nav { flex:1; padding:12px 8px; display:flex; flex-direction:column; gap:2px; }
.sidebar a {
  display:flex; align-items:center; gap:10px; padding:9px 12px; border-radius:var(--radius-sm);
  color:var(--muted); text-decoration:none; font-size:13px; font-weight:500; cursor:pointer;
  transition:all .15s; white-space:nowrap; overflow:hidden;
}
.sidebar a:hover { background:var(--card); color:var(--text); }
.sidebar a.active { background:var(--accent-bg); color:var(--accent-hover); }
.sidebar a .icon { font-size:16px; width:20px; text-align:center; flex-shrink:0; }
.sidebar a .label { transition:opacity .2s; }
.sidebar.collapsed a .label { opacity:0; }
.sidebar.collapsed .logo span:not(.toggle) { opacity:0; }
.sidebar .bottom { border-top:1px solid var(--border); padding:8px; margin-top:auto; }

/* ── Main ────────────────────────────────────────────── */
.main { flex:1; overflow-y:auto; padding:28px 36px; }
.view { display:none; }
.view.active { display:block; }

/* ── Page Header ─────────────────────────────────────── */
.page-header { margin-bottom:24px; }
.page-header h1 { font-size:20px; font-weight:700; letter-spacing:-.3px; }
.page-header p { color:var(--muted); font-size:13px; margin-top:2px; }

/* ── Metrics ──────────────────────────────────────────── */
.metrics { display:grid; grid-template-columns:repeat(auto-fit,minmax(280px,1fr)); gap:14px; margin-bottom:28px; }
.metric-card {
  background:var(--card); border:1px solid var(--border); border-radius:var(--radius);
  padding:16px 18px; transition:border-color .2s;
}
.metric-card:hover { border-color:var(--border-hover); }
.metric-card .label { font-size:12px; color:var(--muted); font-weight:500; margin-bottom:4px; }
.metric-card .value { font-size:22px; font-weight:700; letter-spacing:-.5px; }
.metric-card .sub { font-size:12px; color:var(--dim); margin-top:2px; }

/* ── Section ──────────────────────────────────────────── */
.section { margin-bottom:28px; }
.section-header {
  display:flex; align-items:center; justify-content:space-between; margin-bottom:12px;
}
.section-header h2 { font-size:14px; font-weight:600; color:var(--muted); text-transform:uppercase; letter-spacing:.5px; }
.section-header .badge { font-size:11px; color:var(--dim); font-weight:500; }

/* ── Cards ────────────────────────────────────────────── */
.card { background:var(--card); border:1px solid var(--border); border-radius:var(--radius); }
.card-body { padding:16px; }

/* ── Instance Card ────────────────────────────────────── */
.instance-grid { display:grid; grid-template-columns:repeat(auto-fill,minmax(380px,1fr)); gap:14px; }
.inst-card {
  background:var(--card); border:1px solid var(--border); border-radius:var(--radius);
  border-left:3px solid var(--green); padding:16px; transition:border-color .2s;
}
.inst-card:hover { border-color:var(--border-hover); }
.inst-card.stopped { border-left-color:var(--dim); opacity:.5; }
.inst-card.error { border-left-color:var(--red); }
.inst-card .title { font-size:13px; font-weight:600; word-break:break-all; }
.inst-card .meta { font-size:12px; color:var(--dim); margin-top:6px; display:flex; gap:12px; flex-wrap:wrap; align-items:center; }
.inst-card .actions { margin-top:12px; display:flex; gap:6px; flex-wrap:wrap; }

/* ── Quick Launch ──────────────────────────────────────── */
.launch-row { display:grid; grid-template-columns:1fr 120px auto; gap:10px; align-items:end; margin-bottom:10px; }
.launch-row .field { display:flex; flex-direction:column; gap:4px; }
.launch-row .field label { font-size:11px; color:var(--dim); text-transform:uppercase; letter-spacing:.5px; font-weight:600; }

/* ── Forms ─────────────────────────────────────────────── */
select, input[type=text], input[type=number] {
  width:100%; padding:8px 10px; background:var(--surface); border:1px solid var(--border);
  border-radius:var(--radius-sm); color:var(--text); font-size:13px; outline:none;
  transition:border-color .2s; font-family:var(--font);
}
select:focus, input:focus { border-color:var(--accent); }
select { cursor:pointer; }

/* ── Buttons ───────────────────────────────────────────── */
button {
  padding:8px 16px; border-radius:var(--radius-sm); border:none; font-size:13px;
  font-weight:600; cursor:pointer; transition:all .15s; font-family:var(--font);
}
button.primary { background:var(--accent); color:#fff; }
button.primary:hover { background:var(--accent-hover); }
button.secondary { background:var(--border); color:var(--text); }
button.secondary:hover { background:var(--border-hover); }
button.danger { background:var(--red); color:#fff; }
button.danger:hover { opacity:.85; }
button.small { padding:5px 10px; font-size:11px; }
button:disabled { opacity:.4; cursor:default; }
button.ghost { background:transparent; color:var(--muted); padding:6px 8px; }
button.ghost:hover { background:var(--card); color:var(--text); }

/* ── Badge ──────────────────────────────────────────────── */
.badge { display:inline-block; padding:2px 7px; border-radius:4px; font-size:10px; font-weight:700; letter-spacing:.3px; }
.badge-green { background:var(--green-bg); color:var(--green); }
.badge-red { background:var(--red-bg); color:var(--red); }
.badge-amber { background:var(--amber-bg); color:var(--amber); }
.badge-blue { background:var(--blue-bg); color:var(--blue); }

/* ── Flag rows ─────────────────────────────────────────── */
.flag-row { display:flex; gap:6px; margin-bottom:4px; }
.flag-row input { flex:1; }
.flag-row button { flex-shrink:0; }

/* ── Empty state ───────────────────────────────────────── */
.empty-state { text-align:center; padding:40px 20px; color:var(--dim); }
.empty-state .icon { font-size:32px; margin-bottom:8px; }

/* ── Chat ──────────────────────────────────────────────── */
.chat-container { display:flex; flex-direction:column; height:calc(100vh - 48px); }
.chat-header { padding-bottom:12px; margin-bottom:12px; border-bottom:1px solid var(--border); }
.chat-header select { width:auto; min-width:200px; }
.chat-msgs { flex:1; overflow-y:auto; padding:4px 0; margin-bottom:10px; }
.chat-msgs .msg { margin-bottom:10px; padding:10px 14px; border-radius:10px; max-width:80%; line-height:1.6; font-size:13px; }
.chat-msgs .user { background:var(--accent-bg); margin-left:auto; border-bottom-right-radius:4px; }
.chat-msgs .assistant { background:var(--card); border:1px solid var(--border); border-bottom-left-radius:4px; }
.chat-msgs .system { background:transparent; color:var(--dim); font-style:italic; font-size:12px; text-align:center; max-width:100%; }
.chat-input-row { display:flex; gap:8px; }
.chat-input-row input { flex:1; }
.chat-input-row button { flex-shrink:0; }
.chat-loading { animation:pulse 1.2s infinite; display:inline-block; letter-spacing:4px; font-size:16px; line-height:1; color:var(--dim); }
.reasoning { color:var(--dim); font-style:italic; font-size:12px; border-left:2px solid var(--border); padding-left:10px; margin-bottom:6px; }



/* ── Logs Modal ────────────────────────────────────────── */
.modal { display:none; position:fixed; top:0; left:0; width:100%; height:100%; background:rgba(0,0,0,.7); z-index:100; }
.modal-content { background:var(--surface); margin:5% auto; padding:20px; width:80%; max-width:700px; max-height:70vh; border-radius:var(--radius); overflow:auto; border:1px solid var(--border); }
.modal-content pre { background:var(--bg); padding:12px; border-radius:var(--radius-sm); margin-top:12px; font-size:11px; line-height:1.4; overflow:auto; max-height:55vh; white-space:pre-wrap; color:var(--muted); }

/* ── Error line ───────────────────────────────────────── */
.error-line { font-size:11px; color:var(--red); margin-top:6px; padding:4px 8px; background:var(--red-bg); border-radius:4px; word-break:break-all; }

/* ── Model list ────────────────────────────────────────── */
.model-row { display:flex; justify-content:space-between; align-items:center; padding:10px 12px; border-radius:var(--radius-sm); transition:background .15s; }
.model-row:hover { background:var(--surface); }
.model-row .name { font-size:13px; }
.model-row .info { font-size:11px; color:var(--dim); margin-top:3px; display:flex; gap:6px; flex-wrap:wrap; align-items:center; }

/* ── Animations ────────────────────────────────────────── */
@keyframes pulse { 0%,100%{opacity:1} 50%{opacity:.4} }
@keyframes spin { to{transform:rotate(360deg)} }
.loading { animation:pulse 1.5s infinite; }
.spinner { display:inline-block; width:12px; height:12px; border:2px solid var(--border); border-top-color:var(--accent); border-radius:50%; animation:spin .6s linear infinite; vertical-align:middle; }
.refreshing { opacity:.5; pointer-events:none; transition:opacity .2s; }

/* ── Pull model row ────────────────────────────────────── */
.pull-row { display:flex; gap:8px; margin-bottom:12px; }
.pull-row input { flex:1; }
.pull-row button { flex-shrink:0; }

/* ── Responsive ────────────────────────────────────────── */
@media(max-width:768px){
  .sidebar { width:56px; }
  .sidebar .logo { font-size:0; padding:16px; }
  .sidebar .logo::after { content:'🦙'; font-size:18px; }
  .sidebar a span { display:none; }
  .sidebar a { justify-content:center; padding:9px; }
  .sidebar .bottom { display:none; }
  .main { padding:16px; }
  .metrics { grid-template-columns:1fr 1fr; }
  .launch-row { grid-template-columns:1fr; }
  .instance-grid { grid-template-columns:1fr; }
}
</style>
</head>
<body>

<!-- ─── Sidebar ──────────────────────────────────────── -->
<aside class="sidebar">
  <div class="logo"><button class="toggle" onclick="toggleSidebar()">◀</button><span>gollama</span><span>.</span></div>
  <nav>
    <a class="active" onclick="switchView('dashboard')">
      <span class="icon">📊</span><span>Dashboard</span>
    </a>
    <a onclick="switchView('models')">
      <span class="icon">📦</span><span>Models</span>
    </a>
    <a onclick="switchView('chat')">
      <span class="icon">💬</span><span>Chat</span>
    </a>
    <a onclick="switchView('settings')">
      <span class="icon">⚙️</span><span>Settings</span>
    </a>
  </nav>
  <div class="bottom" style="display:flex;gap:4px;padding:10px 12px">
    <button class="ghost small" onclick="toggleTheme()" id="themeToggle" style="flex:1;justify-content:center" title="Toggle theme">🌙</button>
  </div>
</aside>

<!-- ─── Main ──────────────────────────────────────────── -->
<main class="main" id="main">

<!-- ── Dashboard ────────────────────────────────────── -->
<div id="view-dashboard" class="view active">
  <div class="page-header">
    <h1>Dashboard</h1>
    <p>Monitor and manage your llama.cpp instances</p>
  </div>

  <div class="metrics">
    <div class="metric-card"><div class="label">Models</div><div class="value" id="metric-models">—</div><div class="sub">downloaded</div></div>
    <div class="metric-card"><div class="label">Running</div><div class="value" id="metric-running">—</div><div class="sub">active instances</div></div>
    <div class="metric-card"><div class="label">Tokens/sec</div><div class="value" id="metric-tps">—</div><div class="sub">fastest instance</div></div>
    <div class="metric-card"><div class="label">Server</div><div class="value"><span id="metric-server">—</span></div><div class="sub" id="metric-backend">backend</div></div>
  </div>

  <div class="section">
    <div class="section-header">
      <h2>Running Instances</h2>
      <span class="badge" id="instanceCount"></span>
    </div>
    <div id="instances" class="instance-grid"><div class="empty-state"><div class="icon">🚀</div><div>No running instances. Launch one below.</div></div></div>
  </div>

  <div class="section">
    <div class="section-header"><h2>Quick Launch</h2></div>
    <div class="card"><div class="card-body">
      <div class="launch-row">
        <div class="field"><label>Model</label><select id="modelSelect"><option value="">Loading models...</option></select></div>
        <div class="field"><label>Port</label><input type="number" id="portInput" value="8081" min="8081" max="8099"></div>
        <div class="field" style="align-self:end"><button class="primary" onclick="launchInstance()" id="launchBtn">Launch</button></div>
      </div>
      <div class="field" style="margin-bottom:6px"><label>Flags</label></div>
      <div id="flagsContainer">
        <div class="flag-row">
          <input type="text" placeholder="e.g. --flash-attn on" class="flag-input">
          <button class="small danger" onclick="this.parentElement.remove()">✕</button>
        </div>
      </div>
      <button class="ghost small" onclick="addFlag()">＋ Add Flag</button>
    </div></div>
  </div>

  <div class="section">
    <div class="section-header"><h2>Pull Model</h2></div>
    <div class="card"><div class="card-body">
      <div class="pull-row">
        <input type="text" id="pullInput" placeholder="hf.co/user/repo:Q4_K_M" value="hf.co/unsloth/Qwen3.5-0.8B-GGUF:Q4_K_M">
        <button class="primary" onclick="pullModel()" id="pullBtn">Pull</button>
      </div>
      <div id="pullStatus" class="text-sm" style="font-size:12px;color:var(--muted)"></div>
    </div></div>
  </div>
</div>

<!-- ── Models ────────────────────────────────────────── -->
<div id="view-models" class="view">
  <div class="page-header">
    <h1>Models</h1>
    <p id="modelCount">downloaded models</p>
  </div>
  <div id="modelList" class="card"><div class="card-body"><div class="empty-state"><div class="icon">📦</div><div>No models downloaded yet.</div></div></div></div>
</div>

<!-- ── Chat ──────────────────────────────────────────── -->
<div id="view-chat" class="view">
  <div class="chat-container">
    <div class="chat-header">
      <div style="display:flex;align-items:center;gap:10px;flex-wrap:wrap">
        <h1 style="font-size:18px;font-weight:700">Chat</h1>
        <select id="chatInstanceSelect" onchange="selectChatInstance()"><option value="">— select a running instance —</option></select>
        <button class="ghost small" onclick="selectChatFor(chatPort,'')">↻</button>
      </div>
    </div>
    <div id="chatPanel" class="chat-msgs" style="display:none"></div>
    <div id="chatEmpty" class="empty-state" style="flex:1;display:flex;flex-direction:column;align-items:center;justify-content:center">
      <div class="icon">💬</div>
      <div>Launch an instance from the Dashboard to start chatting</div>
    </div>
    <div class="chat-input-row" style="margin-top:auto;padding-top:10px;border-top:1px solid var(--border)">
      <input type="text" id="chatInput" placeholder="Type a message..." onkeydown="if(event.key=='Enter')sendChat()">
      <button class="primary" onclick="sendChat()">Send</button>
    </div>
  </div>
</div>

<!-- ── Settings ──────────────────────────────────────── -->
<div id="view-settings" class="view">
  <div class="page-header">
    <h1>Settings</h1>
    <p>Configuration and system information</p>
  </div>
  <div class="card"><div class="card-body">
    <div style="font-size:13px;color:var(--muted);line-height:2">
      <div><strong style="color:var(--text)">gollama</strong> <span id="s-version">—</span></div>
      <div><strong style="color:var(--text)">llama-server</strong> <span id="s-llama-version">—</span></div>
      <div><strong style="color:var(--text)">Backend</strong> <span id="s-backend">—</span></div>
      <div><strong style="color:var(--text)">Config</strong> <code style="font-size:11px;color:var(--dim)">~/.gollama/config.json</code></div>
      <div><strong style="color:var(--text)">Models dir</strong> <code style="font-size:11px;color:var(--dim)">~/.gollama/models/</code></div>
    </div>
  </div></div>
</div>

<!-- ── Logs Modal ──────────────────────────────────── -->
<div class="modal" id="logModal">
  <div class="modal-content">
    <div style="display:flex;justify-content:space-between;align-items:center">
      <h2 style="font-size:14px;font-weight:600">📋 Logs</h2>
      <button class="small danger" onclick="closeLogs()">Close</button>
    </div>
    <pre id="logContent"></pre>
  </div>
</div>



<script>
var chatPort=0,chatHistory=[];
var currentView='dashboard';
var cachedModelCount=0;

// ── Navigation ──────────────────────────────────────
function switchView(name){
  document.querySelectorAll('.view').forEach(function(v){v.classList.remove('active');});
  document.querySelectorAll('.sidebar a').forEach(function(a){a.classList.remove('active');});
  document.getElementById('view-'+name).classList.add('active');
  document.querySelector('.sidebar a[onclick*="'+name+'"]').classList.add('active');
  currentView=name;
  if(name=='chat'&&chatPort)selectChatFor(chatPort,'');
}

// ── Models ───────────────────────────────────────────
async function loadModels(){
  var mc=document.getElementById('modelCount'),ml=document.getElementById('modelList');
  var s=document.getElementById('modelSelect');
  ml.classList.add('refreshing');
  var r=await fetch('/api/v1/models'),m=await r.json();
  ml.classList.remove('refreshing');
  cachedModelCount=m.length;
  mc.textContent=m.length+' downloaded';
  document.getElementById('metric-models').textContent=m.length;

  s.innerHTML='<option value="">— Select model —</option>';
  if(!m||!m.length){s.innerHTML+='<option value="" disabled>No models found. Use gollama pull.</option>';ml.innerHTML='<div class="card-body"><div class="empty-state"><div class="icon">📦</div><div>No models downloaded yet.</div></div></div>';return;}

  var seen={};
  m.forEach(function(x){var n=x.name||'?';if(!seen[n]){seen[n]=1;s.innerHTML+='<option value="'+n+'">'+n+'</option>';}});
  ml.innerHTML='<div class="card-body">'+m.map(function(x){
    var name=x.name||'?',size=x.size?fmtSize(x.size):'?';
    var arch=x.architecture||'',quant=x.quantization||'',ctx=x.context_length||0,badges=[];
    if(quant)badges.push('<span class="badge badge-blue">'+quant+'</span>');
    if(arch)badges.push('<span class="badge badge-amber">'+arch+'</span>');
    if(ctx)badges.push('<span class="badge badge-green">'+(ctx>999?Math.round(ctx/1000)+'K':'<1K')+' ctx</span>');
    return '<div class="model-row"><div><div class="name">'+(name.length>55?name.slice(0,55)+'...':name)+'</div><div class="info">'+size+' '+(badges.length?badges.join(' '):'')+'</div></div><button class="small danger" onclick="deleteModel(\''+name.replace(/\'/g,'')+'\')">🗑</button></div>';
  }).join('')+'</div>';
}

async function deleteModel(name){
  if(!confirm('Delete model "'+name+'"?'))return;
  await fetch('/api/v1/models/delete',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name:name})});
  loadModels();
}

// ── Instances + Metrics ──────────────────────────────
async function loadInstances(){
  var ic=document.getElementById('instanceCount'),c=document.getElementById('instances'),cs=document.getElementById('chatInstanceSelect');
  var r=await fetch('/api/v1/instances'),list=await r.json();
  ic.textContent='('+list.length+')';
  var running=list.filter(function(i){return i.status=='running';});
  document.getElementById('metric-running').textContent=running.length;
  var bestTps=0;
  running.forEach(function(i){if(i.tokens_per_sec&&i.tokens_per_sec>bestTps)bestTps=i.tokens_per_sec;});
  document.getElementById('metric-tps').textContent=bestTps?bestTps.toFixed(1):'—';

  cs.innerHTML='<option value="">— select a running instance —</option>';
  list.forEach(function(i){var mn=i.model||'?';cs.innerHTML+='<option value="'+i.port+'"'+(chatPort==i.port?' selected':'')+'>'+i.port+' - '+(mn.length>35?mn.slice(0,35)+'...':mn)+'</option>';});
  if(!list.length){document.getElementById('chatPanel').style.display='none';document.getElementById('chatEmpty').style.display='flex';c.innerHTML='<div class="empty-state"><div class="icon">🚀</div><div>No running instances. Launch one from the Dashboard.</div></div>';return;}
  document.getElementById('chatPanel').style.display=chatPort?'block':'none';

  c.innerHTML=list.map(function(i){
    var cls=i.status=='running'?'':' stopped';
    var bc=i.status=='running'?'badge-green':'badge-red';
    var mn=i.model||'?';
    var tps=i.tokens_per_sec?'<span style="color:var(--green)">⚡ '+i.tokens_per_sec.toFixed(1)+' t/s</span>':'';
    var errDiv=i.status!='running'?'<div class="error-line" id="err-'+i.port+'"></div>':'';
    return '<div class="inst-card'+cls+'"><div class="title">'+(mn.length>40?mn.slice(0,40)+'...':mn)+'</div>'+
      '<div class="meta"><span>Port '+i.port+'</span><span>PID '+i.pid+'</span><span class="badge '+bc+'">'+i.status+'</span>'+tps+'</div>'+
      errDiv+
      '<div class="actions"><button class="small danger" onclick="stopInstance('+i.port+')">⏹ Stop</button>'+
      '<button class="small secondary" onclick="selectChatFor('+i.port+',\''+mn.replace(/\'/g,'')+'\')">💬 Chat</button>'+
      '<button class="small secondary" onclick="window.open(\'http://\'+location.hostname+\':'+i.port+'\',\'_blank\')">🌐 Open</button>'+
      '<button class="small secondary" onclick="viewLogs('+i.port+')">📋 Logs</button></div></div>';
  }).join('');
  // Fetch error logs for stopped instances
  list.forEach(function(i){
    if(i.status!='running')fetchErrorLog(i.port);
  });
}

async function launchInstance(){
  var btn=document.getElementById('launchBtn'),m=document.getElementById('modelSelect').value,p=parseInt(document.getElementById('portInput').value),f=[];
  document.querySelectorAll('.flag-input').forEach(function(el){(el.value.trim().split(/\s+/)).forEach(function(v){if(v)f.push(v);});});
  if(!m){alert('Select a model');return;}
  var orig=btn.textContent;btn.disabled=true;btn.textContent='Launching...';
  try{
    var r=await fetch('/api/v1/instances',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({model:m,port:p,flags:f})});
    if(!r.ok){var e=await r.text();alert('Error: '+e);return;}
    var i=await r.json();
    document.getElementById('portInput').value=(i.port||0)+1;
    loadInstances();
  }finally{btn.disabled=false;btn.textContent=orig;}
}

async function stopInstance(p){
  if(!confirm('Stop instance on port '+p+'?'))return;
  await fetch('/api/v1/instances/stop?port='+p,{method:'POST'});
  loadInstances();
  if(chatPort==p){chatPort=0;document.getElementById('chatPanel').style.display='none';document.getElementById('chatEmpty').style.display='flex';}
}

// ── Error Log ──────────────────────────────────────────
async function fetchErrorLog(port){
  try{
    var r=await fetch('/api/v1/instances/logs?port='+port),d=await r.json();
    if(d.error||!d.lines||!d.lines.length)return;
    var el=document.getElementById('err-'+port);
    if(!el)return;
    for(var i=d.lines.length-1;i>=0;i--){
      var l=d.lines[i].trim();
      if(l&&!l.includes('\r')){el.textContent='⚠️ '+l;break;}
    }
  }catch(e){}
}

// ── Flags ─────────────────────────────────────────────
function addFlag(){
  var c=document.getElementById('flagsContainer'),r=document.createElement('div');
  r.className='flag-row';
  r.innerHTML='<input type="text" placeholder="e.g. --flash-attn on" class="flag-input"><button class="small danger" onclick="this.parentElement.remove()">✕</button>';
  c.appendChild(r);
}

// ── Pull Model ────────────────────────────────────────
async function pullModel(){
  var ref=document.getElementById('pullInput').value.trim();
  if(!ref){alert('Enter a model reference');return;}
  var btn=document.getElementById('pullBtn'),st=document.getElementById('pullStatus');
  btn.disabled=true;btn.textContent='Pulling...';st.textContent='Downloading...';
  try{
    var r=await fetch('/api/v1/models/pull',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({model:ref})});
    var d=await r.json();
    if(d.error){st.textContent='Error: '+d.error;alert(d.error);}else{st.innerHTML='✅ Pulled '+ref;loadModels();}
  }catch(e){st.textContent='Error: '+e;alert(e);}
  btn.disabled=false;btn.textContent='Pull';
}

// ── Chat ──────────────────────────────────────────────
function selectChatInstance(){
  var s=document.getElementById('chatInstanceSelect');chatPort=parseInt(s.value)||0;
  if(chatPort){chatHistory=[];document.getElementById('chatPanel').innerHTML='';document.getElementById('chatPanel').style.display='block';document.getElementById('chatEmpty').style.display='none';addSystemMsg('Connected');}
}
function selectChatFor(port,model){
  chatPort=port;chatHistory=[];
  document.getElementById('chatInstanceSelect').value=port;
  document.getElementById('chatPanel').innerHTML='';
  document.getElementById('chatPanel').style.display='block';
  document.getElementById('chatEmpty').style.display='none';
  addSystemMsg('Chatting with '+(model||'port '+port));
  if(currentView!='chat')switchView('chat');
}
function addSystemMsg(t){var c=document.getElementById('chatPanel');c.innerHTML+='<div class="msg system">'+t+'</div>';c.scrollTop=c.scrollHeight;}
function addMsg(r,t,re){
  var c=document.getElementById('chatPanel');
  var el=document.createElement('div');el.className='msg '+r;
  if(re){el.insertAdjacentHTML('beforebegin','<div class="reasoning">'+escHtml(re)+'</div>');}
  el.textContent=t;c.appendChild(el);c.scrollTop=c.scrollHeight;return el;
}
function escHtml(s){return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');}

async function sendChat(){
  var input=document.getElementById('chatInput'),msg=input.value.trim();
  if(!msg||!chatPort)return;
  input.value='';addMsg('user',msg);chatHistory.push({role:'user',content:msg});
  var li=addMsg('assistant','');
  li.innerHTML='<span class="chat-loading">● ● ●</span>';
  try{
    var r=await fetch('/api/v1/chat?port='+chatPort,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({model:'default',messages:chatHistory.slice(-20),max_tokens:256,stream:false})});
    var d=await r.json(),msg=d.choices&&d.choices[0]&&d.choices[0].message?d.choices[0].message:{},reply=msg.content||'(no response)',reasoning=msg.reasoning_content||'';
    chatHistory.push({role:'assistant',content:reply});
    li.innerHTML='';li.textContent=reply;
    if(reasoning){li.insertAdjacentHTML('beforebegin','<div class="reasoning">'+escHtml(reasoning)+'</div>');}
  }catch(e){li.innerHTML='Error: '+e.message;li.className='msg system';}
}

// ── Logs ──────────────────────────────────────────────
async function viewLogs(port){
  var r=await fetch('/api/v1/instances/logs?port='+port),d=await r.json();
  if(d.error){alert('No logs');return;}
  document.getElementById('logContent').textContent=d.lines&&d.lines.length?d.lines.slice(-50).join('\n'):'(empty)';
  document.getElementById('logModal').style.display='block';
}
function closeLogs(){document.getElementById('logModal').style.display='none';}

// ── Helpers ───────────────────────────────────────────
function fmtSize(b){
  if(!b)return'?';
  if(b>1073741824)return(b/1073741824).toFixed(1)+' GB';
  if(b>1048576)return(b/1048576).toFixed(0)+' MB';
  return(b/1024).toFixed(0)+' KB';
}

// ── Sidebar ──────────────────────────────────────────
function toggleSidebar(){
  document.querySelector('.sidebar').classList.toggle('collapsed');
  var btn=document.querySelector('.sidebar .toggle');
  btn.textContent=document.querySelector('.sidebar').classList.contains('collapsed')?'▶':'◀';
  localStorage.setItem('gollama-sidebar',document.querySelector('.sidebar').classList.contains('collapsed')?'collapsed':'');
}
(function(){
  if(localStorage.getItem('gollama-sidebar')==='collapsed'){
    document.querySelector('.sidebar').classList.add('collapsed');
    document.querySelector('.sidebar .toggle').textContent='▶';
  }
})();

// ── Theme ─────────────────────────────────────────────
function toggleTheme(){
  var b=document.body;
  b.classList.toggle('light');
  document.getElementById('themeToggle').textContent=b.classList.contains('light')?'☀️':'🌙';
  localStorage.setItem('gollama-theme',b.classList.contains('light')?'light':'dark');
}
(function(){
  if(localStorage.getItem('gollama-theme')==='light'){document.body.classList.add('light');document.getElementById('themeToggle').textContent='☀️';}
})();

// ── Init (staggered, no pile-up) ─────────────────────
loadModels();
setTimeout(loadInstances,100);
setTimeout(function tick(){
  loadInstances();
  setTimeout(tick,3000);
},2000);
setTimeout(function tick(){
  loadModels();
  setTimeout(tick,10000);
},5000);
</script>
</body>
</html>`
