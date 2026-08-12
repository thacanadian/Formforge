/* FormForge 1.7 — focused retro-terminal interface and motion layer. */
const ffV17PrimaryDesktop=['dashboard','coach','workouts','nutrition','progress','more'];
const ffV17PrimaryMobile=['dashboard','coach','workouts','nutrition','more'];
const ffV17SecondaryPages=new Set(['team','agent','health','habits','social','checkin','market','mobile','security','appearance','settings','admin','progress']);
let ffV17ScrollObserver=null;
let ffV17ClockTimer=null;

function ffV17Icon(id){
  const paths={
    dashboard:'<rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/>',
    coach:'<path d="M8 10a4 4 0 0 1 8 0v4a4 4 0 0 1-8 0z"/><path d="M12 6V3M5 12H3m18 0h-2M9.5 13h.01m4.99 0h.01M9 18c1.8 1.3 4.2 1.3 6 0"/>',
    workouts:'<path d="M6 8v8M3 10v4m15-6v8m3-6v4M6 12h12"/>',
    nutrition:'<path d="M7 3v8a3 3 0 0 1-3 3V3m3 5H4m8-5v18m0-10c5 0 7-3 7-8-5 0-7 3-7 8Z"/>',
    progress:'<path d="M4 19V9m6 10V5m6 14v-7m4 7H2"/><path d="m4 7 5-4 5 4 6-5"/>',
    more:'<circle cx="5" cy="12" r="1"/><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/>',
    health:'<path d="M20.8 4.6a5.5 5.5 0 0 0-7.8 0L12 5.7l-1.1-1.1a5.5 5.5 0 0 0-7.8 7.8L12 21l8.8-8.6a5.5 5.5 0 0 0 0-7.8Z"/>',
    team:'<circle cx="9" cy="7" r="4"/><path d="M3 21v-2a6 6 0 0 1 12 0v2m2-17a4 4 0 0 1 0 7.75M17 15a6 6 0 0 1 4 5.65V21"/>',
    agent:'<path d="M12 2 4 6v6c0 5 3.4 8.7 8 10 4.6-1.3 8-5 8-10V6Z"/><path d="m9 12 2 2 4-5"/>',
    settings:'<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .34 1.88l.06.06-2.83 2.83-.06-.06A1.7 1.7 0 0 0 15 19.4a1.7 1.7 0 0 0-1 .6 1.7 1.7 0 0 0-.4 1V21H10v-.08a1.7 1.7 0 0 0-1.1-1.58 1.7 1.7 0 0 0-1.88.34l-.06.06-2.83-2.83.06-.06A1.7 1.7 0 0 0 4.6 15a1.7 1.7 0 0 0-.6-1 1.7 1.7 0 0 0-1-.4H3V10h.08a1.7 1.7 0 0 0 1.58-1.1 1.7 1.7 0 0 0-.34-1.88l-.06-.06 2.83-2.83.06.06A1.7 1.7 0 0 0 9 4.6a1.7 1.7 0 0 0 1-.6 1.7 1.7 0 0 0 .4-1V3H14v.08a1.7 1.7 0 0 0 1.1 1.58 1.7 1.7 0 0 0 1.88-.34l.06-.06 2.83 2.83-.06.06A1.7 1.7 0 0 0 19.4 9c.15.38.37.72.65 1 .28.27.64.41 1 .4H21V14h-.08a1.7 1.7 0 0 0-1.52 1Z"/>',
    default:'<circle cx="12" cy="12" r="9"/><path d="m9 12 2 2 4-4"/>'
  };
  return `<svg class="ff-nav-icon" viewBox="0 0 24 24" aria-hidden="true" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">${paths[id]||paths.default}</svg>`;
}

function ffV17NavLabel(id){return ({dashboard:'Dashboard',coach:'AI Coach',workouts:'Workouts',nutrition:'Nutrition',progress:'Progress',more:'More'})[id]||navItems.find(x=>x[0]===id)?.[2]||id}
function ffV17NavHTML(ids,target){
  const active=ids.includes(state.page)?state.page:(ffV17SecondaryPages.has(state.page)?'more':'dashboard');
  return ids.map(id=>`<button class="nav-btn ${active===id?'active':''}" data-page="${id}" aria-label="${ffV17NavLabel(id)}"><span>${ffV17Icon(id)}</span><span>${ffV17NavLabel(id)}</span></button>`).join('');
}

buildNav=function(){
  const desktop=$('#desktop-nav'),mobile=$('#mobile-nav');
  const full=ffPreferences.navigationMode==='full'&&!ffMemberPreview();
  const desktopIds=full?navItems.filter(x=>(x[0]!=='admin'||state.user?.role==='admin')&&x[0]!=='more').map(x=>x[0]):ffV17PrimaryDesktop;
  if(desktop)desktop.innerHTML=ffV17NavHTML(desktopIds,'desktop');
  if(mobile)mobile.innerHTML=ffV17NavHTML(ffV17PrimaryMobile,'mobile');
  $$('[data-page]').forEach(b=>b.onclick=()=>{state.page=b.dataset.page;renderPage()});
  if($('#user-chip'))$('#user-chip').textContent=ffMemberPreview()?`${state.user.name} · member preview`:`${state.user.name} · ${state.user.role}`;
  ffV17UpdateChrome();
};

function ffV17UpdateChrome(){
  $('#appearance-shortcut')?.remove();
  $('#member-preview-toggle')?.remove();
  const actions=$('.top-actions');
  if(actions&&!$('#ff-live-status'))actions.insertAdjacentHTML('afterbegin','<div id="ff-live-status" class="ff-live-status"><span class="ff-online-dot"></span><span>SYNC</span><time id="ff-clock"></time></div>');
  const tick=()=>{const el=$('#ff-clock');if(el)el.textContent=new Date().toLocaleTimeString([], {hour:'2-digit',minute:'2-digit'})};
  tick();
  if(!ffV17ClockTimer)ffV17ClockTimer=setInterval(tick,30000);
  if(!$('#ff-scroll-progress'))document.body.insertAdjacentHTML('afterbegin','<div id="ff-scroll-progress" aria-hidden="true"></div>');
  if(!document.documentElement.dataset.ffScrollBound){
    document.documentElement.dataset.ffScrollBound='1';
    const update=()=>{const max=document.documentElement.scrollHeight-innerHeight;const pct=max>0?Math.min(100,Math.max(0,scrollY/max*100)):0;document.documentElement.style.setProperty('--ff-scroll',pct+'%')};
    addEventListener('scroll',update,{passive:true});addEventListener('resize',update,{passive:true});update();
  }
}

function ffV17Metric(metrics,names){
  const wanted=new Set(names.map(x=>x.toLowerCase()));
  return (metrics||[]).find(x=>wanted.has(String(x.metricType||'').toLowerCase()))||null;
}
function ffV17MetricText(metric,fallback='No data'){
  if(!metric||!Number.isFinite(+metric.value))return fallback;
  const value=Math.round((+metric.value)*10)/10;
  return `${value}${metric.unit?` ${esc(metric.unit)}`:''}`;
}
function ffV17Pct(value,goal){return goal>0?Math.max(0,Math.min(100,Math.round(value/goal*100))):0}
function ffV17Day(){return ['Sun','Mon','Tue','Wed','Thu','Fri','Sat'][new Date().getDay()]}
function ffV17Torso(){return '<svg class="ff-torso" viewBox="0 0 180 180" aria-hidden="true"><circle cx="90" cy="34" r="18"/><path d="M61 68c8-13 19-19 29-19s21 6 29 19l12 32-18 12-8-25v70H75V87l-8 25-18-12Z"/><path d="M76 78c8 7 20 7 28 0M90 50v108M69 111h42"/></svg>'}

const ffV17BaseDashboard=renderDashboard;
renderDashboard=async function(){
  const results=await Promise.allSettled([
    api('/api/dashboard'),api('/api/recovery-score'),api('/api/health?limit=120'),api('/api/ai/status'),api('/api/coaching/pack'),api('/api/workouts')
  ]);
  if(results[0].status!=='fulfilled')throw results[0].reason;
  const d=results[0].value||{},recovery=results[1].status==='fulfilled'?results[1].value:{score:70,status:'ready',reasons:[]};
  const health=results[2].status==='fulfilled'?results[2].value:{metrics:[]};
  const ai=results[3].status==='fulfilled'?results[3].value:{mode:'offline',apiKeyConfigured:false};
  const pack=results[4].status==='fulfilled'?results[4].value:{blend:'FormForge Balanced'};
  const workouts=results[5].status==='fulfilled'?results[5].value:[];
  const t=d.today||{},p=d.profile||{},day=ffV17Day(),planName=String(d.plan?.[day]||'').trim();
  const planned=workouts.find(x=>x.name===planName)||null;
  const recoveryScore=Math.max(0,Math.min(100,+recovery.score||0));
  const metrics=health.metrics||[];
  const sleep=ffV17Metric(metrics,['sleep','sleep_hours','sleep duration']);
  const hrv=ffV17Metric(metrics,['hrv','heart_rate_variability']);
  const rhr=ffV17Metric(metrics,['rhr','resting_heart_rate','resting heart rate']);
  const stress=ffV17Metric(metrics,['stress','stress_score']);
  const totalMacros=(+t.protein||0)+(+t.carbs||0)+(+t.fat||0);
  const proteinPct=totalMacros?Math.round((+t.protein||0)/totalMacros*100):0;
  const carbsPct=totalMacros?Math.round((+t.carbs||0)/totalMacros*100):0;
  const fatPct=Math.max(0,100-proteinPct-carbsPct);
  const firstName=String(state.user?.name||p.name||'Athlete').split(' ')[0];
  const coachCopy=recoveryScore>=75?`Recovery looks strong, ${firstName}. Keep the plan or ask me to fine-tune it.`:recoveryScore>=55?`You are in a workable range, ${firstName}. Train with control and leave a little in reserve.`:`Recovery is low today, ${firstName}. I can reduce volume and keep the session productive.`;
  const workoutTitle=planned?.name||(planName||'Recovery / open session');
  const workoutCategory=planned?.category||(planName?'Planned session':'Choose a workout');
  const workoutDuration=planned?.duration||0;
  const recoveryLabel=String(recovery.status||'ready').toUpperCase();
  $('#content').innerHTML=`
    <div class="ff-command-grid">
      <article class="card ff-panel ff-workout-panel ff-reveal">
        <div class="ff-panel-head"><div><p class="eyebrow">TODAY'S WORKOUT</p><h2>${esc(workoutTitle)}</h2><p class="muted">${esc(workoutCategory)}${workoutDuration?` · ${workoutDuration} min`:''}</p></div><span class="pill">${esc(day)}</span></div>
        <div class="ff-workout-body"><div class="ff-workout-details"><div><small>FOCUS</small><strong>${planned?.exercises?.slice(0,3).map(x=>esc(x.name)).join(' · ')||'Pick the session that fits today'}</strong></div><div><small>WEEKLY WORKOUTS</small><strong>${d.weeklyWorkouts||0} completed</strong></div></div>${ffV17Torso()}</div>
        <button class="primary-btn ff-wide-action" id="ff-start-workout">${planned?'Start workout':'Choose workout'} <span>›</span></button>
      </article>

      <article class="card ff-panel ff-recovery-panel ff-reveal">
        <div class="ff-panel-head"><div><p class="eyebrow">RECOVERY SCORE</p><h2>Readiness</h2></div><span class="pill ${recoveryScore<55?'ff-warning-pill':''}">${recoveryLabel}</span></div>
        <div class="ff-recovery-body"><div class="ff-gauge" style="--score:${recoveryScore*3.6}deg"><div><strong>${recoveryScore}</strong><small>/100</small></div></div><dl class="ff-metrics"><div><dt>HRV</dt><dd>${ffV17MetricText(hrv)}</dd></div><div><dt>SLEEP</dt><dd>${ffV17MetricText(sleep)}</dd></div><div><dt>RHR</dt><dd>${ffV17MetricText(rhr)}</dd></div><div><dt>STRESS</dt><dd>${ffV17MetricText(stress,'—')}</dd></div></dl></div>
        <button class="outline-btn ff-wide-action" data-go="health">View recovery <span>›</span></button>
      </article>

      <article class="card ff-panel ff-coach-panel ff-reveal">
        <div class="ff-panel-head"><div><p class="eyebrow">AI COACH</p><h2>${esc(pack.blend||'FormForge Coach')}</h2></div><span class="ff-status-text"><i></i>${ai.apiKeyConfigured?'ONLINE + OFFLINE':'OFFLINE READY'}</span></div>
        <div class="ff-coach-message"><div class="ff-coach-orb">${ffV17Icon('coach')}</div><p>${esc(coachCopy)}</p></div>
        <div class="ff-action-stack"><button class="primary-btn" id="ff-open-coach">Open coach <span>›</span></button><button class="outline-btn" id="ff-adjust-plan">Adjust today's plan</button></div>
      </article>

      <article class="card ff-panel ff-nutrition-panel ff-reveal">
        <div class="ff-panel-head"><div><p class="eyebrow">NUTRITION SUMMARY</p><h2>Today's fuel</h2></div><span class="pill">${ffV17Pct(+t.calories||0,+p.calorieGoal||0)}%</span></div>
        <div class="ff-calorie-line"><strong>${Math.round(+t.calories||0).toLocaleString()}</strong><span>/ ${Math.round(+p.calorieGoal||0).toLocaleString()} kcal</span></div>
        <div class="ff-long-progress"><i style="width:${ffV17Pct(+t.calories||0,+p.calorieGoal||0)}%"></i></div>
        <div class="ff-macro-grid"><div><i></i><small>PROTEIN</small><strong>${Math.round(+t.protein||0)}g</strong><span>${proteinPct}%</span></div><div><i></i><small>CARBS</small><strong>${Math.round(+t.carbs||0)}g</strong><span>${carbsPct}%</span></div><div><i></i><small>FATS</small><strong>${Math.round(+t.fat||0)}g</strong><span>${fatPct}%</span></div></div>
        <button class="outline-btn ff-wide-action" data-go="nutrition">View nutrition <span>›</span></button>
      </article>
    </div>`;
  $('#ff-start-workout').onclick=()=>planned?ffV17OpenWorkoutSession(planned):ffV17Go('workouts');
  $('#ff-open-coach').onclick=()=>ffV17Go('coach');
  $('#ff-adjust-plan').onclick=()=>ffV17CoachPrompt(`My recovery score is ${recoveryScore}/100. Adjust today's ${planned?.name||'workout'} so it matches my readiness while preserving the main goal.`);
  $$('[data-go]').forEach(b=>b.onclick=()=>ffV17Go(b.dataset.go));
};

function ffV17Go(page){state.page=page;renderPage()}
function ffV17CoachPrompt(prompt){try{sessionStorage.setItem('ff_v17_coach_prompt',prompt)}catch{}ffV17Go('coach')}

function ffV17OpenWorkoutSession(workout){
  const exercises=workout.exercises||[];
  modal(workout.name,`<section class="ff-session"><div class="ff-session-summary"><div><p class="eyebrow">ACTIVE SESSION</p><strong id="ff-session-time">00:00</strong><small>${exercises.length} exercises · ${workout.duration||0} min target</small></div><span class="ff-session-pulse"></span></div><div class="ff-session-list">${exercises.map((x,i)=>`<button type="button" class="ff-session-exercise" data-session-index="${i}"><span class="ff-session-check">${i+1}</span><span><strong>${esc(x.name)}</strong><small>${x.sets||0} sets × ${esc(x.reps||'')} · Rest ${esc(x.rest||'—')}</small></span></button>`).join('')||'<div class="empty">No exercises in this workout.</div>'}</div><div class="button-row"><button class="primary-btn" id="ff-finish-session">Finish and log</button><button class="outline-btn" id="ff-cancel-session">Pause</button></div></section>`,'WORKOUT MODE');
  const started=Date.now();
  const timer=setInterval(()=>{const total=Math.floor((Date.now()-started)/1000),m=String(Math.floor(total/60)).padStart(2,'0'),s=String(total%60).padStart(2,'0');if($('#ff-session-time'))$('#ff-session-time').textContent=`${m}:${s}`;else clearInterval(timer)},1000);
  $$('.ff-session-exercise').forEach(b=>b.onclick=()=>{b.classList.toggle('done');b.querySelector('.ff-session-check').textContent=b.classList.contains('done')?'✓':String(+b.dataset.sessionIndex+1)});
  $('#ff-finish-session').onclick=()=>{clearInterval(timer);closeModal();openWorkoutLogModal(workout)};
  $('#ff-cancel-session').onclick=()=>{clearInterval(timer);closeModal();toast('Workout paused. Nothing was logged.')};
  $('#modal').addEventListener('close',()=>clearInterval(timer),{once:true});
}

const ffV17BaseCoach=renderCoach;
renderCoach=async function(){
  await ffV17BaseCoach();
  let prompt='';try{prompt=sessionStorage.getItem('ff_v17_coach_prompt')||'';sessionStorage.removeItem('ff_v17_coach_prompt')}catch{}
  if(prompt&&$('#coach-form textarea')){$('#coach-form textarea').value=prompt;$('#coach-form textarea').focus()}
};

renderMore=async function(){
  const groups=[
    ['TRAINING + COACHING',[
      ['team','team','Coaching Team','Blend evidence-based coaching influences without impersonation.'],
      ['agent','agent','AI Agent','Local autonomous tasks, memory and optional researched answers.'],
      ['progress','progress','Progress','Body measurements, performance history and trends.'],
      ['health','health','Health + Recovery','Wearables, pain flags, recovery, photos and meal plans.']]],
    ['CONSISTENCY + COMMUNITY',[
      ['habits','default','Habits','Daily consistency, streaks and routines.'],
      ['checkin','default','Weekly Check-In','Review readiness, schedule and recovery.'],
      ['social','team','Community','Leaderboards, nudges and shared workouts.'],
      ['market','default','Marketplace','Programs, meal plans and coach packs.']]],
    ['APP + ACCOUNT',[
      ['mobile','default','Mobile App','Install the PWA and connect phones or tablets.'],
      ['appearance','default','Appearance + Units','Terminal themes, density, colors and metric/imperial units.'],
      ['security','settings','Security','Two-factor authentication, sessions and protected exports.'],
      ['settings','settings','Profile + Data','Goals, profile, imports, exports, backup and network settings.']]]
  ];
  if(state.user?.role==='admin')groups.push(['ADMINISTRATION',[[
    'admin','settings','Users + Audit','Manage accounts, access, plan tiers and audit history.'
  ]]]);
  $('#content').innerHTML=`<div class="ff-more-intro ff-reveal"><div><p class="eyebrow">EVERYTHING IN FORMFORGE</p><h2>Focused by default. Nothing removed.</h2><p class="muted">The main navigation stays calm; every advanced feature remains available here.</p></div>${state.user?.role==='admin'?`<button class="outline-btn" id="ff-preview-toggle">${ffMemberPreview()?'Exit member preview':'Preview member view'}</button>`:''}</div>${groups.map(([title,items])=>`<section class="ff-more-section"><div class="section-head"><h2>${title}</h2></div><div class="feature-grid ff-feature-grid">${items.map(([id,icon,title,copy])=>`<button class="feature-card ff-reveal" data-open-page="${id}"><span>${ffV17Icon(icon)}</span><div><strong>${title}</strong><small>${copy}</small></div><b>›</b></button>`).join('')}</div></section>`).join('')}`;
  $$('[data-open-page]').forEach(b=>b.onclick=()=>ffV17Go(b.dataset.openPage));
  if($('#ff-preview-toggle'))$('#ff-preview-toggle').onclick=()=>{ffSetMemberPreview(!ffMemberPreview());buildNav();renderMore()};
};

function ffV17AnimatePage(){
  const content=$('#content');if(!content)return;
  if(matchMedia('(prefers-reduced-motion: reduce)').matches){$$('.ff-reveal,.card,.feature-card,.list-row,.section-head',content).forEach(x=>x.classList.add('is-visible'));return}
  const nodes=$$('.ff-reveal,.card,.feature-card,.list-row,.section-head',content).filter((x,i,a)=>a.indexOf(x)===i);
  nodes.forEach((el,i)=>{el.classList.add('ff-reveal');el.style.setProperty('--ff-delay',`${Math.min(i,10)*45}ms`)});
  if(!('IntersectionObserver' in window)){nodes.forEach(el=>el.classList.add('is-visible'));return}
  ffV17ScrollObserver?.disconnect();
  ffV17ScrollObserver=new IntersectionObserver(entries=>entries.forEach(entry=>{if(entry.isIntersecting){entry.target.classList.add('is-visible');ffV17ScrollObserver.unobserve(entry.target)}}),{threshold:.08,rootMargin:'0px 0px -24px 0px'});
  nodes.forEach(el=>ffV17ScrollObserver.observe(el));
}

const ffV17BaseRenderPage=renderPage;
renderPage=async function(){
  const content=$('#content');if(content){content.classList.remove('ff-page-enter');content.classList.add('ff-page-exit')}
  await ffV17BaseRenderPage();
  ffV17UpdateChrome();
  requestAnimationFrame(()=>{const c=$('#content');if(c){c.classList.remove('ff-page-exit');c.classList.add('ff-page-enter')}ffV17AnimatePage()});
};

const ffV17BaseShowApp=showApp;
showApp=function(){ffV17BaseShowApp();document.documentElement.dataset.ffVersion='17';ffV17UpdateChrome();buildNav()};

// The app may already be visible when this late-loaded layer executes during a hot refresh.
if(!$('#app-shell')?.classList.contains('hidden')){document.documentElement.dataset.ffVersion='17';ffV17UpdateChrome();buildNav();renderPage()}
