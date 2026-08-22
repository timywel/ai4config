// cfg4ai 桌面 Web 前端逻辑
const $ = s => document.querySelector(s);
let ENTITIES = [];

// ---- 主题 ----
function theme(el){
  document.querySelectorAll(".seg button").forEach(b=>b.classList.remove("on"));
  el.classList.add("on");
  document.documentElement.dataset.theme = el.dataset.t;
}
// ---- 页面切换 ----
const TITLES={home:["概览","你的配置健康一览"],entities:["配置","已纳管配置"],sync:["同步","快照与远端"],secrets:["密钥","secretref 托管"]};
function go(el){
  document.querySelectorAll(".nav-item").forEach(n=>n.classList.remove("on"));
  el.classList.add("on");
  goPage(el.dataset.page);
}
function goPage(pg){
  document.querySelectorAll(".page").forEach(p=>p.classList.add("hidden"));
  const t=$("#pg-"+pg); t.classList.remove("hidden");
  t.style.animation="none"; t.offsetHeight; t.style.animation="";
  const tt=TITLES[pg]||["",""];
  $("#ptitle").childNodes[0].textContent=tt[0]; $("#psub").textContent=tt[1];
  document.querySelectorAll(".nav-item").forEach(n=>n.classList.toggle("on", n.dataset.page===pg));
  if(pg==="home") loadOverview();
  if(pg==="entities") loadEntities();
  if(pg==="sync") loadSnapshots();
}
function focusCmd(){ toast("命令面板：输入过滤命令与配置（即将上线）"); }
// ---- 轻提示 ----
function toast(msg){
  let t=document.getElementById("toast");
  if(!t){ t=document.createElement("div"); t.id="toast";
    t.style.cssText="position:fixed;bottom:26px;left:50%;transform:translateX(-50%);background:var(--accent);color:var(--accent-text);padding:10px 20px;border-radius:10px;font-size:13px;box-shadow:var(--shadow-2);z-index:99;transition:opacity .3s"; 
    document.body.appendChild(t); }
  t.textContent=msg; t.style.opacity="1";
  clearTimeout(t._h); t._h=setTimeout(()=>t.style.opacity="0",2600);
}
// ---- API ----
async function api(path, opts){ const r=await fetch(path, opts); return r.json(); }

async function loadOverview(){
  try{
    const ov = await api("/api/overview");
    $("#stEntities").textContent = ov.entities;
    $("#stTools").textContent = ov.tools;
    $("#stSnaps").textContent = ov.snapshots;
    $("#stRepo").textContent = ov.repo_root;
    $("#navEntCount").textContent = ov.entities;
    // 健康分（简化：有实体即健康，预留扩展）
    const score = ov.entities>0 ? Math.min(100, 60+ov.entities) : 0;
    $("#healthScore").textContent = score;
    $("#ringEl").style.setProperty("--p", score);
    $("#healthTitle").textContent = ov.entities>0 ? "配置已纳管" : "尚未采集配置";
    $("#healthDesc").innerHTML = ov.entities>0
      ? "已纳管 <b>"+ov.entities+"</b> 项配置 · 覆盖 <b>"+ov.tools+"</b> 个工具 · <b>"+ov.snapshots+"</b> 个快照"
      : "点击「采集全部」把本机各 AI 工具的配置纳入统一管理";
  }catch(e){ $("#healthTitle").textContent="加载失败"; }
}
async function loadEntities(){
  try{
    ENTITIES = await api("/api/entities");
    renderEntities("");
    // 概览页的构成
    const byKind={};
    ENTITIES.forEach(e=>{byKind[e.kind]=(byKind[e.kind]||0)+1});
    const names={instruction:"指令",mcp:"MCP",skill:"技能",agent:"Agent",hook:"Hook",setting:"设置"};
    const kb=$("#kindBreakdown");
    if(kb){ kb.innerHTML = Object.keys(byKind).map(k=>
      '<div class="row"><span class="tag t-'+k.slice(0,3)+'">'+(names[k]||k)+'</span><span class="id">'+k+'</span><span class="meta">'+byKind[k]+' 项</span></div>'
    ).join("") || '<div class="row"><span class="meta">暂无配置</span></div>'; }
  }catch(e){ $("#entityList").innerHTML='<div class="row"><span class="meta">加载失败</span></div>'; }
}
function renderEntities(kind){
  const list=$("#entityList");
  const items = kind ? ENTITIES.filter(e=>e.kind===kind) : ENTITIES;
  if(items.length===0){ list.innerHTML='<div class="row"><span class="meta">该分类下暂无配置</span></div>'; return; }
  const names={instruction:"指令",mcp:"MCP",skill:"技能",agent:"Agent",hook:"Hook",setting:"设置"};
  list.innerHTML = items.map(e=>
    '<div class="row"><span class="tag t-'+e.kind.slice(0,3)+'">'+(names[e.kind]||e.kind)+'</span><span class="id">'+esc(e.id)+'</span><span class="meta">'+esc(e.note||"")+'</span></div>'
  ).join("");
}
function filterKind(el){
  document.querySelectorAll("#pg-entities .chip").forEach(c=>c.classList.remove("on"));
  el.classList.add("on");
  renderEntities(el.dataset.k);
}
async function loadSnapshots(){
  try{
    const snaps = await api("/api/snapshots");
    const list=$("#snapList");
    if(!snaps || snaps.length===0){ list.innerHTML='<div class="row"><span class="meta">暂无快照——点击右上角「创建快照」</span></div>'; return; }
    list.innerHTML = snaps.map(s=>
      '<div class="row"><span class="id">'+esc(s.id)+'</span><span class="meta">'+esc(s.note||"")+' · '+s.files+' 文件</span><button class="btn" onclick="doRestore(\''+s.id+'\')">恢复</button></div>'
    ).join("");
  }catch(e){ $("#snapList").innerHTML='<div class="row"><span class="meta">加载失败</span></div>'; }
}
// ---- 操作 ----
async function doCollect(){
  toast("正在采集各工具配置…");
  try{ const r=await api("/api/collect",{method:"POST"}); toast(r.message||"采集完成"); loadOverview(); }
  catch(e){ toast("采集失败"); }
}
async function doSnapshot(){
  toast("正在创建快照…");
  try{ const r=await api("/api/snapshot/create?note="+encodeURIComponent("手动"),{method:"POST"}); toast(r.message||"快照已创建"); loadSnapshots(); loadOverview(); }
  catch(e){ toast("创建失败"); }
}
async function doRestore(id){
  if(!confirm("恢复快照 "+id+"？\n会先对当前状态做一次保护性快照，再回滚。")) return;
  try{ const r=await api("/api/snapshot/restore?id="+encodeURIComponent(id),{method:"POST"}); toast(r.message||"已恢复"); loadOverview(); }
  catch(e){ toast("恢复失败"); }
}
function esc(s){ const d=document.createElement("div"); d.textContent=s; return d.innerHTML; }
// 启动
loadOverview(); loadEntities();
