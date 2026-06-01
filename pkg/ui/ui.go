package ui

const Page = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>gollama</title>
<style>
:root { --bg:#0a0a0a; --surface:#1a1a1a; --border:#2a2a2a; --text:#e5e5e5; --muted:#888; --accent:#7c3aed; --accent-hover:#6d28d9; --green:#22c55e; --red:#ef4444; --header-text:#a78bfa; --card-title:#c4b5fd; --input-bg:#0a0a0a; --select-bg:#1a1a1a; --hover-bg:#222; --badge-green-bg:#064e3b; --badge-green-text:#34d399; --badge-red-bg:#450a0a; --badge-red-text:#f87171; --badge-blue-bg:#1a1a3a; --badge-blue-text:#818cf8; --chat-user-bg:#1e293b; }
.light { --bg:#f5f5f5; --surface:#fff; --border:#ddd; --text:#1a1a1a; --muted:#777; --accent:#7c3aed; --accent-hover:#6d28d9; --green:#16a34a; --red:#dc2626; --header-text:#6d28d9; --card-title:#5b21b6; --input-bg:#f5f5f5; --select-bg:#fff; --hover-bg:#eee; --badge-green-bg:#dcfce7; --badge-green-text:#166534; --badge-red-bg:#fee2e2; --badge-red-text:#991b1b; --badge-blue-bg:#e0e7ff; --badge-blue-text:#4338ca; --chat-user-bg:#e0e7ff; }
* { margin:0; padding:0; box-sizing:border-box; }
body { font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif; background:var(--bg); color:var(--text); padding:20px; max-width:1400px; margin:0 auto; transition:background .2s,color .2s; }
h1 { color:var(--header-text); margin-bottom:4px; display:inline-block; font-size:22px; }
.subtitle { color:var(--muted); font-size:13px; margin-bottom:20px; }
h2 { color:var(--card-title); margin:0 0 12px 0; font-size:14px; display:flex; align-items:center; gap:8px; }
.card { background:var(--surface); border-radius:10px; padding:16px; margin-bottom:16px; border:1px solid var(--border); }
.card-row { display:flex; gap:16px; flex-wrap:wrap; }
.card-row .card { flex:1; min-width:300px; }
.label { font-size:11px; color:var(--muted); text-transform:uppercase; letter-spacing:.5px; margin-bottom:4px; }
select, input, button { width:100%; padding:8px 12px; background:var(--input-bg); border:1px solid var(--border); border-radius:8px; color:var(--text); font-size:13px; margin-bottom:8px; outline:none; transition:border-color .2s; }
select:focus, input:focus { border-color:var(--accent); }
select option { background:var(--select-bg); }
button { background:linear-gradient(135deg,var(--accent),var(--accent-hover)); border:none; cursor:pointer; font-weight:600; font-size:13px; transition:all .2s; padding:8px 16px; border-radius:8px; color:#fff; }
button:hover { transform:translateY(-1px); box-shadow:0 4px 14px rgba(124,58,237,.3); }
button.secondary { background:var(--border); color:var(--text); box-shadow:none; }
button.secondary:hover { transform:none; background:var(--hover-bg); }
button.danger { background:linear-gradient(135deg,var(--red),#b91c1c); }
button.danger:hover { box-shadow:0 4px 14px rgba(220,38,38,.3); }
button.small { width:auto; padding:4px 10px; font-size:11px; border-radius:6px; }
#themeToggle { width:36px; height:36px; padding:0; font-size:16px; border-radius:50%; display:flex; align-items:center; justify-content:center; float:right; cursor:pointer; background:var(--border); color:var(--text); border:1px solid var(--border); }
#themeToggle:hover { background:var(--hover-bg); transform:none; box-shadow:none; }
.flag-row { display:flex; gap:8px; margin-bottom:4px; }
.flag-row input { flex:1; margin-bottom:0; }
.flag-row button { width:auto; }
.mt-8 { margin-top:8px; }
.text-sm { font-size:12px; color:var(--muted); }
.text-xs { font-size:11px; color:var(--muted); }
.flex { display:flex; justify-content:space-between; align-items:center; }
.badge { display:inline-block; padding:2px 8px; border-radius:6px; font-size:10px; font-weight:700; text-transform:uppercase; letter-spacing:.3px; }
.badge-green { background:var(--badge-green-bg); color:var(--badge-green-text); }
.badge-red { background:var(--badge-red-bg); color:var(--badge-red-text); }
.badge-blue { background:var(--badge-blue-bg); color:var(--badge-blue-text); }
.grid { display:grid; grid-template-columns:repeat(auto-fill,minmax(320px,1fr)); gap:12px; }
.inst-card { border-left:4px solid var(--green); padding:14px; background:var(--surface); border-radius:0 10px 10px 0; border:1px solid var(--border); border-left-width:4px; transition:all .2s; }
.inst-card:hover { border-color:var(--hover-bg); }
.inst-card.stopped { border-left-color:var(--red); opacity:.6; }
.inst-card .title { font-weight:600; font-size:13px; word-break:break-all; }
.inst-card .meta { font-size:11px; color:var(--muted); margin-top:6px; display:flex; gap:8px; flex-wrap:wrap; align-items:center; }
.inst-card .actions { margin-top:10px; display:flex; gap:6px; flex-wrap:wrap; }
.chat-msgs { flex:1; overflow-y:auto; padding:12px; background:var(--bg); border-radius:8px; margin-bottom:8px; font-size:13px; line-height:1.6; }
.chat-msgs .msg { margin-bottom:10px; padding:8px 12px; border-radius:10px; max-width:85%; line-height:1.5; }
.chat-msgs .user { background:var(--chat-user-bg); margin-left:auto; border-bottom-right-radius:4px; }
.chat-msgs .assistant { background:var(--surface); border:1px solid var(--border); border-bottom-left-radius:4px; }
.chat-msgs .system { background:transparent; color:var(--muted); font-style:italic; font-size:11px; text-align:center; max-width:100%; }
.chat-input-row { display:flex; gap:8px; }
.chat-input-row input { flex:1; margin-bottom:0; }
.chat-input-row button { width:auto; padding:8px 20px; }
.empty-state { text-align:center; padding:40px 20px; color:var(--muted); }
.empty-state .icon { font-size:36px; margin-bottom:10px; }
.chat-panel { display:none; }
.chat-panel.active { display:flex; flex-direction:column; height:400px; }
.instance-selector { display:flex; gap:8px; align-items:center; margin-bottom:12px; }
.instance-selector select { margin-bottom:0; }
.model-row { display:flex; justify-content:space-between; align-items:center; padding:8px 10px; border-radius:6px; transition:background .2s; }
.model-row:hover { background:var(--hover-bg); }
.model-row .name { font-size:13px; color:var(--text); }
.model-row .info { font-size:11px; color:var(--muted); margin-top:2px; }
@keyframes pulse { 0%,100%{opacity:1} 50%{opacity:.5} }
@keyframes spin { to{transform:rotate(360deg)} }
.loading { animation:pulse 1.5s infinite; }
.chat-loading { animation:pulse 1.2s infinite; display:inline-block; letter-spacing:4px; font-size:18px; line-height:1; color:var(--muted); }
.spinner { display:inline-block; width:14px; height:14px; border:2px solid var(--border); border-top-color:var(--accent); border-radius:50%; animation:spin .6s linear infinite; vertical-align:middle; margin-right:6px; }
.refreshing { opacity:.5; pointer-events:none; transition:opacity .2s; }
@media(max-width:768px){.card-row{flex-direction:column}.grid{grid-template-columns:1fr}}
</style>
</head>
<body>
<div class="flex"><h1>gollama</h1> <button id="themeToggle" onclick="toggleTheme()" title="Toggle theme">🌙</button></div>
<div class="subtitle">llama.cpp instance manager</div>

<div class="card-row">
  <div class="card">
    <h2>📥 Pull Model</h2>
    <input type="text" id="pullInput" placeholder="hf.co/user/repo:Q4_K_M" value="hf.co/unsloth/gemma-4-E2B-it-GGUF:Q4_K_M">
    <button onclick="pullModel()" id="pullBtn">Pull</button>
    <div id="pullStatus" class="text-sm" style="margin-top:4px"></div>
  </div>

  <div class="card">
    <h2>🚀 New Instance</h2>
    <div class="label">Model</div>
    <select id="modelSelect"><option value="">Loading models...</option></select>
    <div class="label">Port</div>
    <input type="number" id="portInput" value="8081" min="8081" max="8099">
    <div class="label">Flags</div>
    <div id="flagsContainer">
      <div class="flag-row">
        <input type="text" placeholder="e.g. --tensor-split 12,8" class="flag-input">
        <button class="small danger" onclick="this.parentElement.remove()">x</button>
      </div>
    </div>
    <button class="secondary small" onclick="addFlag()">+ Add Flag</button>
    <button class="mt-8" onclick="launchInstance()">Launch</button>
  </div>
</div>

<div class="card-row">
  <div class="card" style="flex:1">
    <h2>📦 Models <span id="modelCount" class="text-sm" style="font-weight:400"></span></h2>
    <div id="modelList"><div class="text-sm">Loading...</div></div>
  </div>

  <div class="card" style="flex:1">
    <h2>🟢 Running <span id="instanceCount" class="text-sm" style="font-weight:400"></span></h2>
    <div id="instances" class="grid"><div class="text-sm">Loading...</div></div>
  </div>

  <div class="card" style="flex:1">
    <h2>💬 Chat</h2>
    <div class="instance-selector">
      <select id="chatInstanceSelect" onchange="selectChatInstance()"><option value="">— select running instance —</option></select>
      <button class="small secondary" onclick="refreshChat()">↻</button>
    </div>
    <div id="chatPanel" class="chat-panel">
      <div id="chatMsgs" class="chat-msgs"></div>
      <div class="chat-input-row">
        <input type="text" id="chatInput" placeholder="Type a message..." onkeydown="if(event.key=='Enter')sendChat()">
        <button onclick="sendChat()">Send</button>
      </div>
    </div>
    <div id="chatEmpty" class="empty-state">
      <div class="icon">💬</div>
      <div>Launch an instance to start chatting</div>
    </div>
  </div>
</div>

<script>
var chatPort=0,chatHistory=[];

// ── Models — single fetch for both selector + list ─────────────
async function loadModels(){
  var mc=document.getElementById('modelCount'),c=document.getElementById('modelList'),s=document.getElementById('modelSelect');
  mc.innerHTML='<span class="spinner"></span>';
  c.classList.add('refreshing');
  var r=await fetch('/api/v1/models'),m=await r.json();
  c.classList.remove('refreshing');
  mc.textContent='('+m.length+')';

  s.innerHTML='<option value="">— Select model —</option>';
  if(!m||!m.length){s.innerHTML+='<option value="" disabled>No models found. Use gollama pull.</option>';c.innerHTML='<div class="text-sm">No models downloaded</div>';return;}
  var seen={};
  m.forEach(function(x){
    var n=x.name||'(unnamed)',src=x.source||'unknown';
    if(!seen[n]){seen[n]=1;s.innerHTML+='<option value="'+n+'">'+n+' ['+src+']</option>';}
  });
  c.innerHTML=m.map(function(x){
    var name=x.name||'?',size=x.size?fmtSize(x.size):'?';
    var arch=x.architecture||'',quant=x.quantization||'',ctx=x.context_length||0,badges=[];
    if(quant)badges.push('<span class="badge badge-blue">'+quant+'</span>');
    if(arch)badges.push('<span class="badge badge-blue">'+arch+'</span>');
    if(ctx)badges.push('<span class="badge badge-green">'+(ctx>999?Math.round(ctx/1000)+'K':'<1K')+' ctx</span>');
    return '<div class="model-row"><div><div class="name">'+(name.length>50?name.slice(0,50)+'...':name)+'</div><div class="info">'+size+' | '+(badges.length?badges.join(' '):'[no metadata]')+' ['+(x.source||'?')+']</div></div>'+
      '<button class="small danger" onclick="deleteModel(\''+name.replace(/\'/g,'')+'\')">🗑</button></div>';
  }).join('');
}

// ── Instances — single fetch for cards + chat selector ─────────
async function loadInstances(){
  var ic=document.getElementById('instanceCount'),c=document.getElementById('instances'),cs=document.getElementById('chatInstanceSelect');
  ic.innerHTML='<span class="spinner"></span>';
  c.classList.add('refreshing');
  var r=await fetch('/api/v1/instances'),list=await r.json();
  c.classList.remove('refreshing');
  ic.textContent='('+list.length+')';

  // Update chat selector
  cs.innerHTML='<option value="">— select running instance —</option>';
  list.forEach(function(i){var mn=i.model||'?';cs.innerHTML+='<option value="'+i.port+'"'+(chatPort==i.port?' selected':'')+'>'+i.port+' - '+(mn.length>35?mn.slice(0,35)+'...':mn)+'</option>';});
  if(!list.length){document.getElementById('chatPanel').classList.remove('active');document.getElementById('chatEmpty').style.display='block';}

  // Render instance cards
  if(!list.length){c.innerHTML='<div class="text-sm">No running instances</div>';return;}
  c.innerHTML=list.map(function(i){
    var sc=i.status=='running'?'':' stopped';
    var bc=i.status=='running'?'badge-green':'badge-red';
    var mn=i.model||'?';
    var tps=i.tokens_per_sec?'<span style="color:#22c55e">⚡ '+i.tokens_per_sec.toFixed(1)+' t/s</span>':'';
    return '<div class="inst-card'+sc+'"><div class="title">'+(mn.length>40?mn.slice(0,40)+'...':mn)+'</div>'+
      '<div class="meta">Port: '+i.port+' | PID: '+i.pid+' | <span class="badge '+bc+'">'+i.status+'</span> '+tps+'</div>'+
      '<div class="actions"><button class="small danger" onclick="stopInstance('+i.port+')">⏹ Stop</button>'+
      '<button class="small secondary" onclick="selectChatFor('+i.port+',\''+mn.replace(/\'/g,'')+'\')">💬 Chat</button>'+
      '<button class="small secondary" onclick="window.open(\'http://\'+location.hostname+\':'+i.port+'\',\'_blank\')">🌐 UI</button>'+
      '<button class="small secondary" onclick="viewLogs('+i.port+')">📋 Logs</button></div></div>';
  }).join('');
}

function fmtSize(b){
  if(!b)return'?';
  if(b>1073741824)return(b/1073741824).toFixed(1)+' GB';
  if(b>1048576)return(b/1048576).toFixed(0)+' MB';
  return(b/1024).toFixed(0)+' KB';
}

async function deleteModel(name){
  if(!confirm('Delete model "'+name+'"?'))return;
  await fetch('/api/v1/models/delete',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name:name})});
  loadModels();
}

async function launchInstance(){
  var btn=document.querySelector('.card:nth-child(2) .mt-8'),m=document.getElementById('modelSelect').value,p=parseInt(document.getElementById('portInput').value),f=[];
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
  if(chatPort==p){document.getElementById('chatPanel').classList.remove('active');document.getElementById('chatEmpty').style.display='block';}
}

function addFlag(){
  var c=document.getElementById('flagsContainer'),r=document.createElement('div');
  r.className='flag-row';
  r.innerHTML='<input type="text" placeholder="e.g. --tensor-split 12,8" class="flag-input"><button class="small danger" onclick="this.parentElement.remove()">x</button>';
  c.appendChild(r);
}

function selectChatInstance(){
  var s=document.getElementById('chatInstanceSelect');chatPort=parseInt(s.value)||0;
  if(chatPort){chatHistory=[];document.getElementById('chatMsgs').innerHTML='';document.getElementById('chatPanel').classList.add('active');document.getElementById('chatEmpty').style.display='none';addSystemMsg('Connected');}
}

function selectChatFor(port,model){
  chatPort=port;chatHistory=[];
  document.getElementById('chatInstanceSelect').value=port;
  document.getElementById('chatMsgs').innerHTML='';
  document.getElementById('chatPanel').classList.add('active');
  document.getElementById('chatEmpty').style.display='none';
  addSystemMsg('Chatting with '+(model||'port '+port));
}

function addSystemMsg(t){var c=document.getElementById('chatMsgs');c.innerHTML+='<div class="msg system">'+t+'</div>';c.scrollTop=c.scrollHeight;}
function addMsg(r,t,re){
  var c=document.getElementById('chatMsgs');
  var el=document.createElement('div');el.className='msg '+r;
  if(re){var r2=re.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');var rd=document.createElement('div');rd.style.cssText='color:var(--muted);font-style:italic;font-size:12px;border-left:2px solid var(--border);padding-left:8px;margin-bottom:4px';rd.innerHTML=r2;c.appendChild(rd);}
  if(t.indexOf('<')===-1&&t.indexOf('&')===-1){el.textContent=t;}else{el.innerHTML=t;}
  c.appendChild(el);c.scrollTop=c.scrollHeight;return el;
}

async function sendChat(){
  var input=document.getElementById('chatInput'),msg=input.value.trim();
  if(!msg||!chatPort)return;
  input.value='';addMsg('user',msg);chatHistory.push({role:'user',content:msg});
  var li=addMsg('assistant','<span class="chat-loading">● ● ●</span>');
  try{
    var r=await fetch('/api/v1/chat?port='+chatPort,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({model:'default',messages:chatHistory.slice(-20),max_tokens:256,stream:false})});
    var d=await r.json(),msg=d.choices&&d.choices[0]&&d.choices[0].message?d.choices[0].message:{},reply=msg.content||'(no response)',reasoning=msg.reasoning_content||'';
    chatHistory.push({role:'assistant',content:reply});
    li.innerHTML=reply;li.className='msg assistant';
    if(reasoning){var r2=reasoning.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');li.insertAdjacentHTML('beforebegin','<div style="color:var(--muted);font-style:italic;font-size:12px;border-left:2px solid var(--border);padding-left:8px;margin-bottom:4px">'+r2+'</div>');}
  }catch(e){li.innerHTML='Error: '+e.message;li.className='msg system';}
}

async function viewLogs(port){
  var r=await fetch('/api/v1/instances/logs?port='+port),d=await r.json();
  if(d.error){alert('No logs');return;}
  document.getElementById('logContent').textContent=d.lines&&d.lines.length?d.lines.slice(-50).join('\\n'):'(empty)';
  document.getElementById('logModal').style.display='block';
}
function closeLogs(){document.getElementById('logModal').style.display='none';}

async function pullModel(){
  var ref=document.getElementById('pullInput').value.trim();
  if(!ref){alert('Enter a model reference');return;}
  var btn=document.getElementById('pullBtn'),st=document.getElementById('pullStatus');
  btn.disabled=true;btn.textContent='Pulling...';st.textContent='Downloading...';
  try{
    var r=await fetch('/api/v1/models/pull',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({model:ref})});
    var d=await r.json();
    if(d.error){st.textContent='Error: '+d.error;alert(d.error);}else{st.textContent='✅ '+ref;loadModels();}
  }catch(e){st.textContent='Error: '+e;alert(e);}
  btn.disabled=false;btn.textContent='Pull';
}

function refreshChat(){if(chatPort)selectChatFor(chatPort,'');}

function toggleTheme(){
  var b=document.body;
  b.classList.toggle('light');
  document.getElementById('themeToggle').textContent=b.classList.contains('light')?'☀️':'🌙';
  localStorage.setItem('gollama-theme',b.classList.contains('light')?'light':'dark');
}
(function(){
  if(localStorage.getItem('gollama-theme')==='light'){document.body.classList.add('light');document.getElementById('themeToggle').textContent='☀️';}
})();

loadModels();loadInstances();
setInterval(function(){loadInstances();},3000);
setInterval(function(){loadModels();},10000);
</script>

<div id="logModal" style="display:none;position:fixed;top:0;left:0;width:100%;height:100%;background:rgba(0,0,0,.7);z-index:1000">
  <div style="background:var(--surface);margin:5% auto;padding:20px;width:80%;max-width:700px;max-height:70vh;border-radius:10px;overflow:auto;border:1px solid var(--border)">
    <div class="flex"><h2 style="margin-bottom:0">📋 Logs</h2><button class="small danger" onclick="closeLogs()">Close</button></div>
    <pre id="logContent" style="background:var(--input-bg);padding:12px;border-radius:6px;margin-top:12px;font-size:11px;line-height:1.4;overflow:auto;max-height:55vh;white-space:pre-wrap;color:var(--muted)"></pre>
  </div>
</div>
</body>
</html>`
