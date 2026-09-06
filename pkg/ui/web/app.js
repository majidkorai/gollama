var chatPort = 0, chatHistory = [], chatSessionId = null;
var chatAbortController = null; // T4: in-flight chat stream controller
var currentView = 'dashboard';
var cachedModelCount = 0;
var cachedModels = [];
var imageHistory = [];
var imageGenerating = false;

// ── API auth (v3.8.0) ─────────────────────────────────
// All API calls go through apiFetch, which attaches the stored token as
// Authorization: Bearer. A 401 raises the token gate. UI assets are open;
// only /api/v1/* and /v1/* require the token.
function getToken() {
  try { return localStorage.getItem('gollama_token') || ''; } catch (e) { return ''; }
}
function setToken(tok) {
  try {
    if (tok) localStorage.setItem('gollama_token', tok);
    else localStorage.removeItem('gollama_token');
  } catch (e) {}
}
async function apiFetch(url, opts) {
  opts = opts || {};
  opts.headers = Object.assign({}, opts.headers || {});
  var tok = getToken();
  if (tok) opts.headers['Authorization'] = 'Bearer ' + tok;
  var res = await fetch(url, opts);
  if (res.status === 401) showTokenGate();
  return res;
}
function showTokenGate() {
  var g = document.getElementById('tokenGate');
  if (g) g.style.display = 'flex';
}
function submitTokenGate() {
  var input = document.getElementById('tokenGateInput');
  var tok = (input.value || '').trim();
  if (!tok) return;
  setToken(tok);
  location.reload();
}
function copyToken() {
  var el = document.getElementById('apiTokenValue');
  if (!el || el.textContent === '—') return;
  fallbackCopy(el.textContent);
}
function copyOpenaiEndpoint() {
  var el = document.getElementById('openaiEndpoint');
  if (!el || el.textContent === '—') return;
  fallbackCopy(el.textContent);
}
async function regenerateToken() {
  if (!confirm('Regenerate the API token? Existing clients (cron jobs, agents) will need the new token.')) return;
  var r = await apiFetch('/api/v1/config/token', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ action: 'regenerate' }) });
  if (r.ok) {
    var d = await r.json();
    setToken(d.api_token);
    var el = document.getElementById('apiTokenValue');
    if (el) el.textContent = d.api_token;
  }
}
async function clearToken() {
  if (!confirm('Disable API authentication? Anyone who can reach this port can then control gollama.')) return;
  var r = await apiFetch('/api/v1/config/token', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ action: 'clear' }) });
  if (r.ok) {
    setToken('');
    var el = document.getElementById('apiTokenValue');
    if (el) el.textContent = '—';
  }
}

// ── Navigation ──────────────────────────────────────
function switchView(name) {
  document.querySelectorAll('.view').forEach(function(v) { v.classList.remove('active'); });
  document.querySelectorAll('.nav-item').forEach(function(a) { a.classList.remove('active'); });
  var view = document.getElementById('view-' + name);
  if (view) view.classList.add('active');
  document.querySelector('.nav-item[data-view="' + name + '"]').classList.add('active');
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
async function loadModels(refresh) {
  var mc = document.getElementById('modelCount'), ml = document.getElementById('modelList'), ms = document.getElementById('modelCountSpinner');
  var s = document.getElementById('modelSelect');
  if (ms) ms.style.display = 'inline-block';
  if (mc) mc.innerHTML = '<span class="spinner"></span> Loading…';
  ml.classList.add('refreshing');
  try {
    var r = await apiFetch('/api/v1/models' + (refresh ? '?refresh=1' : '')), m = await r.json();
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
      return '<div class="model-row" onclick="showModelDetails(\'' + escAttr(name.replace(/'/g, '')) + '\')"><div><div class="name">' + escHtml(name.length > 55 ? name.slice(0, 55) + '…' : name) + ' <span class="info-icon">ⓘ</span></div><div class="info">' + size + ' ' + (badges.length ? badges.join(' ') : '') + '</div></div><button class="small danger" onclick="event.stopPropagation();deleteModel(\'' + escAttr(name.replace(/'/g, '')) + '\')" aria-label="Delete ' + escAttr(name) + '"><span class="icon">🗑</span></button></div>';
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
    await apiFetch('/api/v1/models/delete', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name: name }) });
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
    var r = await apiFetch('/api/v1/instances'), list = await r.json();
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
      var gpuBadge = (i.gpu_util_per_gpu && i.gpu_util_per_gpu.length)
        ? '<span class="badge" style="background:var(--surface-2);color:var(--text-dim)" title="GPU utilization per device">' + i.gpu_util_per_gpu.map(function(u, k) { return 'GPU' + k + ' ' + Math.round(u) + '%'; }).join(' / ') + '</span>'
        : (i.gpu_util > 0 ? '<span class="badge" style="background:var(--surface-2);color:var(--text-dim)" title="GPU utilization">GPU ' + Math.round(i.gpu_util) + '%</span>' : '');
var metrics = typeLabel + gpuBadge + (i.device_split ? '<span class="badge" style="background:var(--surface-2);color:var(--text-dim)" title="Model split">📊 ' + i.device_split + '</span>' : '') + (i.profile ? ' <span class="badge badge-profile" title="Active model profile">📋 ' + escHtml(i.profile) + '</span>' : '');
      var flags = i.flags && i.flags.length ? formatFlags(i.flags) : '';
      var flagsHtml = flags ? '<div style="font-size: 11px; color: var(--text-dim); margin-top: 8px; padding-top: 8px; border-top: 1px solid var(--border); word-break: break-all; font-family: var(--font-mono)">' + escHtml(flags) + '</div>' : '';
      var errDiv = i.status != 'running' ? '<div class="error-line" id="err-' + i.port + '"></div>' : '';
      return '<div class="inst-card' + cls + '"><div class="title">' + escHtml(mn.length > 40 ? mn.slice(0, 40) + '…' : mn) + '</div>' +
        '<div class="meta"><span>Port ' + i.port + '</span>' + tps + uptime + idle + tokens + '</div>' +
        '<div class="meta meta-badges">' + metrics + '<span class="badge ' + bc + '">' + statusLabel + '</span></div>' +
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
    var r = await apiFetch('/api/v1/instances', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ model: m, port: p, flags: f, replace_flags: true }) });
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
    await apiFetch('/api/v1/instances/stop?port=' + p, { method: 'POST' });
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

  await apiFetch('/api/v1/instances/stop?port=' + port, { method: 'POST' });

  var userFlags = (inst.flags || []).slice();
  for (var k = 0; k < userFlags.length; k++) { if (userFlags[k] === '-m' || userFlags[k] === '--port') { userFlags.splice(k, 2); k--; } }

  try {
    var r = await apiFetch('/api/v1/instances', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ model: inst.model, port: inst.port, flags: userFlags, replace_flags: true }) });
    if (r.ok) { loadInstances(); }
  } catch (e) {}
}

// ── Error Log ──────────────────────────────────────────
async function fetchErrorLog(port) {
  try {
    var r = await apiFetch('/api/v1/instances/logs?port=' + port), d = await r.json();
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
// BEGIN generated flag catalog (tools/flags) — do not edit by hand
// Derived from the live llama-server binary's --help output plus per-flag
// probes (see tools/flags). To refresh after a llama.cpp upgrade:
//   llama-server --help > help.txt && <flags-probe> probe help.txt <binary> > probes.tsv
//   go run ./tools/flags parse help.txt probes.tsv
var commonFlags = [
 '--adaptive-decay', '--adaptive-target', '--agent', '--alias', '--api-key', '--api-key-file', '--api-prefix',
  '--backend-sampling', '--batch-size', '--cache-idle-slots', '--cache-list', '--cache-prompt', '--cache-ram',
  '--cache-reuse', '--cache-type-k', '--cache-type-k-draft', '--cache-type-v', '--cache-type-v-draft',
  '--chat-template', '--chat-template-file', '--chat-template-kwargs', '--check-tensors',
  '--checkpoint-min-step', '--cont-batching', '--context-shift', '--control-vector',
  '--control-vector-layer-range', '--control-vector-scaled', '--cors-credentials', '--cors-headers',
  '--cors-methods', '--cors-origins', '--cpu-mask', '--cpu-mask-batch', '--cpu-mask-batch-draft',
  '--cpu-mask-draft', '--cpu-moe', '--cpu-moe-draft', '--cpu-range', '--cpu-range-batch', '--cpu-range-draft',
  '--cpu-strict', '--cpu-strict-batch', '--cpu-strict-batch-draft', '--cpu-strict-draft', '--ctx-checkpoints',
  '--ctx-size', '--defrag-thold', '--device', '--device-draft', '--direct-io', '--docker-repo',
  '--dry-allowed-length', '--dry-base', '--dry-multiplier', '--dry-penalty-last-n', '--dry-sequence-breaker',
  '--dynatemp-exp', '--dynatemp-range', '--embd-gemma-default', '--embd-normalize', '--embedding', '--escape',
  '--fim-qwen-1.5b-default', '--fim-qwen-14b-spec', '--fim-qwen-30b-default', '--fim-qwen-3b-default',
  '--fim-qwen-7b-default', '--fim-qwen-7b-spec', '--fit', '--fit-ctx', '--fit-target', '--flash-attn',
  '--frequency-penalty', '--gpt-oss-120b-default', '--gpt-oss-20b-default', '--grammar', '--grammar-file',
  '--hf-file', '--hf-repo', '--hf-token', '--host', '--ignore-eos', '--image-max-tokens',
  '--image-min-tokens', '--jinja', '--json-schema', '--json-schema-file', '--keep', '--kv-offload',
  '--kv-unified', '--kv-unified-per-slot', '--lazy-mode', '--list-devices', '--load-mode', '--log-colors',
  '--log-disable', '--log-file', '--log-prefix', '--log-prompts-dir', '--log-timestamps', '--log-verbose',
  '--logit-bias', '--lookup-cache-dynamic', '--lookup-cache-static', '--lora', '--lora-init-without-apply',
  '--lora-scaled', '--main-gpu', '--mcp-servers-config', '--mcp-servers-json', '--media-path', '--metrics',
  '--min-p', '--mirostat', '--mirostat-ent', '--mirostat-lr', '--mlock', '--mmap', '--mmproj',
  '--mmproj-auto', '--mmproj-device', '--mmproj-offload', '--mmproj-url', '--model', '--model-url',
  '--models-autoload', '--models-dir', '--models-max', '--models-preset', '--moe-expert-cache',
  '--moe-expert-cache-inserts', '--mtmd-batch-max-tokens', '--n-cpu-ffn', '--n-cpu-moe', '--n-cpu-moe-draft',
  '--n-gpu-layers', '--n-gpu-layers-draft', '--no-agent', '--no-cache-idle-slots', '--no-cache-prompt',
  '--no-cont-batching', '--no-context-shift', '--no-cors-credentials', '--no-direct-io', '--no-escape',
  '--no-host', '--no-jinja', '--no-kv-offload', '--no-kv-unified', '--no-log-prefix', '--no-log-timestamps',
  '--no-mmap', '--no-mmproj', '--no-mmproj-auto', '--no-mmproj-offload', '--no-models-autoload',
  '--no-op-offload', '--no-perf', '--no-prefill-assistant', '--no-reasoning-preserve', '--no-repack',
  '--no-skip-chat-parsing', '--no-slots', '--no-spec-draft-backend-sampling', '--no-ui', '--no-ui-mcp-proxy',
  '--no-warmup', '--numa', '--offline', '--op-offload', '--override-kv', '--override-tensor',
  '--override-tensor-draft', '--parallel', '--path', '--perf', '--poll', '--poll-batch', '--poll-batch-draft',
  '--poll-draft', '--pooling', '--port', '--predict', '--prefill-assistant', '--presence-penalty', '--prio',
  '--prio-batch', '--prio-batch-draft', '--prio-draft', '--props', '--reasoning', '--reasoning-budget',
  '--reasoning-budget-message', '--reasoning-effort', '--reasoning-format', '--reasoning-preserve',
  '--repack', '--repeat-last-n', '--repeat-penalty', '--rerank', '--reuse-port', '--reverse-prompt',
  '--rope-freq-base', '--rope-freq-scale', '--rope-scale', '--rope-scaling', '--samplers', '--sampling-seq',
  '--seed', '--skip-chat-parsing', '--sleep-idle-seconds', '--slot-prompt-similarity', '--slot-save-path',
  '--slots', '--spec-default', '--spec-draft-backend-sampling', '--spec-draft-cpu-mask',
  '--spec-draft-cpu-mask-batch', '--spec-draft-cpu-moe', '--spec-draft-cpu-range', '--spec-draft-cpu-strict',
  '--spec-draft-cpu-strict-batch', '--spec-draft-device', '--spec-draft-hf', '--spec-draft-model',
  '--spec-draft-n-cpu-moe', '--spec-draft-n-max', '--spec-draft-n-min', '--spec-draft-ngl',
  '--spec-draft-p-min', '--spec-draft-p-split', '--spec-draft-poll', '--spec-draft-poll-batch',
  '--spec-draft-prio', '--spec-draft-prio-batch', '--spec-draft-threads', '--spec-draft-threads-batch',
  '--spec-draft-type-k', '--spec-draft-type-v', '--spec-ngram-map-k-min-hits', '--spec-ngram-map-k-size-m',
  '--spec-ngram-map-k-size-n', '--spec-ngram-map-k4v-min-hits', '--spec-ngram-map-k4v-size-m',
  '--spec-ngram-map-k4v-size-n', '--spec-ngram-mod-n-match', '--spec-ngram-mod-n-max',
  '--spec-ngram-mod-n-min', '--spec-ngram-simple-min-hits', '--spec-ngram-simple-size-m',
  '--spec-ngram-simple-size-n', '--spec-synth-len', '--spec-synth-rates', '--spec-type', '--special',
  '--split-mode', '--spm-infill', '--sse-ping-interval', '--ssl-cert-file', '--ssl-key-file',
  '--swa-checkpoints', '--swa-full', '--tags', '--temp', '--tensor-split', '--threads', '--threads-batch',
  '--threads-batch-draft', '--threads-draft', '--threads-http', '--timeout', '--tools', '--tools-runtime',
  '--top-k', '--top-n-sigma', '--top-p', '--typical-p', '--ubatch-size', '--ui', '--ui-config',
  '--ui-config-file', '--ui-mcp-proxy', '--verbose', '--verbosity', '--video-ffmpeg-dir', '--video-fps',
  '--video-timestamp-interval', '--vision-gemma-12b-default', '--vision-gemma-4b-default', '--warmup',
  '--xtc-probability', '--xtc-threshold', '--yarn-attn-factor', '--yarn-beta-fast', '--yarn-beta-slow',
  '--yarn-ext-factor', '--yarn-orig-ctx'
];
var standaloneFlags = {
  '--agent':1,
  '--backend-sampling':1,
  '--cache-idle-slots':1,
  '--cache-list':1,
  '--cache-prompt':1,
  '--check-tensors':1,
  '--cont-batching':1,
  '--context-shift':1,
  '--cors-credentials':1,
  '--cpu-moe':1,
  '--cpu-moe-draft':1,
  '--direct-io':1,
  '--embd-gemma-default':1,
  '--embedding':1,
  '--embeddings':1,
  '--escape':1,
  '--fim-qwen-1.5b-default':1,
  '--fim-qwen-14b-spec':1,
  '--fim-qwen-30b-default':1,
  '--fim-qwen-3b-default':1,
  '--fim-qwen-7b-default':1,
  '--fim-qwen-7b-spec':1,
  '--gpt-oss-120b-default':1,
  '--gpt-oss-20b-default':1,
  '--ignore-eos':1,
  '--jinja':1,
  '--kv-offload':1,
  '--kv-unified':1,
  '--list-devices':1,
  '--log-disable':1,
  '--log-prefix':1,
  '--log-timestamps':1,
  '--log-verbose':1,
  '--lora-init-without-apply':1,
  '--metrics':1,
  '--mlock':1,
  '--mmap':1,
  '--mmproj-auto':1,
  '--mmproj-offload':1,
  '--models-autoload':1,
  '--no-agent':1,
  '--no-cache-idle-slots':1,
  '--no-cache-prompt':1,
  '--no-cont-batching':1,
  '--no-context-shift':1,
  '--no-cors-credentials':1,
  '--no-direct-io':1,
  '--no-escape':1,
  '--no-host':1,
  '--no-jinja':1,
  '--no-kv-offload':1,
  '--no-kv-unified':1,
  '--no-log-prefix':1,
  '--no-log-timestamps':1,
  '--no-mmap':1,
  '--no-mmproj':1,
  '--no-mmproj-auto':1,
  '--no-mmproj-offload':1,
  '--no-models-autoload':1,
  '--no-op-offload':1,
  '--no-perf':1,
  '--no-prefill-assistant':1,
  '--no-reasoning-preserve':1,
  '--no-repack':1,
  '--no-skip-chat-parsing':1,
  '--no-slots':1,
  '--no-spec-draft-backend-sampling':1,
  '--no-ui':1,
  '--no-ui-mcp-proxy':1,
  '--no-warmup':1,
  '--no-webui':1,
  '--no-webui-mcp-proxy':1,
  '--offline':1,
  '--op-offload':1,
  '--perf':1,
  '--prefill-assistant':1,
  '--props':1,
  '--reasoning-preserve':1,
  '--repack':1,
  '--rerank':1,
  '--reranking':1,
  '--reuse-port':1,
  '--skip-chat-parsing':1,
  '--slots':1,
  '--spec-default':1,
  '--spec-draft-backend-sampling':1,
  '--spec-draft-cpu-moe':1,
  '--special':1,
  '--spm-infill':1,
  '--swa-full':1,
  '--ui':1,
  '--ui-mcp-proxy':1,
  '--verbose':1,
  '--vision-gemma-12b-default':1,
  '--vision-gemma-4b-default':1,
  '--warmup':1,
  '--webui':1,
  '--webui-mcp-proxy':1
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
  '--cache-list': 'show list of models in cache and exit',
  '--cache-prompt': 'enable prompt caching',
  '--cache-ram': 'max cache size in MiB (default 8192)',
  '--cache-reuse': 'min chunk size to reuse from cache',
  '--cache-type-k': 'f32, f16, bf16, q8_0, q4_0, ...',
  '--cache-type-k-draft': 'KV cache data type K for draft (f16, q8_0, ...)',
  '--cache-type-v': 'f32, f16, bf16, q8_0, q4_0, ...',
  '--cache-type-v-draft': 'KV cache data type V for draft (f16, q8_0, ...)',
  '--chat-template': 'jinja chat template name',
  '--chat-template-file': 'path to jinja chat template file',
  '--chat-template-kwargs': 'extra JSON params for template parser',
  '--check-tensors': 'check model tensor data for invalid values',
  '--checkpoint-min-step': 'minimum spacing between context checkpoints (default 256)',
  '--cont-batching': 'enable continuous batching',
  '--context-shift': 'enable context shift for infinite text gen',
  '--control-vector': 'path to control vector',
  '--control-vector-layer-range': 'START END layer range for control vector',
  '--control-vector-scaled': 'FNAME:SCALE control vector with scaling',
  '--cors-credentials': 'allow credentials for CORS (default: on)',
  '--cors-headers': 'comma-separated CORS allowed headers (default *)',
  '--cors-methods': 'comma-separated CORS allowed methods (default GET, POST, DELETE, OPTIONS)',
  '--cors-origins': 'comma-separated CORS allowed origins (default *); "localhost" reflects only local Origins',
  '--cpu-mask': 'CPU affinity mask (hex)',
  '--cpu-mask-batch': 'CPU affinity mask for batch processing (hex)',
  '--cpu-mask-batch-draft': 'draft batch CPU affinity mask (hex)',
  '--cpu-mask-draft': 'draft CPU affinity mask (hex)',
  '--cpu-moe': 'keep MoE weights in CPU',
  '--cpu-moe-draft': 'keep MoE weights for draft in CPU',
  '--cpu-range': 'CPU range for affinity (lo-hi)',
  '--cpu-range-batch': 'CPU range for batch affinity (lo-hi)',
  '--cpu-range-draft': 'draft CPU range (lo-hi)',
  '--cpu-strict': '<0|1> strict CPU placement',
  '--cpu-strict-batch': '<0|1> strict CPU placement for batch',
  '--cpu-strict-batch-draft': '<0|1> strict CPU placement for draft batch',
  '--cpu-strict-draft': '<0|1> strict CPU placement for draft',
  '--ctx-checkpoints': 'max context checkpoints per slot (default 32)',
  '--ctx-size': 'context size in tokens',
  '--defrag-thold': 'DEPRECATED: KV cache defragmentation threshold',
  '--device': 'comma-separated devices for offloading',
  '--device-draft': 'devices for draft model offloading',
  '--direct-io': 'DEPRECATED: use --load-mode dio — use DirectIO if available',
  '--docker-repo': '<repo>/<model>[:quant] Docker Hub repo',
  '--dry-allowed-length': 'DRY allowed length (default 2)',
  '--dry-base': 'DRY base value (default 1.75)',
  '--dry-multiplier': 'DRY multiplier (0.0 = disabled)',
  '--dry-penalty-last-n': 'DRY penalty last N tokens (-1 = ctx)',
  '--dry-sequence-breaker': 'DRY sequence breaker string',
  '--dynatemp-exp': 'dynamic temperature exponent (default 1.0)',
  '--dynatemp-range': 'dynamic temperature range (default 0.0)',
  '--embd-gemma-default': 'use default EmbeddingGemma model (note: can download weights from the internet)',
  '--embd-normalize': 'normalization: -1=none, 0=max, 1=taxicab, 2=euclidean',
  '--embedding': 'embedding mode only',
  '--escape': 'process escape sequences in prompt',
  '--fim-qwen-1.5b-default': 'use default Qwen 2.5 Coder 1.5B (note: can download weights from the internet)',
  '--fim-qwen-14b-spec': 'use Qwen 2.5 Coder 14B + 0.5B draft for speculative decoding (note: can download weights from the…',
  '--fim-qwen-30b-default': 'use default Qwen 3 Coder 30B A3B Instruct (note: can download weights from the internet)',
  '--fim-qwen-3b-default': 'use default Qwen 2.5 Coder 3B (note: can download weights from the internet)',
  '--fim-qwen-7b-default': 'use default Qwen 2.5 Coder 7B (note: can download weights from the internet)',
  '--fim-qwen-7b-spec': 'use Qwen 2.5 Coder 7B + 0.5B draft for speculative decoding (note: can download weights from the…',
  '--fit': 'on/off — auto-adjust args to fit device memory',
  '--fit-ctx': 'minimum ctx size set by --fit (default 4096)',
  '--fit-target': 'target margin per device for --fit (MiB)',
  '--flash-attn': 'on, off, or auto',
  '--frequency-penalty': '0.0–2.0',
  '--gpt-oss-120b-default': 'use gpt-oss-120b (note: can download weights from the internet)',
  '--gpt-oss-20b-default': 'use gpt-oss-20b (note: can download weights from the internet)',
  '--grammar': 'BNF-like grammar string',
  '--grammar-file': 'path to grammar file',
  '--hf-file': 'Hugging Face model file name',
  '--hf-repo': '<user>/<model>[:quant] Hugging Face repo',
  '--hf-token': 'Hugging Face access token',
  '--host': 'IP address (default 0.0.0.0)',
  '--ignore-eos': 'ignore EOS token and continue generating',
  '--image-max-tokens': 'max tokens per image',
  '--image-min-tokens': 'min tokens per image',
  '--jinja': 'use jinja template engine for chat',
  '--json-schema': 'JSON schema string',
  '--json-schema-file': 'path to JSON schema file',
  '--keep': 'tokens to keep from prompt (-1 = all)',
  '--kv-offload': 'enable KV cache offloading (default: on)',
  '--kv-unified': 'shared KV buffer across sequences',
  '--kv-unified-per-slot': 'per-slot context limit for shared KV pool',
  '--lazy-mode': 'on-demand reading of certain tensors, for example per-layer embeddings (default: auto) - on: read…',
  '--list-devices': 'print list of available devices and exit',
  '--load-mode': 'auto, none, mmap, mlock, mmap+mlock, dio — replaces --mmap/--mlock/--direct-io',
  '--log-colors': 'on/off/auto — colored logging',
  '--log-disable': 'disable all logging',
  '--log-file': 'path to log file',
  '--log-prefix': 'enable prefix in log messages',
  '--log-prompts-dir': 'directory to log prompts (debug)',
  '--log-timestamps': 'enable timestamps in log messages',
  '--log-verbose': 'set verbosity level to infinity',
  '--logit-bias': 'TOKEN_ID(+/-)BIAS',
  '--lookup-cache-dynamic': 'path to dynamic lookup cache',
  '--lookup-cache-static': 'path to static lookup cache',
  '--lora': 'path to LoRA adapter(s)',
  '--lora-init-without-apply': 'load LoRA adapters without applying',
  '--lora-scaled': 'FNAME:SCALE LoRA with scaling',
  '--main-gpu': 'GPU index for main GPU (default 0)',
  '--mcp-servers-config': 'path to JSON with MCP server definitions (experimental)',
  '--mcp-servers-json': 'inline JSON with MCP server definitions (experimental)',
  '--media-path': 'directory for local media files',
  '--metrics': 'enable Prometheus metrics endpoint',
  '--min-p': '0.0–1.0 (default 0.05)',
  '--mirostat': '0=off, 1=MIROSTAT, 2=MIROSTAT 2.0',
  '--mirostat-ent': 'target entropy (default 5.0)',
  '--mirostat-lr': 'learning rate (default 0.1)',
  '--mlock': 'DEPRECATED: use --load-mode mlock — force model to stay in RAM',
  '--mmap': 'DEPRECATED: use --load-mode mmap',
  '--mmproj': 'path to multimodal projector file',
  '--mmproj-auto': 'auto-download multimodal projector from HF',
  '--mmproj-device': 'device for multimodal projector (none = CPU, default: auto)',
  '--mmproj-offload': 'offload multimodal projector to GPU',
  '--mmproj-url': 'URL to multimodal projector file',
  '--model': 'path to model file',
  '--model-url': 'model download URL',
  '--models-autoload': 'auto-load models for router server',
  '--models-dir': 'directory for router server models',
  '--models-max': 'max models to load simultaneously (default 4)',
  '--models-preset': 'path to router model presets INI file',
  '--moe-expert-cache': 'GPU cache slots per host-resident MoE expert layer, 0 = disabled (default: 0)',
  '--moe-expert-cache-inserts': 'max expert uploads per layer per decode step for the MoE expert cache (default: 2)',
  '--mtmd-batch-max-tokens': 'max image tokens per batch (default 1024)',
  '--n-cpu-ffn': 'keep dense FFN weights of first N layers in CPU',
  '--n-cpu-moe': 'keep MoE of first N layers in CPU',
  '--n-cpu-moe-draft': 'keep first N MoE layers in CPU for draft',
  '--n-gpu-layers': 'number or "all"',
  '--n-gpu-layers-draft': 'GPU layers for draft model',
  '--no-agent': 'enable CORS proxy and all built-in tools',
  '--no-cache-idle-slots': 'save idle slots to prompt cache',
  '--no-cache-prompt': 'enable prompt caching',
  '--no-cont-batching': 'enable continuous batching',
  '--no-context-shift': 'enable context shift for infinite text gen',
  '--no-cors-credentials': 'allow credentials for CORS (default: on)',
  '--no-direct-io': 'DEPRECATED: use --load-mode dio — use DirectIO if available',
  '--no-escape': 'process escape sequences in prompt',
  '--no-host': 'disable host buffer',
  '--no-jinja': 'use jinja template engine for chat',
  '--no-kv-offload': 'enable KV cache offloading (default: on)',
  '--no-kv-unified': 'shared KV buffer across sequences',
  '--no-log-prefix': 'enable prefix in log messages',
  '--no-log-timestamps': 'enable timestamps in log messages',
  '--no-mmap': 'DEPRECATED: use --load-mode mmap',
  '--no-mmproj': 'auto-download multimodal projector from HF',
  '--no-mmproj-auto': 'auto-download multimodal projector from HF',
  '--no-mmproj-offload': 'offload multimodal projector to GPU',
  '--no-models-autoload': 'auto-load models for router server',
  '--no-op-offload': 'offload host tensor operations to device',
  '--no-perf': 'enable performance timings',
  '--no-prefill-assistant': 'prefill assistant response',
  '--no-reasoning-preserve': 'preserve reasoning trace in full history',
  '--no-repack': 'enable weight repacking',
  '--no-skip-chat-parsing': 'force pure content parser for chat',
  '--no-slots': 'expose slots monitoring endpoint (default: on)',
  '--no-spec-draft-backend-sampling': 'offload draft sampling to backend',
  '--no-ui': 'enable/disable the web UI',
  '--no-ui-mcp-proxy': 'enable MCP CORS proxy (experimental)',
  '--no-warmup': 'perform warmup with an empty run (default: on)',
  '--numa': 'NUMA optimization: distribute, isolate, numactl',
  '--offline': 'offline mode — no network access',
  '--op-offload': 'offload host tensor operations to device',
  '--override-kv': 'KEY=TYPE:VALUE — override model metadata',
  '--override-tensor': '<tensor>=<type> override tensor buffer type',
  '--override-tensor-draft': 'override tensor buffer type for draft',
  '--parallel': 'number of server slots',
  '--path': 'path to serve static files',
  '--perf': 'enable performance timings',
  '--poll': '<0-100> polling level to wait for work (default 50)',
  '--poll-batch': '<0|1> use polling for batch work',
  '--poll-batch-draft': '<0|1> use polling for draft batch work',
  '--poll-draft': '<0|1> use polling for draft work',
  '--pooling': 'none, mean, cls, last, rank',
  '--port': 'port number (default 8080)',
  '--predict': 'tokens to predict (-1 = infinity)',
  '--prefill-assistant': 'prefill assistant response',
  '--presence-penalty': '0.0–2.0',
  '--prio': 'thread priority: 0=normal, 1=medium, 2=high, 3=realtime',
  '--prio-batch': 'batch thread priority: -1=low, 0=normal, 1=medium, 2=high, 3=realtime',
  '--prio-batch-draft': 'draft batch thread priority: 0-normal, 1-medium, 2-high, 3-realtime',
  '--prio-draft': 'draft thread priority: 0-normal, 1-medium, 2-high, 3-realtime',
  '--props': 'enable changing properties via POST /props',
  '--reasoning': 'on, off, or auto',
  '--reasoning-budget': 'token budget for thinking (-1 = unlimited)',
  '--reasoning-budget-message': 'message when budget exhausted',
  '--reasoning-effort': 'default, minimal, low, medium, high, xhigh, max',
  '--reasoning-format': 'none, deepseek, deepseek-legacy',
  '--reasoning-preserve': 'preserve reasoning trace in full history',
  '--repack': 'enable weight repacking',
  '--repeat-last-n': 'last N tokens for penalty (0=off, -1=ctx)',
  '--repeat-penalty': '1.0–1.5 (default 1.0)',
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
  '--skip-chat-parsing': 'force pure content parser for chat',
  '--sleep-idle-seconds': 'auto-sleep after N seconds idle (-1=off)',
  '--slot-prompt-similarity': 'prompt match threshold (default 0.1)',
  '--slot-save-path': 'path to save slot KV cache',
  '--slots': 'expose slots monitoring endpoint (default: on)',
  '--spec-default': 'enable default speculative decoding config',
  '--spec-draft-backend-sampling': 'offload draft sampling to backend',
  '--spec-draft-cpu-mask': 'draft CPU affinity mask (hex)',
  '--spec-draft-cpu-mask-batch': 'draft batch CPU affinity mask (hex)',
  '--spec-draft-cpu-moe': 'keep MoE weights for draft in CPU',
  '--spec-draft-cpu-range': 'draft CPU range (lo-hi)',
  '--spec-draft-cpu-strict': '<0|1> strict CPU placement for draft',
  '--spec-draft-cpu-strict-batch': '<0|1> strict CPU placement for draft batch',
  '--spec-draft-device': 'devices for draft model offloading',
  '--spec-draft-hf': '<user>/<model>[:quant] HF repo for draft',
  '--spec-draft-model': 'path to draft model',
  '--spec-draft-n-cpu-moe': 'keep first N MoE layers in CPU for draft',
  '--spec-draft-n-max': 'draft tokens max (default 3)',
  '--spec-draft-n-min': 'draft tokens min (default 0)',
  '--spec-draft-ngl': 'GPU layers for draft model',
  '--spec-draft-p-min': 'minimum speculative decoding probability (default 0.00)',
  '--spec-draft-p-split': 'speculative split probability (default 0.1)',
  '--spec-draft-poll': '<0|1> use polling for draft work',
  '--spec-draft-poll-batch': '<0|1> use polling for draft batch work',
  '--spec-draft-prio': 'draft thread priority: 0-normal, 1-medium, 2-high, 3-realtime',
  '--spec-draft-prio-batch': 'draft batch thread priority: 0-normal, 1-medium, 2-high, 3-realtime',
  '--spec-draft-threads': 'CPU threads for draft model',
  '--spec-draft-threads-batch': 'batch threads for draft model',
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
  '--spec-synth-len': 'target mean synthetic acceptance length (benchmarking only)',
  '--spec-synth-rates': 'per-position synthetic acceptance probabilities (benchmarking only)',
  '--spec-type': 'none, draft-simple, draft-mtp, ngram-cache, ...',
  '--special': 'enable special tokens output',
  '--split-mode': 'none, layer, row, tensor',
  '--spm-infill': 'Suffix/Prefix/Middle infill pattern',
  '--sse-ping-interval': 'SSE ping interval in seconds (default 30)',
  '--ssl-cert-file': 'path to SSL cert file (PEM)',
  '--ssl-key-file': 'path to SSL key file (PEM)',
  '--swa-checkpoints': 'max context checkpoints per slot (default 32)',
  '--swa-full': 'use full-size SWA cache',
  '--tags': 'comma-separated model tags',
  '--temp': '0.0–2.0 (default 0.7)',
  '--tensor-split': 'GPU split proportions, e.g. 3,1',
  '--threads': 'CPU thread count',
  '--threads-batch': 'batch/prompt processing threads',
  '--threads-batch-draft': 'batch threads for draft model',
  '--threads-draft': 'CPU threads for draft model',
  '--threads-http': 'HTTP request threads',
  '--timeout': 'server read/write timeout in seconds (default 3600)',
  '--tools': 'comma-separated built-in tools (or "all")',
  '--tools-runtime': 'run tools in separate runtime (docker:<image>, ssh:<target>, ...)',
  '--top-k': 'e.g. 40 (0 = off)',
  '--top-n-sigma': 'top-n-sigma sampling (-1 = disabled)',
  '--top-p': '0.0–1.0 (default 0.95)',
  '--typical-p': 'locally typical sampling (1.0 = disabled)',
  '--ubatch-size': 'physical max batch size (default 512)',
  '--ui': 'enable/disable the web UI',
  '--ui-config': 'JSON string for default UI settings',
  '--ui-config-file': 'path to JSON file for default UI settings',
  '--ui-mcp-proxy': 'enable MCP CORS proxy (experimental)',
  '--verbose': 'set verbosity level to infinity',
  '--verbosity': 'set verbosity threshold (0-5, default 3)',
  '--video-ffmpeg-dir': 'path to directory containing ffmpeg and ffprobe',
  '--video-fps': 'target video frame rate (default 4.0)',
  '--video-timestamp-interval': 'interval in ms between text timestamps (default 5000)',
  '--vision-gemma-12b-default': 'use Gemma 3 12B QAT (note: can download weights from the internet)',
  '--vision-gemma-4b-default': 'use Gemma 3 4B QAT (note: can download weights from the internet)',
  '--warmup': 'perform warmup with an empty run (default: on)',
  '--xtc-probability': 'XTC probability (0.0 = disabled)',
  '--xtc-threshold': 'XTC threshold (default 0.1)',
  '--yarn-attn-factor': 'YaRN attention magnitude scale',
  '--yarn-beta-fast': 'YaRN low correction dim',
  '--yarn-beta-slow': 'YaRN high correction dim',
  '--yarn-ext-factor': 'YaRN extrapolation mix factor',
  '--yarn-orig-ctx': 'YaRN original context size'
};
// END generated flag catalog

function makeFlagRow(name, value) {
  var r = document.createElement('div'); r.className = 'flag-row';
  // Every flag — cataloged or not — lives in the search field, so a flag the
  // user typed that is not in the catalog round-trips through the same field
  // they typed it into. The separate key input only appears when the user
  // explicitly picks "Custom…" from the dropdown. A non-catalog flag gets a
  // subtle amber underline so it reads as "stored as typed", not selected.
  var isCustom = name && commonFlags.indexOf(name) === -1;
  r.innerHTML =
    '<div class="flag-search-wrapper">' +
      '<input type="text" class="flag-search' + (isCustom ? ' flag-unknown' : '') + '" placeholder="Search or type flag\u2026" autocomplete="off" value="' + escAttr(name || '') + '"' + (isCustom ? ' title="Not in the llama.cpp flag catalog — stored as typed"' : '') + '>' +
      '<div class="flag-dropdown" style="display:none"></div>' +
    '</div>' +
    '<input type="text" class="flag-custom" placeholder="Flag name" autocomplete="off" style="display:none">' +
    '<input type="text" class="flag-value" placeholder="Value" autocomplete="off" value="' + escAttr(value || '') + '">' +
    '<button class="small danger" onclick="this.parentElement.remove()" aria-label="Remove flag">\u2715</button>';
  if (standaloneFlags[name]) r.querySelector('.flag-value').style.display = 'none';
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
    var r = await apiFetch('/api/v1/presets'), presets = await r.json();
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
  apiFetch('/api/v1/presets').then(function(r) { return r.json(); }).then(function(presets) {
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
    await apiFetch('/api/v1/presets', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name: name, flags: flags }) });
    loadPresets();
  } catch (e) { alert('Error: ' + e); }
}

async function deletePreset() {
  var sel = document.getElementById('presetSelect'), name = sel.value;
  if (!name) { alert('Select a preset first'); return; }
  if (!confirm('Delete preset "' + name + '"?')) return;
  try {
    await apiFetch('/api/v1/presets?name=' + encodeURIComponent(name), { method: 'DELETE' });
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
    var r = await apiFetch('/api/v1/models/pull/stream?model=' + encodeURIComponent(ref), { signal: _pullAbort.signal });
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
    var r = await apiFetch('/api/v1/models/search?q=' + encodeURIComponent(q));
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
    var r = await apiFetch('/api/v1/models/repo-files?repo=' + encodeURIComponent(repo));
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
    var r = await apiFetch('/api/v1/config/default-flags'), flags = await r.json();
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
  // T4: create an AbortController so the user can stop the stream.
  chatAbortController = new AbortController();
  setChatBusy(true);
  try {
    var r = await apiFetch('/api/v1/chat?port=' + chatPort, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ model: 'default', messages: chatHistory.slice(-20), max_tokens: 4096, stream: true }), signal: chatAbortController.signal });
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
  } catch (e) {
    // T4: an abort (user pressed Stop) is not an error. Preserve any partial
    // content so the next turn's context stays coherent (deliberate choice —
    // see the Phase 6 plan, T4).
    if (e.name === 'AbortError') {
      var partial = reasoning ? '<think>' + reasoning + '</think>' + content : content;
      if (partial) { chatHistory.push({ role: 'assistant', content: partial }); saveChatHistory(); }
      addSystemMsg('Stopped');
    }
    else { if (msgEl) { msgEl.innerHTML = 'Error: ' + escHtml(e.message); msgEl.className = 'msg system'; } else { var em = addMsg('assistant', ''); em.innerHTML = 'Error: ' + escHtml(e.message); em.className = 'msg system'; } }
  } finally {
    chatAbortController = null;
    setChatBusy(false);
  }
}

// T4: toggle the Send button into a Stop button while a stream is in flight.
function setChatBusy(busy) {
  var btn = document.getElementById('chatSendBtn');
  if (!btn) return;
  if (busy) { btn.textContent = 'Stop'; btn.classList.add('danger'); btn.setAttribute('onclick', 'stopChat()'); btn.disabled = false; }
  else { btn.textContent = 'Send'; btn.classList.remove('danger'); btn.setAttribute('onclick', 'sendChat()'); }
}
function stopChat() {
  if (chatAbortController) chatAbortController.abort();
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
    var r = await apiFetch('/api/v1/chats/' + (chatSessionId || ''), { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload) });
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
    var r = await apiFetch('/api/v1/chats/' + id);
    if (!r.ok) return;
    var session = await r.json();
    session.title = newTitle;
    await apiFetch('/api/v1/chats/' + id, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(session) });
    showChatHistory();
  } catch (e) {}
}

async function showChatHistory() {
  document.getElementById('chatHistoryModal').style.display = 'block';
  var list = document.getElementById('chatHistoryList');
  try {
    var r = await apiFetch('/api/v1/chats'), chats = await r.json();
    if (!chats || !chats.length) { list.innerHTML = '<div class="empty-state"><div class="icon">💬</div><div class="title">No saved chats</div></div>'; return; }
    list.innerHTML = chats.map(function(c) {
      var title = c.title || c.preview || '(empty)';
      var shortModel = c.model ? c.model.split('/').pop().split(':')[0].replace(/-GGUF$/i, '').slice(0, 25) : '';
      return '<div class="chat-history-item" data-id="' + c.id + '"><div class="chat-history-main" onclick="loadChat(\'' + c.id + '\')"><div class="chat-history-title">' + escHtml(title.length > 50 ? title.slice(0, 50) + '…' : title) + '</div><div class="chat-history-meta">' + c.msg_count + ' msgs · ' + escHtml(shortModel) + ' · ' + new Date(c.updated_at).toLocaleDateString() + '</div></div><div class="chat-history-actions"><button class="small ghost" onclick="event.stopPropagation();renameChat(\'' + c.id + '\', \'' + escAttr(title.replace(/'/g, '')) + '\')" title="Rename">✏️</button><button class="small danger" onclick="event.stopPropagation();deleteChat(\'' + c.id + '\')" title="Delete"><span class="icon">🗑</span></button></div></div>';
    }).join('');
  } catch (e) { list.innerHTML = '<div class="empty-state"><div class="title">Error loading chats</div></div>'; }
}

function closeChatHistory() { document.getElementById('chatHistoryModal').style.display = 'none'; }
document.getElementById('chatHistoryModal').addEventListener('click', closeChatHistory);

async function loadChat(id) {
  closeChatHistory();
  try {
    var r = await apiFetch('/api/v1/chats/' + id), session = await r.json();
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
    await apiFetch('/api/v1/chats/' + id, { method: 'DELETE' });
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
    var r = await apiFetch('/api/v1/instances/logs?port=' + port), d = await r.json();
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

// T3: render the llama-server freshness badge next to the version tag.
//  - outdated && comparable → amber "N BEHIND" (links to release notes)
//  - comparable && current  → green "UP TO DATE"
//  - not comparable / unknown / lookup error → no badge (no false alarms on
//    custom builds, and an unreachable GitHub is not shown as "outdated").
function renderLlamaFreshness(vd) {
  var el = document.getElementById('s-llama-freshness');
  if (!el) return;
  el.innerHTML = '';
  if (vd.llama_server_comparable !== true) return;
  if (vd.llama_server_outdated === true) {
    var n = vd.llama_server_builds_behind || 0;
    var text = n + ' BEHIND';
    var href = vd.llama_server_release_url || '';
    if (href) {
      el.innerHTML = '<a class="badge badge-amber" href="' + escAttr(href) + '" target="_blank" rel="noopener" title="llama.cpp release notes">' + escHtml(text) + '</a>';
    } else {
      el.innerHTML = '<span class="badge badge-amber">' + escHtml(text) + '</span>';
    }
  } else {
    el.innerHTML = '<span class="badge badge-green">UP TO DATE</span>';
  }
}

async function loadSettings() {
  try {
    var vr = await apiFetch('/api/v1/version'), vd = await vr.json();
    if (vd.version) { document.getElementById('s-version').textContent = vd.version; var fv = document.getElementById('faceVersion'); if (fv) fv.textContent = vd.version; }
    if (vd.llama_server) document.getElementById('s-llama-version').textContent = vd.llama_server;
    if (vd.backend) document.getElementById('s-backend').textContent = vd.backend;
    renderLlamaFreshness(vd);
  } catch (e) {}
  // T5: OpenAI-compatible endpoint (always derivable from the current origin).
  var ep = document.getElementById('openaiEndpoint');
  if (ep) ep.textContent = location.origin + '/v1';
  try {
    var r = await apiFetch('/api/v1/config'), cfg = await r.json();
    // API token: show it and keep this browser's stored copy in sync
    // (the config GET already required the token, so storing it is safe).
    var tv = document.getElementById('apiTokenValue');
    if (tv) tv.textContent = cfg.api_token || '—';
    if (cfg.api_token) setToken(cfg.api_token);
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
    var r = await apiFetch('/api/v1/config', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ idle_ttl: val }) });
    if (r.ok) { st.textContent = 'Saved'; setTimeout(function() { st.textContent = ''; }, 2000); }
    else { st.textContent = 'Error saving'; }
  } catch (e) { st.textContent = 'Error: ' + e.message; }
}

async function saveSettingsFlags() {
  var st = document.getElementById('settingsFlagsStatus');
  try {
    var r = await apiFetch('/api/v1/config', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ default_flags: collectFlags(document.getElementById('settingsFlagsContainer')) }) });
    if (r.ok) { st.textContent = 'Saved'; setTimeout(function() { st.textContent = ''; }, 2000); loadSettings(); document.getElementById('settings-ql').classList.remove('editing'); }
    else { st.textContent = 'Error saving'; }
  } catch (e) { st.textContent = 'Error: ' + e.message; }
}

async function saveProxyFlags() {
  var st = document.getElementById('proxyFlagsStatus');
  try {
    var r = await apiFetch('/api/v1/config', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ proxy_defaults: collectFlags(document.getElementById('proxyFlagsContainer')) }) });
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
    var r = await apiFetch('/api/v1/config', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ profiles: profiles }) });
    if (r.ok) { st.textContent = 'Saved'; setTimeout(function() { st.textContent = ''; }, 2000); loadSettings(); document.getElementById('settings-profiles').classList.remove('editing'); document.getElementById('settings-image-profiles').classList.remove('editing'); }
    else { st.textContent = 'Error saving'; }
  } catch (e) { st.textContent = 'Error: ' + e.message; }
}

async function restartGollama() {
  if (!confirm('Restart gollama? The web UI will be unavailable for a few seconds.')) return;
  try {
    await apiFetch('/api/v1/restart', { method: 'POST' });
    setTimeout(function() { location.reload(); }, 2000);
  } catch (e) { alert('Restart failed: ' + e.message); }
}

// ── Image Generation ──────────────────────────────────
var _cachedImageProfiles = {};

async function loadImageProfiles() {
  var sel = document.getElementById('imageProfileSelect');
  if (!sel) return;
  try {
    var r = await apiFetch('/api/v1/config'), cfg = await r.json();
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
    var r = await apiFetch('/api/v1/image-models'), models = await r.json();
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
      var r = await apiFetch('/api/v1/image-models/search?q=' + encodeURIComponent(query));
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
    var r = await apiFetch('/api/v1/image-models/install', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name: name, model_id: modelId }) });
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
    var r = await apiFetch('/v1/images/generations', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
    var retries = 0;
    while (r.status == 503 && retries < 20) {
      var retryAfter = parseInt(r.headers.get('Retry-After')) || 5;
      loadingCard.innerHTML = '<div class="spinner"></div><div>Loading model… (attempt ' + (retries + 1) + ')</div>';
      await new Promise(function(res) { setTimeout(res, retryAfter * 1000); });
      r = await apiFetch('/v1/images/generations', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
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