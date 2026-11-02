// Copyright (C) 2026 Ajay Raikvar. SPDX-License-Identifier: AGPL-3.0-or-later. See ../LICENSE and ../NOTICE.
(() => {
  'use strict';
  const $=id=>document.getElementById(id), fmt=(n,d=1)=>Number.isFinite(Number(n))?Number(n).toFixed(d):'—';
  let rows=[];
  function rate(n){if(!Number.isFinite(Number(n)))return '—';return n>=1000?`${(n/1000).toFixed(2)} Gbps`:`${n.toFixed(1)} Mbps`;}
  async function load(){
    $('status').textContent='Refreshing host data…';
    try{
      const [statsR,clientsR,logsR]=await Promise.all([fetch('/api/admin/stats',{cache:'no-store'}),fetch('/api/admin/clients',{cache:'no-store'}),fetch('/api/admin/logs',{cache:'no-store'})]);
      if(!statsR.ok||!clientsR.ok||!logsR.ok)throw new Error(`HTTP ${statsR.status}/${clientsR.status}/${logsR.status}`);
      const stats=await statsR.json(),clients=await clientsR.json(),logs=await logsR.json();rows=Array.isArray(logs.items)?logs.items.slice().reverse():[];
      $('totalTests').textContent=stats.totalTests??0;$('activeClients').textContent=`${stats.activeClients??0} / ${stats.maxClients??'—'}`;$('averageDownload').textContent=rate(stats.averageDownloadMbps);$('averageUpload').textContent=rate(stats.averageUploadMbps);$('averageLatency').textContent=`${fmt(stats.averageLatencyMs)} ms`;$('clientLimit').textContent=`· ${clients.count}/${clients.maxClients} online`;
      renderClients(clients.items||[]);renderRows();$('logCount').textContent=`· ${logs.count??rows.length} records`;$('status').textContent=`Updated ${new Date().toLocaleTimeString()}. Persistent log is stored in the host data directory.`;
    }catch(e){$('status').textContent=`Could not load host data: ${e.message}`;}
  }
  function renderClients(items){const root=$('clientGrid');root.replaceChildren();if(!items.length){const p=document.createElement('p');p.className='empty';p.textContent='No active browser clients. Active sessions expire after 35 seconds without a heartbeat.';root.append(p);return;}for(const c of items){const card=document.createElement('article');card.className='client-card';const name=document.createElement('strong');name.textContent=c.clientName||'Browser client';const ip=document.createElement('span');ip.textContent=c.clientIp||'IP unavailable';const details=document.createElement('small');details.textContent=`${c.browser||'Browser'} · ${c.platform||'Platform unknown'} · ${c.connectionType||'Network unknown'}`;const seen=document.createElement('small');seen.textContent=`Last seen ${new Date(c.lastSeen).toLocaleTimeString()}`;card.append(name,ip,details,seen);root.append(card);}}
  function renderRows(){const root=$('logRows');root.replaceChildren();const q=$('searchBox').value.trim().toLowerCase();const visible=rows.filter(x=>[x.clientName,x.clientIp,x.browser,x.platform,x.testId,x.id].join(' ').toLowerCase().includes(q));$('logCount').textContent=`· ${visible.length} shown / ${rows.length} loaded`;
    if(!visible.length){const tr=document.createElement('tr'),td=document.createElement('td');td.colSpan=10;td.textContent='No matching test records.';tr.append(td);root.append(tr);return;}
    for(const x of visible){const tr=document.createElement('tr');const values=[x.timestamp?new Date(x.timestamp).toLocaleString():'—',x.clientName||'Browser client',x.clientIp||'—',`${rate(x.downloadMbps)} / ${rate(x.downloadPeakMbps)}`,`${rate(x.uploadMbps)} / ${rate(x.uploadPeakMbps)}`,`${fmt(x.latencyMs)} ms`,`${fmt(x.jitterMs)} ms`,x.mode||'sequential',String(x.streams||1),x.id||'—'];for(const v of values){const td=document.createElement('td');td.textContent=v;tr.append(td);}root.append(tr);}
    renderComparison(visible);
  }
  function renderComparison(items){const root=$('comparison');root.replaceChildren();const sorted=items.slice().sort((a,b)=>(b.downloadMbps||0)-(a.downloadMbps||0)).slice(0,8);if(!sorted.length){const p=document.createElement('p');p.className='empty';p.textContent='Run tests to compare client throughput.';root.append(p);return;}const max=Math.max(1,...sorted.map(x=>x.downloadMbps||0));for(const x of sorted){const row=document.createElement('div');row.className='compare-row';const name=document.createElement('span');name.textContent=`${x.clientName||'Client'} · ${x.clientIp||'—'}`;const bar=document.createElement('div');bar.className='bar-track';const fill=document.createElement('i');fill.style.width=`${Math.max(1,(x.downloadMbps||0)/max*100)}%`;bar.append(fill);const value=document.createElement('b');value.textContent=rate(x.downloadMbps||0);row.append(name,bar,value);root.append(row);}}
  $('refreshButton').addEventListener('click',load);$('searchBox').addEventListener('input',renderRows);load();setInterval(load,10000);
})();
