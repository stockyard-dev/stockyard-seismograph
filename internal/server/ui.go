package server

import "net/http"

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(dashboardHTML))
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0">
<title>Seismograph</title>
<style>
:root{--bg:#1a1410;--bg2:#241e18;--bg3:#2e261e;--rust:#e8753a;--leather:#a0845c;--cream:#f0e6d3;--cd:#bfb5a3;--cm:#7a7060;--gold:#d4a843;--green:#4a9e5c;--red:#c94444;--orange:#d4843a;--blue:#4a7ec9;--mono:'JetBrains Mono',monospace;--serif:'Libre Baskerville',serif}
*{margin:0;padding:0;box-sizing:border-box}body{background:var(--bg);color:var(--cream);font-family:var(--serif);line-height:1.6}
.header{padding:1rem 1.5rem;border-bottom:1px solid var(--bg3);display:flex;justify-content:space-between;align-items:center}
.header h1{font-family:var(--mono);font-size:.9rem;letter-spacing:2px}
.stats-bar{display:flex;gap:1.5rem;font-family:var(--mono);font-size:.7rem}
.stat-item{display:flex;align-items:center;gap:.3rem}
.stat-dot{width:6px;height:6px;border-radius:50%}
.filters{padding:.8rem 1.5rem;border-bottom:1px solid var(--bg3);display:flex;gap:.5rem;flex-wrap:wrap}
.filter-btn{font-family:var(--mono);font-size:.65rem;padding:.25rem .6rem;border:1px solid var(--bg3);background:var(--bg);color:var(--cm);cursor:pointer}
.filter-btn:hover{border-color:var(--leather);color:var(--cream)}
.filter-btn.active{border-color:var(--rust);color:var(--rust)}
.content{padding:1rem 1.5rem;max-width:1000px;margin:0 auto}
.error-row{border:1px solid var(--bg3);background:var(--bg2);margin-bottom:.5rem;cursor:pointer;transition:border-color .15s}
.error-row:hover{border-color:var(--leather)}
.error-top{padding:.8rem 1rem;display:flex;align-items:flex-start;gap:.8rem}
.level-badge{font-family:var(--mono);font-size:.55rem;padding:.15rem .4rem;text-transform:uppercase;letter-spacing:1px;flex-shrink:0;margin-top:.15rem}
.level-fatal{background:#c9444433;color:#ff6b6b;border:1px solid #c9444455}
.level-error{background:#c9444422;color:var(--red);border:1px solid #c9444444}
.level-warning{background:#d4843a22;color:var(--orange);border:1px solid #d4843a44}
.level-info{background:#4a7ec922;color:var(--blue);border:1px solid #4a7ec944}
.level-debug{background:var(--bg3);color:var(--cm);border:1px solid var(--bg3)}
.error-info{flex:1;min-width:0}
.error-title{font-family:var(--mono);font-size:.8rem;margin-bottom:.15rem;word-break:break-all}
.error-meta{font-family:var(--mono);font-size:.6rem;color:var(--cm);display:flex;gap:1rem;flex-wrap:wrap}
.error-right{display:flex;flex-direction:column;align-items:flex-end;gap:.3rem;flex-shrink:0}
.count-badge{font-family:var(--mono);font-size:.7rem;background:var(--bg3);padding:.1rem .5rem;color:var(--cd)}
.status-badge{font-family:var(--mono);font-size:.55rem;padding:.1rem .4rem;text-transform:uppercase;letter-spacing:1px}
.status-open{color:var(--red);border:1px solid #c9444444}
.status-acknowledged{color:var(--orange);border:1px solid #d4843a44}
.status-resolved{color:var(--green);border:1px solid #4a9e5c44}
.status-ignored{color:var(--cm);border:1px solid var(--bg3)}
.error-detail{display:none;padding:0 1rem 1rem;border-top:1px solid var(--bg3);margin-top:0}
.error-detail.open{display:block}
.stack-trace{background:#0d0b09;padding:.8rem;font-family:var(--mono);font-size:.7rem;color:var(--cd);white-space:pre-wrap;overflow-x:auto;margin:.5rem 0;line-height:1.8}
.actions{display:flex;gap:.4rem;margin-top:.5rem}
.btn{font-family:var(--mono);font-size:.6rem;padding:.25rem .6rem;cursor:pointer;border:1px solid var(--bg3);background:var(--bg);color:var(--cd)}.btn:hover{border-color:var(--leather);color:var(--cream)}
.empty{text-align:center;padding:3rem;color:var(--cm);font-style:italic}
</style>
</head>
<body>
<div class="header">
  <h1>SEISMOGRAPH</h1>
  <div class="stats-bar" id="statsBar"></div>
</div>
<div class="filters" id="filters"></div>
<div class="content" id="main"></div>

<script>
const API='/api';
let errors=[],sources=[],filterLevel='',filterStatus='',filterSource='';

async function load(){
  const[e,src,st]=await Promise.all([
    fetch(API+'/errors').then(r=>r.json()),
    fetch(API+'/sources').then(r=>r.json()),
    fetch(API+'/stats').then(r=>r.json()),
  ]);
  errors=e.errors||[];sources=src.sources||[];
  renderStats(st);renderFilters();render();
}

function renderStats(st){
  document.getElementById('statsBar').innerHTML=
    '<div class="stat-item"><div class="stat-dot" style="background:var(--red)"></div>'+st.open+' open</div>'+
    '<div class="stat-item"><div class="stat-dot" style="background:var(--orange)"></div>'+st.acknowledged+' acked</div>'+
    '<div class="stat-item"><div class="stat-dot" style="background:var(--green)"></div>'+st.resolved+' resolved</div>'+
    '<div class="stat-item" style="color:var(--cm)">'+st.total+' total</div>';
}

function renderFilters(){
  let h='<span style="font-family:var(--mono);font-size:.6rem;color:var(--cm);padding:.25rem 0">FILTER:</span>';
  ['','fatal','error','warning','info'].forEach(l=>{
    h+='<button class="filter-btn'+(filterLevel===l?' active':'')+'" onclick="setLevel(\''+l+'\')">'+(!l?'All levels':l)+'</button>';
  });
  h+='<span style="width:1px;background:var(--bg3);margin:0 .3rem"></span>';
  ['','open','acknowledged','resolved','ignored'].forEach(s=>{
    h+='<button class="filter-btn'+(filterStatus===s?' active':'')+'" onclick="setStatusFilter(\''+s+'\')">'+(!s?'All status':s)+'</button>';
  });
  if(sources.length){
    h+='<span style="width:1px;background:var(--bg3);margin:0 .3rem"></span>';
    h+='<button class="filter-btn'+(filterSource===''?' active':'')+'" onclick="setSource(\'\')">All sources</button>';
    sources.forEach(s=>{h+='<button class="filter-btn'+(filterSource===s?' active':'')+'" onclick="setSource(\''+s+'\')">'+esc(s)+'</button>';});
  }
  document.getElementById('filters').innerHTML=h;
}

function setLevel(l){filterLevel=l;applyFilters();}
function setStatusFilter(s){filterStatus=s;applyFilters();}
function setSource(s){filterSource=s;applyFilters();}
async function applyFilters(){
  let url=API+'/errors?';
  if(filterLevel)url+='level='+filterLevel+'&';
  if(filterStatus)url+='status='+filterStatus+'&';
  if(filterSource)url+='source='+encodeURIComponent(filterSource)+'&';
  const r=await fetch(url).then(r=>r.json());
  errors=r.errors||[];renderFilters();render();
}

function render(){
  const m=document.getElementById('main');
  if(!errors.length){m.innerHTML='<div class="empty">No errors captured yet. POST to /api/errors to start tracking.</div>';return;}
  let h='';
  errors.forEach((e,i)=>{
    h+='<div class="error-row" onclick="toggle('+i+')"><div class="error-top"><span class="level-badge level-'+e.level+'">'+e.level+'</span><div class="error-info"><div class="error-title">'+esc(e.title)+'</div><div class="error-meta"><span>'+esc(e.source||'unknown')+'</span><span>first: '+fmtTime(e.first_seen)+'</span><span>last: '+fmtTime(e.last_seen)+'</span></div></div><div class="error-right"><span class="count-badge">'+e.count+'×</span><span class="status-badge status-'+e.status+'">'+e.status+'</span></div></div>';
    h+='<div class="error-detail" id="detail-'+i+'">';
    if(e.message&&e.message!==e.title)h+='<div style="font-size:.82rem;color:var(--cd);margin:.5rem 0">'+esc(e.message)+'</div>';
    if(e.stack)h+='<div class="stack-trace">'+esc(e.stack)+'</div>';
    if(e.metadata&&e.metadata!=='{}')h+='<div style="font-family:var(--mono);font-size:.65rem;color:var(--cm);margin:.3rem 0">Metadata: '+esc(e.metadata)+'</div>';
    h+='<div style="font-family:var(--mono);font-size:.6rem;color:var(--cm)">Fingerprint: '+e.fingerprint+'</div>';
    h+='<div class="actions">';
    if(e.status==='open')h+='<button class="btn" onclick="event.stopPropagation();setStatus(\''+e.id+'\',\'acknowledged\')">Acknowledge</button>';
    if(e.status!=='resolved')h+='<button class="btn" onclick="event.stopPropagation();setStatus(\''+e.id+'\',\'resolved\')">Resolve</button>';
    if(e.status!=='ignored')h+='<button class="btn" onclick="event.stopPropagation();setStatus(\''+e.id+'\',\'ignored\')">Ignore</button>';
    if(e.status!=='open')h+='<button class="btn" onclick="event.stopPropagation();setStatus(\''+e.id+'\',\'open\')">Reopen</button>';
    h+='<button class="btn" onclick="event.stopPropagation();del(\''+e.id+'\')" style="color:var(--red)">Delete</button>';
    h+='</div></div></div>';
  });
  m.innerHTML=h;
}

function toggle(i){document.getElementById('detail-'+i).classList.toggle('open');}

async function setStatus(id,status){
  await fetch(API+'/errors/'+id+'/status',{method:'PATCH',headers:{'Content-Type':'application/json'},body:JSON.stringify({status})});
  load();
}
async function del(id){if(confirm('Delete this error group and all occurrences?')){await fetch(API+'/errors/'+id,{method:'DELETE'});load();}}

function esc(s){if(!s)return'';const d=document.createElement('div');d.textContent=s;return d.innerHTML;}
function fmtTime(t){if(!t)return'';const d=new Date(t);return d.toLocaleDateString()+' '+d.toLocaleTimeString([],{hour:'2-digit',minute:'2-digit'});}

load();
</script>
<script>
(function(){
  fetch('/api/config').then(function(r){return r.json()}).then(function(cfg){
    if(!cfg||typeof cfg!=='object')return;
    if(cfg.dashboard_title){
      document.title=cfg.dashboard_title;
      var h1=document.querySelector('h1');
      if(h1){
        var inner=h1.innerHTML;
        var firstSpan=inner.match(/<span[^>]*>[^<]*<\/span>/);
        if(firstSpan){h1.innerHTML=firstSpan[0]+' '+cfg.dashboard_title}
        else{h1.textContent=cfg.dashboard_title}
      }
    }
  }).catch(function(){});
})();
</script>
</body>
</html>`
