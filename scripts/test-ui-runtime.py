#!/usr/bin/env python3
"""Optional FormForge browser runtime smoke test using Chrome DevTools Protocol.

Requires a Chromium/Chrome process launched with --remote-debugging-port and
Python packages requests + websocket-client. The test uses an in-memory data URL
and mocked API responses, so it does not touch live FormForge data.
"""
from __future__ import annotations

import base64
import json
import sys
import time
from pathlib import Path
from urllib.parse import quote

import requests
import websocket

ROOT = Path(__file__).resolve().parents[1]
DEBUG_BASE = sys.argv[1] if len(sys.argv) > 1 else "http://127.0.0.1:9223"


def cdp(ws_url: str):
    ws = websocket.create_connection(ws_url, timeout=20, origin=DEBUG_BASE)
    seq = 0

    def call(method: str, params: dict | None = None):
        nonlocal seq
        seq += 1
        ws.send(json.dumps({"id": seq, "method": method, "params": params or {}}))
        while True:
            msg = json.loads(ws.recv())
            if msg.get("id") == seq:
                if "error" in msg:
                    raise RuntimeError(msg["error"])
                return msg.get("result", {})

    return ws, call


def main() -> int:
    index = (ROOT / "web" / "index.html").read_text(encoding="utf-8")
    v14 = (ROOT / "web" / "v14.js").read_text(encoding="utf-8")
    coaching = (ROOT / "web" / "coaching-ui.js").read_text(encoding="utf-8")
    app = (ROOT / "web" / "app.js").read_text(encoding="utf-8")
    v15 = (ROOT / "web" / "v15.js").read_text(encoding="utf-8")
    v16 = (ROOT / "web" / "v16.js").read_text(encoding="utf-8")
    v17 = (ROOT / "web" / "v17.js").read_text(encoding="utf-8")
    v18 = (ROOT / "web" / "v18.js").read_text(encoding="utf-8")
    # Keep the production DOM, but remove external resources and inject a mock API.
    index = index.replace('<link rel="manifest" href="/manifest.webmanifest">', "")
    index = index.replace('<link rel="icon" href="/icon.svg" type="image/svg+xml">', "")
    for resource in [
        '<link rel="stylesheet" href="/styles.css?v=180">',
        '<link rel="stylesheet" href="/v17.css?v=180">',
        '<link rel="stylesheet" href="/v18.css?v=180">',
        '<script src="/v14.js?v=180" defer></script>',
        '<script src="/coaching-ui.js?v=180" defer></script>',
        '<script src="/app.js?v=180" defer></script>',
        '<script src="/v15.js?v=180" defer></script>',
        '<script src="/v16.js?v=180" defer></script>',
        '<script src="/v17.js?v=180" defer></script>',
        '<script src="/v18.js?v=180" defer></script>',
    ]:
        index = index.replace(resource, "")

    mock = r'''
<script>
window.__fetchCalls=[]; window.__errors=[]; window.addEventListener('error',e=>window.__errors.push(String(e.message)+' @ '+e.filename+':'+e.lineno)); window.addEventListener('unhandledrejection',e=>window.__errors.push('promise: '+String(e.reason)));
const profile={name:'Runtime Tester',age:25,gender:'male',heightCm:180,weightKg:82,goalWeightKg:85,goal:'Build muscle',experience:'intermediate',daysPerWeek:4,equipment:'Dumbbells only',calorieGoal:2800,proteinGoal:180,carbGoal:300,fatGoal:80};
const user={id:'u1',name:'Runtime Tester',email:'runtime@example.test',role:'admin',active:true,planTier:'pro'};
function payload(path,opt){
  const method=(opt&&opt.method)||'GET';
  if(path.startsWith('/api/system/status')) return {setupRequired:false,version:'1.8.0'};
  if(path.startsWith('/api/auth/session')) return {user,csrf:'test-csrf'};
  if(path.startsWith('/api/dashboard')) return {today:{calories:0,protein:0,carbs:0,fat:0,habitDone:0,habitTotal:0},profile,weeklyWorkouts:0,streak:0,plan:{},recentWorkouts:[]};
  if(path.startsWith('/api/profile')) return profile;
  if(path.startsWith('/api/coaching/team')) return {blend:'60% Jeff Nippard · 40% FormForge Balanced',disclosure:'Editorial profiles do not imply endorsement.',preferences:{responseStyle:'teach',preferredCoachId:'jeff-nippard',influences:[{profileId:'jeff-nippard',weight:60},{profileId:'formforge-balanced',weight:40}]},profiles:[{id:'jeff-nippard',name:'Jeff Nippard',initials:'JN',category:'Evidence-based hypertrophy',status:'editorial',summary:'Evidence-focused programming.',principles:['Use progressive overload.'],communication:['calm','analytical'],safetyNote:'Unofficial editorial profile.',sourceCount:0},{id:'formforge-balanced',name:'FormForge Balanced',initials:'FF',category:'General fitness',status:'official',summary:'Balanced coaching.',principles:['Use repeatable plans.'],communication:['clear'],safetyNote:'Official FormForge profile.',sourceCount:0}],sources:[]};
  if(path==='/api/coaching/profiles') return [];
  if(path.startsWith('/api/coaching/takedowns')) return [];
  if(path.startsWith('/api/coaching/link-preview')) return {platform:'youtube',url:'https://youtube.com/test',title:'Test Coach',handle:'Test'};
  if(path.startsWith('/api/coaching/pack')) return {blend:'60% Jeff Nippard · 40% FormForge Balanced',preferences:{responseStyle:'teach',preferredCoachId:'jeff-nippard',influences:[{profileId:'jeff-nippard',weight:60},{profileId:'formforge-balanced',weight:40}]},profiles:[],sources:[]};
  if(path.startsWith('/api/coaching/preferences')) return {responseStyle:'teach',influences:[{profileId:'jeff-nippard',weight:60},{profileId:'formforge-balanced',weight:40}]};
  if(path.startsWith('/api/coaching/sources')) return {ok:true};
  if(path.startsWith('/api/ai/status')) return {mode:'auto',baseUrl:'https://api.openai.com/v1',model:'gpt-4o-mini',apiKeyConfigured:false,offlineReady:true};
  if(path.startsWith('/api/ai/history')) return [];
  if(path.startsWith('/api/ai/usage')) return {date:'2026-07-24',planTier:'pro',inputTokens:0,outputTokens:0,totalTokens:0,costMicros:0,dailyTokenCap:50000,dailyCostCapMicros:2500000,onlineAllowed:true};
  if(path.startsWith('/api/ai/chat')) return {reply:'Offline coach response: 4-day dumbbell plan ready.',mode:'offline',grounding:[{kind:'general_knowledge',label:'General fitness knowledge'}]};
  if(path.startsWith('/api/ai/test')) return {reply:'Online connection test passed.'};
  if(path.startsWith('/api/ai/settings')) return {ok:true};
  if(path.startsWith('/api/workouts')) return [{id:'w1',name:'Upper Power',category:'Strength',duration:50,builtIn:true,why:'Build strength.',exercises:[{name:'Dumbbell Press',sets:4,reps:'6-8',rest:'2min',why:'Compound press.'}]}];
  if(path.startsWith('/api/plan')) return {};
  if(path.startsWith('/api/workout-logs')) return [];
  if(path.startsWith('/api/nutrition')) return [];
  if(path.startsWith('/api/food-search')) return [];
  if(path.startsWith('/api/food-lookup')) return {name:'Test Food',serving:'100g',calories:100,protein:10,carbs:10,fat:2};
  if(path.startsWith('/api/health/providers')) return [{id:'apple-health',name:'Apple Health'},{id:'google-fit',name:'Google Fit'},{id:'garmin',name:'Garmin'},{id:'whoop',name:'WHOOP'},{id:'oura',name:'Oura'},{id:'hr-strap',name:'Heart-rate strap'}];
  if(path.startsWith('/api/health')) return {metrics:[],connections:[]};
  if(path.startsWith('/api/pain-flags')) return [];
  if(path.startsWith('/api/training/progression')) return [];
  if(path.startsWith('/api/progress-photos')) return [];
  if(path.startsWith('/api/meal-plans')) return [];
  if(path.startsWith('/api/social/leaderboard')) return [{userId:'u1',name:'Runtime Tester',workouts:0,streak:0,habits:0}];
  if(path.startsWith('/api/social/users')) return [user];
  if(path.startsWith('/api/social/nudges')) return [];
  if(path.startsWith('/api/social/workouts')) return [];
  if(path.startsWith('/api/security/2fa')) return {enabled:false,adminOnly:true};
  if(path.startsWith('/api/security/sessions')) return [{id:'sess1',current:true,ip:'127.0.0.1',deviceName:'Test browser',createdAt:new Date().toISOString(),lastSeenAt:new Date().toISOString()}];
  if(path.startsWith('/api/legal/terms')) return {version:'2.0',versions:{terms:'2.0',privacy:'1.0',community:'1.0',subscription:'1.0'},text:'Test terms',takedownContact:'admin@example.test',config:{configured:true,minimumAge:18,warnings:[]},links:{terms:'/legal/terms',privacy:'/legal/privacy',community:'/legal/community'}};
  if(path.startsWith('/api/legal/status')) return {versions:{terms:'2.0',privacy:'1.0',community:'1.0',subscription:'1.0'},accepted:{terms:true,privacy:true,community:true,age:true},eligibleForCommunity:true,config:{configured:true,minimumAge:18,warnings:[]},links:{terms:'/legal/terms',privacy:'/legal/privacy',community:'/legal/community'}};
  if(path.startsWith('/api/social/blocks')) return [];
  if(path.startsWith('/api/social/reports')) return [];
  if(path.startsWith('/api/admin/moderation/reports')) return [];
  if(path.startsWith('/api/system/health')) return {status:'up',ok:true,version:'1.8.0',checks:{schemaVersion:6,users:1,failedJobs:0},issues:[]};
  if(path.startsWith('/api/update/status')) return {configured:false,currentVersion:'1.8.0',message:'Not configured'};
  if(path.startsWith('/api/progress')) return [];
  if(path.startsWith('/api/habits')) return [];
  if(path.startsWith('/api/checkins')) return [];
  if(path.startsWith('/api/admin/users')) return [user];
  if(path.startsWith('/api/admin/audit')) return [];
  if(path.startsWith('/api/system/mobile')) return {enabled:false,port:8443,urls:['https://192.168.1.25:8443'],currentUrl:'https://127.0.0.1:8443',caUrl:'/api/system/ca'};
  if(path.startsWith('/api/theme')) return {current:{preset:'forge',accent:'#ff7a1a',background:'#08090a',surface:'#111315',text:'#f2e7dc',radius:10,density:'comfortable',measurementSystem:'imperial',navigationMode:'focused'},presets:{forge:{preset:'forge',accent:'#ff7a1a',background:'#08090a',surface:'#111315',text:'#f2e7dc',radius:10,density:'comfortable'},midnight:{preset:'midnight',accent:'#7c5cff',background:'#080b17',surface:'#12182b',text:'#f4f6ff',radius:18,density:'comfortable'},iron:{preset:'iron',accent:'#d4af37',background:'#111111',surface:'#222222',text:'#f5f1e6',radius:6,density:'compact'},arctic:{preset:'arctic',accent:'#0077ff',background:'#f2f6fb',surface:'#ffffff',text:'#152033',radius:16,density:'comfortable'},forest:{preset:'forest',accent:'#43a047',background:'#0d1711',surface:'#17251b',text:'#eff8f0',radius:12,density:'comfortable'}}};
  if(path.startsWith('/api/knowledge/status')) return {builtInChunks:58,domains:['Hypertrophy','Strength','Nutrition','Recovery'],approvedSources:0,verifiedQuotes:0};
  if(path.startsWith('/api/recovery-score')) return {score:82,status:'ready',reasons:[]};
  if(path.startsWith('/api/agent/tasks')) return [];
  if(path.startsWith('/api/agent/memories')) return [];
  if(path.startsWith('/api/agent/settings')) return {enabled:true,allowWeb:true,baseUrl:'http://127.0.0.1:11434/v1',model:'llama3.1:8b',searchUrl:'http://127.0.0.1:8080/search',maxSteps:8};
  if(path.startsWith('/api/marketplace')) return [];
  if(path.startsWith('/api/settings')) return {lanEnabled:false,port:8443,backupCopyPath:'',aiDailyTokenCap:50000,aiDailyCostCapMicros:2500000,freeDailyTokenCap:0,freeDailyCostCapMicros:0,backupIntervalHours:24,termsVersion:'1.4'};
  if(path.startsWith('/api/backups')) return [];
  return {ok:true,method};
}
window.fetch=async function(input,opt={}){
  const path=typeof input==='string'?input:input.url;
  window.__fetchCalls.push({path,method:opt.method||'GET'});
  const body=payload(path,opt);
  return new Response(JSON.stringify(body),{status:200,headers:{'content-type':'application/json'}});
};
window.confirm=()=>true;
window.prompt=()=>'';
</script>
'''
    scripts = [v14, coaching, app, v15, v16, v17, v18]
    injected = mock + "".join("<script>\n" + src.replace("</script>", "<" + "\\/" + "script>") + "\n</script>" for src in scripts)
    html = index.replace("</body>", injected + "</body>")
    target = requests.put(DEBUG_BASE + "/json/new?" + quote("about:blank", safe=""), timeout=10).json()
    ws, call = cdp(target["webSocketDebuggerUrl"])
    try:
        call("Runtime.enable")
        call("Page.enable")
        frame = call("Page.getFrameTree")["frameTree"]["frame"]["id"]
        call("Page.setDocumentContent", {"frameId": frame, "html": html})
        deadline = time.time() + 15
        while time.time() < deadline:
            result = call("Runtime.evaluate", {"expression": "document.querySelector('#app-shell') && !document.querySelector('#app-shell').classList.contains('hidden')", "returnByValue": True})
            if result.get("result", {}).get("value") is True:
                break
            time.sleep(0.2)
        else:
            body = call("Runtime.evaluate", {"expression": "document.body.innerText", "returnByValue": True})
            errs = call("Runtime.evaluate", {"expression": "(()=>{let manual='';try{showApp();manual='ok'}catch(e){manual=String(e.stack||e)}return JSON.stringify({errors:window.__errors,state:typeof state!=='undefined'?state:null,app:document.querySelector('#app-shell')?.className,auth:document.querySelector('#auth-shell')?.className,calls:window.__fetchCalls,manual})})()", "returnByValue": True})
            raise RuntimeError("App did not boot: " + str(body) + " DEBUG=" + str(errs))

        pages = ["dashboard", "team", "coach", "agent", "workouts", "nutrition", "health", "progress", "habits", "social", "checkin", "market", "mobile", "security", "admin", "appearance", "more", "settings"]
        failures = []
        for page in pages:
            expression = f'''(async()=>{{state.page={json.dumps(page)};await renderPage();return {{title:document.querySelector('#page-title').textContent,error:[...document.querySelectorAll('#content .notice.danger')].find(x=>x.textContent.includes('Could not load this screen'))?.textContent||'',text:document.querySelector('#content').innerText.slice(0,180)}}}})()'''
            out = call("Runtime.evaluate", {"expression": expression, "awaitPromise": True, "returnByValue": True})
            value = out.get("result", {}).get("value", {})
            if value.get("error") or not value.get("text"):
                failures.append({page: value})
            print(f"PASS screen {page}: {value.get('title')}")

        chat_expr = '''(async()=>{state.page='coach';await renderPage();const form=document.querySelector('#coach-form');form.elements.message.value='Give me a 4-day dumbbell workout';form.dispatchEvent(new Event('submit',{bubbles:true,cancelable:true}));for(let i=0;i<50;i++){await new Promise(r=>setTimeout(r,50));if(document.querySelectorAll('.chat-assistant').length)return document.querySelector('#chat-history').innerText;}return '';})()'''
        chat = call("Runtime.evaluate", {"expression": chat_expr, "awaitPromise": True, "returnByValue": True}).get("result", {}).get("value", "")
        if "Offline coach response" not in chat:
            failures.append({"coach_chat": chat})
        else:
            print("PASS AI Coach message submission")

        offline_expr = "localCoachReply('Build me a 3-day workout',{name:'Mobile Tester',goal:'Build muscle',daysPerWeek:3,equipment:'Dumbbells only'})"
        offline = call("Runtime.evaluate", {"expression": offline_expr, "returnByValue": True}).get("result", {}).get("value", "")
        if "3-day offline plan" not in offline or "Dumbbell" not in offline or "Coach blend" not in offline:
            failures.append({"phone_offline_coach": offline})
        else:
            print("PASS phone offline coach fallback")

        errors = call("Runtime.evaluate", {"expression": "[...document.querySelectorAll('#content .notice.danger')].filter(x=>x.textContent.includes('Could not load this screen')).length", "returnByValue": True}).get("result", {}).get("value", 0)
        if errors:
            failures.append({"danger_notices": errors})

        if failures:
            print(json.dumps(failures, indent=2), file=sys.stderr)
            return 1
        print("All FormForge UI runtime smoke tests passed.")
        return 0
    finally:
        ws.close()
        try:
            requests.get(DEBUG_BASE + "/json/close/" + target["id"], timeout=5)
        except Exception:
            pass


if __name__ == "__main__":
    raise SystemExit(main())
