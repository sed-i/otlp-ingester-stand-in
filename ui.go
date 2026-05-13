package main

const htmlUI = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>OTLP Telemetry Viewer</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background:#0d1117;color:#c9d1d9;padding:20px}
h1{font-size:1.5em;margin-bottom:4px;color:#58a6ff}
.subtitle{color:#8b949e;font-size:0.85em;margin-bottom:16px}
.tabs{display:flex;gap:0;margin-bottom:16px;border-bottom:2px solid #21262d}
.tab{padding:8px 20px;cursor:pointer;color:#8b949e;font-size:14px;font-weight:600;border:none;background:none;border-bottom:2px solid transparent;margin-bottom:-2px;transition:color .15s,border-color .15s}
.tab:hover{color:#c9d1d9}
.tab.active{color:#58a6ff;border-bottom-color:#58a6ff}
table{width:100%;border-collapse:collapse;font-size:13px}
th,td{padding:8px 12px;text-align:left;border-bottom:1px solid #21262d}
th{background:#161b22;color:#8b949e;font-weight:600;position:sticky;top:0}
tr:hover{background:#1c2128}
td{max-width:400px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.empty{text-align:center;padding:40px;color:#8b949e}
.status{color:#8b949e;font-size:0.85em;margin-bottom:12px}
.mono{font-family:'SF Mono','Fira Code',monospace;font-size:12px}
.severity{text-transform:uppercase;font-weight:600;font-size:11px;padding:2px 6px;border-radius:4px}
.sev-ERROR,.sev-FATAL{background:#da3633;color:#fff}
.sev-WARN{background:#d29922;color:#000}
.sev-INFO,.sev-INFORMATION{background:#238636;color:#fff}
.sev-DEBUG,.sev-TRACE{background:#484f58;color:#fff}
.badge{font-size:11px;padding:2px 6px;border-radius:4px;font-weight:600}
.badge-sum{background:#1f6feb;color:#fff}
.badge-gauge{background:#bf4b8a;color:#fff}
.badge-histogram{background:#8957e5;color:#fff}
.badge-exponential_histogram{background:#1a7f37;color:#fff}
.badge-server{background:#1f6feb;color:#fff}
.badge-client{background:#bf4b8a;color:#fff}
.badge-producer{background:#d29922;color:#000}
.badge-consumer{background:#8957e5;color:#fff}
.badge-internal{background:#8b949e;color:#fff}
</style>
</head>
<body>
<h1>OTLP Telemetry Viewer</h1>
<div class="subtitle">Ingest via POST /v1/logs /v1/metrics /v1/traces</div>
<div class="tabs">
  <button class="tab active" data-signal="logs">Logs</button>
  <button class="tab" data-signal="metrics">Metrics</button>
  <button class="tab" data-signal="traces">Spans</button>
</div>
<div class="status" id="status">Loading...</div>
<div style="overflow-x:auto">
<table>
<thead id="thead"></thead>
<tbody id="tbody"></tbody>
</table>
</div>
<script>
(function(){
  var signal="logs";
  var endpoint="/api/v1/logs";
  var labels={logs:"log record",metrics:"data point",traces:"span"};

  var schemas={
    logs:[
      {key:"timestamp",label:"Time",cls:"mono"},
      {key:"service",label:"Service"},
      {key:"scope",label:"Scope"},
      {key:"severity",label:"Severity",render:function(v){return '<span class="severity sev-'+v.replace(/\d.*/,"")+'">'+v+'</span>'}},
      {key:"body",label:"Body"},
      {key:"trace_id",label:"Trace ID",cls:"mono"},
      {key:"span_id",label:"Span ID",cls:"mono"}
    ],
    metrics:[
      {key:"timestamp",label:"Time",cls:"mono"},
      {key:"service",label:"Service"},
      {key:"scope",label:"Scope"},
      {key:"name",label:"Name"},
      {key:"type",label:"Type",render:function(v){return '<span class="badge badge-'+v+'">'+v+'</span>'}},
      {key:"value",label:"Value"}
    ],
    traces:[
      {key:"timestamp",label:"Start Time",cls:"mono"},
      {key:"service",label:"Service"},
      {key:"scope",label:"Scope"},
      {key:"name",label:"Name"},
      {key:"kind",label:"Kind",render:function(v){return '<span class="badge badge-'+v+'">'+v+'</span>'}},
      {key:"duration",label:"Duration",cls:"mono"},
      {key:"trace_id",label:"Trace ID",cls:"mono"},
      {key:"span_id",label:"Span ID",cls:"mono"}
    ]
  };

  function renderHead(){var s=schemas[signal];document.getElementById("thead").innerHTML='<tr>'+s.map(function(c){return'<th>'+c.label+'</th>'}).join("")+'</tr>'}

  function renderBody(rows){
    var s=schemas[signal],tbody=document.getElementById("tbody");
    if(rows.length===0){
      var ep=endpoint.replace("/api","");tbody.innerHTML='<tr><td colspan="'+s.length+'" class="empty">Waiting for '+signal+' &mdash; POST to '+ep+'</td></tr>';
      return
    }
    tbody.innerHTML=rows.slice().reverse().map(function(r){return'<tr>'+s.map(function(c){
      var raw=c.render?c.render(r[c.key]):r[c.key];
      return'<td'+(c.cls?' class="'+c.cls+'"':'')+'>'+raw+'</td>'
    }).join("")+'</tr>'}).join("")
  }

  async function poll(){var status=document.getElementById("status");try{var res=await fetch(endpoint);var rows=await res.json();status.textContent=rows.length+" "+labels[signal]+(rows.length!==1?"s":"")+"  ·  auto-refreshing every 2s";renderBody(rows)}catch(e){status.textContent="Error: "+e.message;document.getElementById("tbody").innerHTML=""};setTimeout(poll,2000)}

  document.querySelectorAll(".tab").forEach(function(t){t.addEventListener("click",function(){
    document.querySelectorAll(".tab").forEach(function(x){x.classList.remove("active")});this.classList.add("active");
    signal=this.dataset.signal;endpoint="/api/v1/"+signal;document.getElementById("status").textContent="Loading...";
    renderHead();poll()
  })});

  renderHead();poll()
})();
</script>
</body>
</html>`
