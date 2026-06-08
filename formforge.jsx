import { useState, useEffect, useRef } from "react";

// ── PALETTE (Intense Gym + Forest Green) ───────────────────────────────────
const C = {
  bg:       "#232729",   // Black Beauty
  surface:  "#2d3235",
  card:     "#47484a",   // Midnight Magic tint
  border:   "#3a3d40",
  muted:    "#657079",   // Stormy Weather
  light:    "#8194a1",   // Coastal Vista
  sand:     "#c3af9f",   // Frontier Fort
  silver:   "#d9dada",   // Silver Setting
  green:    "#3a6b35",   // Forest green
  greenLt:  "#4e8f48",
  greenXlt: "#6aad63",
  accent:   "#5a9e52",   // Main CTA
  text:     "#e8e4e0",
  textDim:  "#a09890",
  danger:   "#c0392b",
  warn:     "#c3af9f",
};

const GS = `
@import url('https://fonts.googleapis.com/css2?family=Syne:wght@500;600;700;800&family=DM+Sans:wght@300;400;500;600&display=swap');
*{box-sizing:border-box;margin:0;padding:0;}
html,body{background:${C.bg};color:${C.text};font-family:'DM Sans',sans-serif;}
::-webkit-scrollbar{width:3px;}
::-webkit-scrollbar-thumb{background:${C.border};border-radius:2px;}
select,input,textarea{font-family:'DM Sans',sans-serif;color:${C.text};}
select:focus,input:focus,textarea:focus{outline:none;border-color:${C.accent}!important;box-shadow:0 0 0 3px rgba(90,158,82,.12)!important;}
@keyframes fadeUp{from{opacity:0;transform:translateY(12px)}to{opacity:1;transform:translateY(0)}}
@keyframes pulse{0%,100%{opacity:.25;transform:scale(.75)}50%{opacity:1;transform:scale(1)}}
@keyframes scanline{0%{top:-4px}100%{top:100%}}
@keyframes shimmer{0%{opacity:.6}50%{opacity:1}100%{opacity:.6}}
.fadeUp{animation:fadeUp .4s ease forwards;}
.btn{cursor:pointer;border:none;transition:all .18s;}
.btn:hover{filter:brightness(1.08);}
.btn:active{transform:scale(.97);}
`;

// ── STORAGE ────────────────────────────────────────────────────────────────
const S = {
  get: (k, d) => { try { const v = localStorage.getItem(k); return v ? JSON.parse(v) : d; } catch { return d; } },
  set: (k, v) => { try { localStorage.setItem(k, JSON.stringify(v)); } catch {} },
};

const today = () => new Date().toISOString().slice(0, 10);

// ── CLAUDE API ─────────────────────────────────────────────────────────────
const claude = async (messages, system = "") => {
  const body = { model: "claude-sonnet-4-20250514", max_tokens: 1000, messages };
  if (system) body.system = system;
  const r = await fetch("https://api.anthropic.com/v1/messages", {
    method: "POST", headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const d = await r.json();
  return d.content?.map(b => b.text || "").join("") || "";
};

// ── SHARED UI ──────────────────────────────────────────────────────────────
const syne = { fontFamily: "'Syne', sans-serif" };

const Card = ({ children, style = {}, className = "" }) => (
  <div className={className} style={{ background: C.surface, border: `1px solid ${C.border}`, borderRadius: 12, padding: "18px 16px", ...style }}>
    {children}
  </div>
);

const Label = ({ children, style = {} }) => (
  <div style={{ fontSize: 11, fontWeight: 600, letterSpacing: "1.5px", textTransform: "uppercase", color: C.muted, marginBottom: 6, ...style }}>{children}</div>
);

const field = { background: C.bg, border: `1px solid ${C.border}`, color: C.text, padding: "11px 14px", borderRadius: 8, fontSize: 14, width: "100%", transition: "border-color .2s, box-shadow .2s" };

const Sel = ({ value, onChange, children, style = {} }) => (
  <select value={value} onChange={e => onChange(e.target.value)} style={{ ...field, appearance: "none", backgroundImage: `url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='6' viewBox='0 0 10 6'%3E%3Cpath fill='%23657079' d='M5 6L0 0h10z'/%3E%3C/svg%3E")`, backgroundRepeat: "no-repeat", backgroundPosition: "right 12px center", paddingRight: 32, ...style }}>
    {children}
  </select>
);

const Inp = ({ value, onChange, placeholder, style = {}, type = "text", onKeyDown }) => (
  <input type={type} value={value} onChange={e => onChange(e.target.value)} placeholder={placeholder} onKeyDown={onKeyDown} style={{ ...field, ...style }} />
);

const Btn = ({ onClick, children, variant = "primary", style = {}, disabled = false, full = false }) => {
  const v = {
    primary: { background: C.accent, color: "#fff", fontWeight: 600 },
    outline: { background: "transparent", color: C.light, border: `1px solid ${C.border}` },
    ghost:   { background: "transparent", color: C.muted },
    danger:  { background: "rgba(192,57,43,.15)", color: "#e07060", border: `1px solid rgba(192,57,43,.25)` },
  };
  return (
    <button className="btn" onClick={onClick} disabled={disabled}
      style={{ padding: "11px 18px", borderRadius: 8, fontSize: 13, fontWeight: 600, opacity: disabled ? .45 : 1, cursor: disabled ? "not-allowed" : "pointer", width: full ? "100%" : "auto", letterSpacing: "0.2px", ...v[variant], ...style }}>
      {children}
    </button>
  );
};

const DotsLoader = ({ msg = "Just a sec…" }) => (
  <div style={{ display: "flex", gap: 6, alignItems: "center", padding: "14px 2px" }}>
    {[0, .18, .36].map((d, i) => (
      <div key={i} style={{ width: 7, height: 7, borderRadius: "50%", background: C.accent, animation: `pulse 1.1s ${d}s ease-in-out infinite` }} />
    ))}
    <span style={{ color: C.muted, fontSize: 13, marginLeft: 6, fontStyle: "italic" }}>{msg}</span>
  </div>
);

// ── ONBOARDING ─────────────────────────────────────────────────────────────
const OnboardingScreen = ({ onDone }) => {
  const [step, setStep] = useState(0);
  const [data, setData] = useState({ name: "", gender: "male", age: "", weight: "", height: "", goal: "Build muscle", level: "Beginner", equipment: "Full gym" });
  const set = (k, v) => setData(d => ({ ...d, [k]: v }));

  const steps = [
    {
      title: "Hey, let's get\nto know you.",
      sub: "Takes 60 seconds. We'll use this to personalize everything.",
      fields: (
        <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
          <div><Label>What should we call you?</Label><Inp value={data.name} onChange={v => set("name", v)} placeholder="Your first name" /></div>
          <div>
            <Label>Gender</Label>
            <div style={{ display: "flex", gap: 8 }}>
              {["male", "female", "other"].map(g => (
                <button key={g} className="btn" onClick={() => set("gender", g)}
                  style={{ flex: 1, padding: "10px 0", borderRadius: 8, border: `1px solid ${data.gender === g ? C.accent : C.border}`, background: data.gender === g ? `rgba(90,158,82,.15)` : C.bg, color: data.gender === g ? C.accent : C.light, fontSize: 13, fontWeight: 600, textTransform: "capitalize" }}>
                  {g === "male" ? "👨 Male" : g === "female" ? "👩 Female" : "🧑 Other"}
                </button>
              ))}
            </div>
          </div>
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 10 }}>
            <div><Label>Age</Label><Inp type="number" value={data.age} onChange={v => set("age", v)} placeholder="25" /></div>
            <div><Label>Weight (lbs)</Label><Inp type="number" value={data.weight} onChange={v => set("weight", v)} placeholder="175" /></div>
          </div>
          <div><Label>Height (e.g. 5'10")</Label><Inp value={data.height} onChange={v => set("height", v)} placeholder="5'10&quot;" /></div>
        </div>
      ),
      valid: data.name.trim() && data.age && data.weight,
    },
    {
      title: `Nice to meet\nyou, ${data.name || "friend"}.`,
      sub: "Now tell us what you're working toward.",
      fields: (
        <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
          <div>
            <Label>Main goal</Label>
            {["Build muscle", "Lose fat", "Improve endurance", "Increase strength", "General fitness"].map(g => (
              <button key={g} className="btn" onClick={() => set("goal", g)}
                style={{ width: "100%", marginBottom: 8, padding: "12px 16px", borderRadius: 8, textAlign: "left", border: `1px solid ${data.goal === g ? C.accent : C.border}`, background: data.goal === g ? `rgba(90,158,82,.12)` : C.bg, color: data.goal === g ? C.accent : C.light, fontSize: 13, fontWeight: 500 }}>
                {data.goal === g ? "✓ " : ""}{g}
              </button>
            ))}
          </div>
          <div>
            <Label>Experience level</Label>
            <div style={{ display: "flex", gap: 8 }}>
              {["Beginner", "Intermediate", "Advanced"].map(l => (
                <button key={l} className="btn" onClick={() => set("level", l)}
                  style={{ flex: 1, padding: "10px 0", borderRadius: 8, border: `1px solid ${data.level === l ? C.accent : C.border}`, background: data.level === l ? `rgba(90,158,82,.15)` : C.bg, color: data.level === l ? C.accent : C.light, fontSize: 12, fontWeight: 600 }}>
                  {l}
                </button>
              ))}
            </div>
          </div>
          <div>
            <Label>Equipment access</Label>
            <Sel value={data.equipment} onChange={v => set("equipment", v)}>
              <option>Full gym</option><option>Dumbbells only</option><option>Bodyweight only</option><option>Home gym</option><option>Resistance bands</option>
            </Sel>
          </div>
        </div>
      ),
      valid: true,
    },
  ];

  const current = steps[step];

  const finish = () => {
    S.set("ff_profile", data);
    onDone(data);
  };

  return (
    <div style={{ minHeight: "100vh", background: C.bg, display: "flex", flexDirection: "column" }}>
      {/* Progress bar */}
      <div style={{ height: 3, background: C.border }}>
        <div style={{ height: "100%", background: C.accent, width: `${((step + 1) / steps.length) * 100}%`, transition: "width .4s ease" }} />
      </div>

      <div style={{ flex: 1, padding: "36px 24px 24px", maxWidth: 480, margin: "0 auto", width: "100%", display: "flex", flexDirection: "column", gap: 28 }}>
        <div>
          <div style={{ fontSize: 11, color: C.muted, letterSpacing: "2px", marginBottom: 12 }}>STEP {step + 1} OF {steps.length}</div>
          <h1 style={{ ...syne, fontSize: 32, fontWeight: 800, lineHeight: 1.15, whiteSpace: "pre-line", color: C.text, marginBottom: 8 }}>{current.title}</h1>
          <p style={{ fontSize: 14, color: C.textDim, lineHeight: 1.6 }}>{current.sub}</p>
        </div>

        <div style={{ flex: 1 }}>{current.fields}</div>

        <div style={{ display: "flex", gap: 10 }}>
          {step > 0 && <Btn onClick={() => setStep(s => s - 1)} variant="outline" style={{ flex: 1 }}>Back</Btn>}
          {step < steps.length - 1
            ? <Btn onClick={() => setStep(s => s + 1)} disabled={!current.valid} style={{ flex: 1 }}>Continue →</Btn>
            : <Btn onClick={finish} style={{ flex: 1 }}>Let's go 💪</Btn>
          }
        </div>
      </div>
    </div>
  );
};

// ── MACRO RING ─────────────────────────────────────────────────────────────
const Ring = ({ value, max, label, color, unit = "g" }) => {
  const pct = Math.min(value / (max || 1), 1);
  const r = 26; const c = 2 * Math.PI * r;
  return (
    <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 5 }}>
      <svg width={64} height={64} style={{ transform: "rotate(-90deg)" }}>
        <circle cx={32} cy={32} r={r} fill="none" stroke={C.border} strokeWidth={5} />
        <circle cx={32} cy={32} r={r} fill="none" stroke={color} strokeWidth={5}
          strokeDasharray={c} strokeDashoffset={c * (1 - pct)} strokeLinecap="round" style={{ transition: "stroke-dashoffset .5s ease" }} />
      </svg>
      <div style={{ textAlign: "center" }}>
        <div style={{ fontSize: 13, fontWeight: 700 }}>{value}<span style={{ fontSize: 10, color: C.muted }}>{unit}</span></div>
        <div style={{ fontSize: 10, color: C.muted }}>{label}</div>
      </div>
    </div>
  );
};

// ── BARCODE SCANNER ────────────────────────────────────────────────────────
const Scanner = ({ onResult, onClose }) => {
  const vRef = useRef(null);
  const [status, setStatus] = useState("starting");
  const [manual, setManual] = useState("");
  const streamRef = useRef(null);
  const stop = () => { if (streamRef.current) { streamRef.current.getTracks().forEach(t => t.stop()); streamRef.current = null; } };

  useEffect(() => {
    let iv;
    (async () => {
      try {
        const stream = await navigator.mediaDevices.getUserMedia({ video: { facingMode: "environment" } });
        streamRef.current = stream;
        if (vRef.current) { vRef.current.srcObject = stream; vRef.current.play(); }
        setStatus("BarcodeDetector" in window ? "scanning" : "manual");
        if ("BarcodeDetector" in window) {
          const det = new window.BarcodeDetector({ formats: ["ean_13", "ean_8", "upc_a", "upc_e", "code_128"] });
          iv = setInterval(async () => {
            if (vRef.current?.readyState === 4) {
              try { const c = await det.detect(vRef.current); if (c.length) { clearInterval(iv); stop(); onResult(c[0].rawValue); } } catch {}
            }
          }, 300);
        }
      } catch { setStatus("denied"); }
    })();
    return () => { clearInterval(iv); stop(); };
  }, []);

  return (
    <div style={{ position: "fixed", inset: 0, background: "rgba(20,22,23,.96)", zIndex: 999, display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", padding: 20 }}>
      <div style={{ width: "100%", maxWidth: 340, background: C.surface, borderRadius: 14, overflow: "hidden", border: `1px solid ${C.border}` }}>
        <div style={{ padding: "14px 18px", borderBottom: `1px solid ${C.border}`, display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <span style={{ ...syne, fontWeight: 700, fontSize: 15 }}>Scan Barcode</span>
          <button className="btn" onClick={() => { stop(); onClose(); }} style={{ background: "none", color: C.muted, fontSize: 20, lineHeight: 1 }}>×</button>
        </div>
        {status === "starting" && <div style={{ padding: 40, textAlign: "center", color: C.muted, fontSize: 13 }}>Starting camera…</div>}
        {status === "denied" && <div style={{ padding: 20, color: "#e07060", fontSize: 13 }}>Camera access denied — enter barcode below.</div>}
        {status === "scanning" && (
          <div style={{ position: "relative", background: "#000", aspectRatio: "1" }}>
            <video ref={vRef} style={{ width: "100%", height: "100%", objectFit: "cover" }} playsInline muted />
            <div style={{ position: "absolute", inset: 0, display: "flex", alignItems: "center", justifyContent: "center" }}>
              <div style={{ width: "68%", height: "28%", border: `2px solid ${C.accent}`, borderRadius: 8, position: "relative", overflow: "hidden" }}>
                <div style={{ position: "absolute", left: 0, right: 0, height: 2, background: C.accent, opacity: .7, animation: "scanline 2s linear infinite" }} />
              </div>
            </div>
          </div>
        )}
        {status === "manual" && <div style={{ padding: "12px 18px 0", color: C.muted, fontSize: 12 }}>BarcodeDetector not available — enter barcode manually:</div>}
        <div style={{ padding: 14, display: "flex", gap: 8 }}>
          <Inp value={manual} onChange={setManual} placeholder="Enter barcode…" onKeyDown={e => e.key === "Enter" && manual && (stop(), onResult(manual))} />
          <Btn onClick={() => manual && (stop(), onResult(manual))}>Search</Btn>
        </div>
      </div>
    </div>
  );
};

// ── FOOD LOOKUP ────────────────────────────────────────────────────────────
const lookupFood = async (q) => {
  const isBarcode = /^\d{8,13}$/.test(q.trim());
  const url = isBarcode
    ? `https://world.openfoodfacts.org/api/v0/product/${q.trim()}.json`
    : `https://world.openfoodfacts.org/cgi/search.pl?search_terms=${encodeURIComponent(q)}&search_simple=1&action=process&json=1&page_size=1`;
  const r = await fetch(url); const d = await r.json();
  const p = isBarcode ? d.product : d.products?.[0];
  if (!p) return null;
  const n = p.nutriments || {};
  return {
    name: p.product_name || p.product_name_en || "Unknown",
    brand: p.brands || "",
    calories: Math.round(n["energy-kcal_100g"] || n["energy-kcal"] || 0),
    protein: Math.round(n.proteins_100g || 0),
    carbs: Math.round(n.carbohydrates_100g || 0),
    fat: Math.round(n.fat_100g || 0),
  };
};

// ═══════════════════════════════════════════════════════════════════════════
// TAB: WORKOUT
// ═══════════════════════════════════════════════════════════════════════════
const WorkoutTab = ({ profile }) => {
  const [extra, setExtra] = useState({ days: "4", notes: "" });
  const [plan, setPlan] = useState(S.get("ff_last_plan", ""));
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const name = profile?.name || "there";
  const firstName = name.split(" ")[0];

  const generate = async () => {
    setLoading(true); setError(""); setPlan("");
    const isBeg = profile?.level === "Beginner";
    const isInt = profile?.level === "Intermediate";
    const explanationStyle = isBeg
      ? `For each exercise, include a "Why this works:" line in simple, friendly language — like texting a friend. Use everyday analogies. No jargon.`
      : isInt
      ? `Add a short one-liner "Why:" for major compound movements only.`
      : `Skip explanations. Experienced lifter — just the plan.`;

    const prompt = `You're a coach texting a workout plan to ${firstName}.

Their stats: ${profile?.age}yo ${profile?.gender}, ${profile?.weight}lbs, ${profile?.height || "height not given"}.
Goal: ${profile?.goal} | Level: ${profile?.level} | Equipment: ${profile?.equipment}
Days/week: ${extra.days}
${extra.notes ? `Notes from them: ${extra.notes}` : ""}

${explanationStyle}

Write in a warm, direct coach voice. Not robotic. Use ${firstName}'s name once at the start. Format clearly with days, exercises, sets/reps/rest. End with one motivating sentence.`;

    try {
      const text = await claude([{ role: "user", content: prompt }]);
      setPlan(text); S.set("ff_last_plan", text);
    } catch { setError("Something went wrong — try again."); }
    finally { setLoading(false); }
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
      <Card>
        <div style={{ ...syne, fontSize: 20, fontWeight: 800, marginBottom: 4 }}>
          Hey {firstName} 👋
        </div>
        <div style={{ fontSize: 13, color: C.textDim, marginBottom: 16, lineHeight: 1.5 }}>
          {profile?.goal} · {profile?.level} · {profile?.equipment}
        </div>
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 10, marginBottom: 12 }}>
          <div>
            <Label>Days per week</Label>
            <Sel value={extra.days} onChange={v => setExtra(e => ({ ...e, days: v }))}>
              {["3","4","5","6"].map(d => <option key={d}>{d}</option>)}
            </Sel>
          </div>
          <div style={{ display: "flex", alignItems: "flex-end" }}>
            <div style={{ fontSize: 12, color: C.muted, lineHeight: 1.5 }}>
              Based on your profile.<br />
              <span style={{ color: C.greenXlt, cursor: "pointer" }}>Edit in settings ↗</span>
            </div>
          </div>
        </div>
        <div style={{ marginBottom: 14 }}>
          <Label>Anything to know? (optional)</Label>
          <textarea value={extra.notes} onChange={e => setExtra(x => ({ ...x, notes: e.target.value }))} placeholder="Bad knee, prefer mornings, skipping chest this week…"
            style={{ ...field, resize: "vertical", minHeight: 68, lineHeight: 1.6 }} />
        </div>
        {profile?.level === "Beginner" && (
          <div style={{ background: `rgba(90,158,82,.1)`, border: `1px solid rgba(90,158,82,.25)`, borderRadius: 8, padding: "10px 14px", marginBottom: 14, fontSize: 12, color: C.greenXlt, lineHeight: 1.6 }}>
            Since you're new to this — your plan will explain <em>why</em> each exercise works in plain English. No gym-bro speak, promise.
          </div>
        )}
        <Btn onClick={generate} disabled={loading} full>
          {loading ? "Building your plan…" : "Generate My Plan ⚡"}
        </Btn>
      </Card>

      {loading && <DotsLoader msg={`Crafting ${firstName}'s plan…`} />}
      {error && <div style={{ color: "#e07060", fontSize: 13, padding: "10px 14px", background: "rgba(192,57,43,.1)", borderRadius: 8 }}>{error}</div>}
      {plan && (
        <Card className="fadeUp" style={{ borderLeft: `3px solid ${C.accent}` }}>
          <div style={{ fontSize: 11, letterSpacing: "2px", textTransform: "uppercase", color: C.accent, marginBottom: 14, fontWeight: 600 }}>Your Plan</div>
          <pre style={{ fontSize: 13, lineHeight: 1.9, color: C.textDim, whiteSpace: "pre-wrap", fontFamily: "'DM Sans', sans-serif" }}>{plan}</pre>
        </Card>
      )}
    </div>
  );
};

// ═══════════════════════════════════════════════════════════════════════════
// TAB: NUTRITION
// ═══════════════════════════════════════════════════════════════════════════
const NutritionTab = ({ profile }) => {
  const GOALS_K = "ff_nut_goals"; const LOG_K = "ff_nut_log";
  const calcGoals = (p) => {
    if (!p?.weight) return { calories: 2000, protein: 150, carbs: 200, fat: 65 };
    const w = parseFloat(p.weight);
    const calories = p.goal === "Lose fat" ? Math.round(w * 12) : p.goal === "Build muscle" ? Math.round(w * 16) : Math.round(w * 14);
    return { calories, protein: Math.round(w * 0.85), carbs: Math.round(calories * 0.40 / 4), fat: Math.round(calories * 0.28 / 9) };
  };
  const [goals, setGoals] = useState(() => S.get(GOALS_K, calcGoals(profile)));
  const [log, setLog] = useState(() => S.get(LOG_K, {}));
  const [showScanner, setShowScanner] = useState(false);
  const [searching, setSearching] = useState(false);
  const [q, setQ] = useState(""); const [food, setFood] = useState(null);
  const [foodErr, setFoodErr] = useState(""); const [qty, setQty] = useState("100");
  const [editGoals, setEditGoals] = useState(false); const [tmp, setTmp] = useState(goals);

  const todayLog = log[today()] || [];
  const totals = todayLog.reduce((a, f) => ({ calories: a.calories + f.calories, protein: a.protein + f.protein, carbs: a.carbs + f.carbs, fat: a.fat + f.fat }), { calories: 0, protein: 0, carbs: 0, fat: 0 });

  const doSearch = async (query) => {
    setSearching(true); setFood(null); setFoodErr("");
    try { const f = await lookupFood(query); f ? setFood(f) : setFoodErr("Nothing found. Try a different name or barcode."); }
    catch { setFoodErr("Lookup failed — check your connection."); }
    setSearching(false);
  };

  const addFood = () => {
    if (!food) return;
    const fx = parseFloat(qty) / 100;
    const entry = { id: Date.now(), name: food.name + (food.brand ? ` · ${food.brand}` : ""), calories: Math.round(food.calories * fx), protein: Math.round(food.protein * fx), carbs: Math.round(food.carbs * fx), fat: Math.round(food.fat * fx), qty: qty + "g", time: new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }) };
    const up = { ...log, [today()]: [...todayLog, entry] };
    setLog(up); S.set(LOG_K, up); setFood(null); setQ(""); setQty("100");
  };

  const rem = (id) => { const up = { ...log, [today()]: todayLog.filter(f => f.id !== id) }; setLog(up); S.set(LOG_K, up); };
  const saveGoals = () => { setGoals(tmp); S.set(GOALS_K, tmp); setEditGoals(false); };
  const calPct = Math.min(totals.calories / (goals.calories || 1), 1);
  const calOver = totals.calories > goals.calories;

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
      {showScanner && <Scanner onResult={async code => { setShowScanner(false); setQ(code); await doSearch(code); }} onClose={() => setShowScanner(false)} />}

      <Card>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", marginBottom: 16 }}>
          <div>
            <div style={{ ...syne, fontSize: 17, fontWeight: 700 }}>Today's Nutrition</div>
            <div style={{ fontSize: 12, color: C.muted, marginTop: 2 }}>{new Date().toLocaleDateString("en-US", { weekday: "long", month: "short", day: "numeric" })}</div>
          </div>
          <button className="btn" onClick={() => { setEditGoals(!editGoals); setTmp(goals); }}
            style={{ background: "none", color: C.muted, fontSize: 11, letterSpacing: "1px", border: `1px solid ${C.border}`, padding: "5px 10px", borderRadius: 6, fontWeight: 600 }}>
            {editGoals ? "Cancel" : "Edit Goals"}
          </button>
        </div>

        {editGoals ? (
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 10, marginBottom: 12 }}>
            {["calories", "protein", "carbs", "fat"].map(k => (
              <div key={k}><Label>{k} {k === "calories" ? "(kcal)" : "(g)"}</Label><Inp type="number" value={tmp[k]} onChange={v => setTmp(g => ({ ...g, [k]: parseInt(v) || 0 }))} /></div>
            ))}
            <div style={{ gridColumn: "span 2" }}><Btn onClick={saveGoals} full>Save</Btn></div>
          </div>
        ) : (
          <>
            <div style={{ display: "flex", justifyContent: "space-around", marginBottom: 16 }}>
              <Ring value={totals.protein} max={goals.protein} label="Protein" color={C.greenXlt} />
              <Ring value={totals.carbs} max={goals.carbs} label="Carbs" color={C.sand} />
              <Ring value={totals.fat} max={goals.fat} label="Fat" color={C.light} />
            </div>
            <div style={{ background: C.bg, borderRadius: 8, padding: "12px 14px" }}>
              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 8 }}>
                <span style={{ fontSize: 13, color: C.muted }}>Calories</span>
                <span style={{ fontSize: 16, fontWeight: 700 }}>{totals.calories} <span style={{ fontSize: 12, color: C.muted }}>/ {goals.calories}</span></span>
              </div>
              <div style={{ background: C.border, borderRadius: 4, height: 5 }}>
                <div style={{ width: `${calPct * 100}%`, height: "100%", background: calOver ? "#c0392b" : C.accent, borderRadius: 4, transition: "width .5s ease" }} />
              </div>
              <div style={{ fontSize: 11, color: calOver ? "#e07060" : C.muted, marginTop: 6, textAlign: "right" }}>
                {calOver ? `${totals.calories - goals.calories} over goal` : `${goals.calories - totals.calories} left today`}
              </div>
            </div>
          </>
        )}
      </Card>

      {/* Search */}
      <Card>
        <div style={{ ...syne, fontSize: 15, fontWeight: 700, marginBottom: 12 }}>Add Food</div>
        <div style={{ display: "flex", gap: 8, marginBottom: 8 }}>
          <Inp value={q} onChange={setQ} placeholder="Search any food…" style={{ flex: 1 }} onKeyDown={e => e.key === "Enter" && q.trim() && doSearch(q)} />
          <Btn onClick={() => q.trim() && doSearch(q)} disabled={searching || !q.trim()}>{searching ? "…" : "Search"}</Btn>
          <button className="btn" onClick={() => setShowScanner(true)} title="Scan barcode"
            style={{ background: C.bg, border: `1px solid ${C.border}`, color: C.accent, padding: "10px 13px", borderRadius: 8, fontSize: 18 }}>📷</button>
        </div>
        {foodErr && <div style={{ color: "#e07060", fontSize: 12, marginBottom: 8 }}>{foodErr}</div>}
        {food && (
          <div className="fadeUp" style={{ background: C.bg, border: `1px solid ${C.border}`, borderRadius: 8, padding: 14 }}>
            <div style={{ fontWeight: 700, fontSize: 14 }}>{food.name}</div>
            {food.brand && <div style={{ fontSize: 11, color: C.muted, marginBottom: 10 }}>{food.brand}</div>}
            <div style={{ display: "flex", flexWrap: "wrap", gap: 10, fontSize: 12, color: C.muted, marginBottom: 12 }}>
              <span>🔥 {food.calories} kcal</span><span>💪 {food.protein}g</span><span>🌾 {food.carbs}g</span><span>🫙 {food.fat}g</span>
              <span style={{ color: C.border }}>per 100g</span>
            </div>
            <div style={{ display: "flex", gap: 8, alignItems: "flex-end" }}>
              <div style={{ flex: 1 }}><Label>Qty (g)</Label><Inp type="number" value={qty} onChange={setQty} /></div>
              <Btn onClick={addFood}>Add</Btn>
            </div>
          </div>
        )}
      </Card>

      {todayLog.length > 0 && (
        <Card>
          <div style={{ ...syne, fontSize: 15, fontWeight: 700, marginBottom: 12 }}>Today's Log</div>
          <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
            {todayLog.map(f => (
              <div key={f.id} style={{ display: "flex", justifyContent: "space-between", alignItems: "center", padding: "10px 12px", background: C.bg, borderRadius: 8 }}>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ fontSize: 13, fontWeight: 600, whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>{f.name}</div>
                  <div style={{ fontSize: 11, color: C.muted }}>{f.time} · {f.qty} · P:{f.protein}g C:{f.carbs}g F:{f.fat}g</div>
                </div>
                <div style={{ display: "flex", alignItems: "center", gap: 10, flexShrink: 0 }}>
                  <span style={{ fontSize: 14, fontWeight: 700 }}>{f.calories}</span>
                  <button className="btn" onClick={() => rem(f.id)} style={{ background: "none", color: C.muted, fontSize: 16 }}>×</button>
                </div>
              </div>
            ))}
          </div>
        </Card>
      )}
    </div>
  );
};

// ═══════════════════════════════════════════════════════════════════════════
// TAB: PROGRESS
// ═══════════════════════════════════════════════════════════════════════════
const ProgressTab = ({ profile }) => {
  const K = "ff_progress";
  const [entries, setEntries] = useState(() => S.get(K, []));
  const [form, setForm] = useState({ weight: "", bodyFat: "", notes: "", date: today() });
  const [showForm, setShowForm] = useState(false);
  const set = (k, v) => setForm(f => ({ ...f, [k]: v }));
  const firstName = profile?.name?.split(" ")[0] || "you";

  const save = () => {
    if (!form.weight) return;
    const entry = { ...form, id: Date.now(), weight: parseFloat(form.weight), bodyFat: parseFloat(form.bodyFat) || null };
    const up = [...entries, entry].sort((a, b) => a.date.localeCompare(b.date));
    setEntries(up); S.set(K, up);
    setForm({ weight: "", bodyFat: "", notes: "", date: today() }); setShowForm(false);
  };

  const rem = (id) => { const up = entries.filter(e => e.id !== id); setEntries(up); S.set(K, up); };
  const weights = entries.map(e => e.weight);
  const latest = entries[entries.length - 1];
  const first = entries[0];
  const diff = latest && first && entries.length > 1 ? (latest.weight - first.weight).toFixed(1) : null;
  const startW = parseFloat(profile?.weight);
  const currentW = latest?.weight;
  const fromStart = startW && currentW ? (currentW - startW).toFixed(1) : null;

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
      {entries.length > 0 && (
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 10 }}>
          <Card style={{ padding: "14px 12px" }}>
            <div style={{ fontSize: 22, fontWeight: 800, ...syne, color: C.greenXlt }}>{latest?.weight} <span style={{ fontSize: 13, color: C.muted, fontFamily: "'DM Sans',sans-serif", fontWeight: 400 }}>lbs</span></div>
            <div style={{ fontSize: 11, color: C.muted, marginTop: 2 }}>Current weight</div>
          </Card>
          <Card style={{ padding: "14px 12px" }}>
            <div style={{ fontSize: 22, fontWeight: 800, ...syne, color: diff < 0 ? C.accent : C.sand }}>
              {diff ? `${parseFloat(diff) > 0 ? "+" : ""}${diff}` : "—"} <span style={{ fontSize: 13, color: C.muted, fontFamily: "'DM Sans',sans-serif", fontWeight: 400 }}>lbs</span>
            </div>
            <div style={{ fontSize: 11, color: C.muted, marginTop: 2 }}>Since first log</div>
          </Card>
        </div>
      )}

      {fromStart && (
        <Card style={{ background: `rgba(90,158,82,.1)`, border: `1px solid rgba(90,158,82,.2)` }}>
          <div style={{ fontSize: 13, color: C.greenXlt, lineHeight: 1.6 }}>
            {parseFloat(fromStart) < 0
              ? `${firstName}, you're down ${Math.abs(fromStart)} lbs from when you started. Keep going 💪`
              : `${firstName}, you're up ${fromStart} lbs from your starting weight — ${profile?.goal === "Build muscle" ? "that's the goal!" : "keep tracking."}`}
          </div>
        </Card>
      )}

      {/* Bar chart */}
      {weights.length > 1 && (
        <Card>
          <div style={{ ...syne, fontSize: 15, fontWeight: 700, marginBottom: 14 }}>Weight Trend</div>
          <div style={{ display: "flex", alignItems: "flex-end", gap: 4, height: 80 }}>
            {weights.map((w, i) => {
              const mn = Math.min(...weights); const mx = Math.max(...weights);
              const h = mx === mn ? 40 : ((w - mn) / (mx - mn)) * 60 + 14;
              return <div key={i} title={`${entries[i].date}: ${w} lbs`} style={{ flex: 1, height: h, background: i === weights.length - 1 ? C.accent : C.border, borderRadius: "3px 3px 0 0", transition: "height .4s ease" }} />;
            })}
          </div>
          <div style={{ display: "flex", justifyContent: "space-between", fontSize: 10, color: C.muted, marginTop: 6 }}>
            <span>{entries[0]?.date}</span><span>{entries[entries.length - 1]?.date}</span>
          </div>
        </Card>
      )}

      <Btn onClick={() => setShowForm(!showForm)} variant={showForm ? "outline" : "primary"} full>
        {showForm ? "Cancel" : "+ Log Today"}
      </Btn>

      {showForm && (
        <Card className="fadeUp">
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 10, marginBottom: 12 }}>
            <div><Label>Weight (lbs)</Label><Inp type="number" value={form.weight} onChange={v => set("weight", v)} placeholder={startW || "175"} /></div>
            <div><Label>Body Fat % (optional)</Label><Inp type="number" value={form.bodyFat} onChange={v => set("bodyFat", v)} placeholder="18" /></div>
            <div style={{ gridColumn: "span 2" }}><Label>Date</Label><Inp type="date" value={form.date} onChange={v => set("date", v)} /></div>
            <div style={{ gridColumn: "span 2" }}>
              <Label>Notes</Label>
              <textarea value={form.notes} onChange={e => set("notes", e.target.value)} placeholder="How are you feeling? Any PRs today?" style={{ ...field, resize: "vertical", minHeight: 60 }} />
            </div>
          </div>
          <Btn onClick={save} full>Save Entry</Btn>
        </Card>
      )}

      {entries.length > 0 && (
        <Card>
          <div style={{ ...syne, fontSize: 15, fontWeight: 700, marginBottom: 12 }}>History</div>
          <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
            {[...entries].reverse().map(e => (
              <div key={e.id} style={{ display: "flex", justifyContent: "space-between", alignItems: "center", padding: "10px 12px", background: C.bg, borderRadius: 8 }}>
                <div>
                  <div style={{ fontSize: 13, fontWeight: 600 }}>{e.weight} lbs {e.bodyFat ? `· ${e.bodyFat}% BF` : ""}</div>
                  <div style={{ fontSize: 11, color: C.muted }}>{e.date}{e.notes ? ` · ${e.notes}` : ""}</div>
                </div>
                <button className="btn" onClick={() => rem(e.id)} style={{ background: "none", color: C.muted, fontSize: 16 }}>×</button>
              </div>
            ))}
          </div>
        </Card>
      )}

      {entries.length === 0 && !showForm && (
        <div style={{ textAlign: "center", color: C.muted, padding: "40px 0", fontSize: 13, lineHeight: 1.6 }}>
          No entries yet.<br />Log your first weigh-in and start seeing progress.
        </div>
      )}
    </div>
  );
};

// ═══════════════════════════════════════════════════════════════════════════
// TAB: HABITS
// ═══════════════════════════════════════════════════════════════════════════
const HabitsTab = ({ profile }) => {
  const HK = "ff_habits"; const LK = "ff_habit_log";
  const DEF = [
    { id: 1, name: "Morning workout", emoji: "🏋️", color: C.accent },
    { id: 2, name: "Hit protein goal", emoji: "💪", color: C.greenXlt },
    { id: 3, name: "8 hours sleep", emoji: "😴", color: C.light },
    { id: 4, name: "Drink 3L water", emoji: "💧", color: "#7ab8d9" },
    { id: 5, name: "No junk food", emoji: "🥗", color: C.sand },
  ];
  const [habits, setHabits] = useState(() => S.get(HK, DEF));
  const [log, setLog] = useState(() => S.get(LK, {}));
  const [newH, setNewH] = useState(""); const [showAdd, setShowAdd] = useState(false);
  const todayStr = today(); const done = log[todayStr] || [];

  const toggle = (id) => {
    const d = done.includes(id) ? done.filter(x => x !== id) : [...done, id];
    const up = { ...log, [todayStr]: d }; setLog(up); S.set(LK, up);
  };
  const addHabit = () => {
    if (!newH.trim()) return;
    const h = { id: Date.now(), name: newH.trim(), emoji: "✅", color: C.accent };
    const up = [...habits, h]; setHabits(up); S.set(HK, up); setNewH(""); setShowAdd(false);
  };
  const remHabit = (id) => { const up = habits.filter(h => h.id !== id); setHabits(up); S.set(HK, up); };
  const pct = habits.length ? Math.round((done.length / habits.length) * 100) : 0;
  const last7 = Array.from({ length: 7 }, (_, i) => { const d = new Date(); d.setDate(d.getDate() - (6 - i)); return d.toISOString().slice(0, 10); });
  const streak = (hid) => { let s = 0; for (let i = last7.length - 1; i >= 0; i--) { if ((log[last7[i]] || []).includes(hid)) s++; else break; } return s; };
  const name = profile?.name?.split(" ")[0] || "you";

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
      <Card style={{ display: "flex", alignItems: "center", gap: 18 }}>
        <div style={{ position: "relative", width: 68, height: 68, flexShrink: 0 }}>
          <svg width={68} height={68} style={{ transform: "rotate(-90deg)" }}>
            <circle cx={34} cy={34} r={28} fill="none" stroke={C.border} strokeWidth={5} />
            <circle cx={34} cy={34} r={28} fill="none" stroke={pct === 100 ? C.accent : C.greenXlt} strokeWidth={5}
              strokeDasharray={2 * Math.PI * 28} strokeDashoffset={2 * Math.PI * 28 * (1 - pct / 100)} strokeLinecap="round" style={{ transition: "stroke-dashoffset .6s ease" }} />
          </svg>
          <div style={{ position: "absolute", inset: 0, display: "flex", alignItems: "center", justifyContent: "center", fontSize: 15, fontWeight: 800, ...syne }}>{pct}%</div>
        </div>
        <div>
          <div style={{ ...syne, fontSize: 16, fontWeight: 700 }}>{done.length}/{habits.length} done</div>
          <div style={{ fontSize: 13, color: C.textDim, marginTop: 3, lineHeight: 1.4 }}>
            {pct === 100 ? `Perfect day, ${name}! 🔥` : pct >= 70 ? `Almost there, ${name}!` : pct >= 40 ? `Good start, keep pushing.` : `Let's get moving, ${name}.`}
          </div>
        </div>
      </Card>

      {/* 7-day grid */}
      <Card>
        <div style={{ ...syne, fontSize: 15, fontWeight: 700, marginBottom: 12 }}>This Week</div>
        <div style={{ overflowX: "auto" }}>
          <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 11 }}>
            <thead>
              <tr>
                <th style={{ textAlign: "left", color: C.muted, paddingBottom: 8, fontWeight: 500, fontSize: 11 }}>Habit</th>
                {last7.map(d => (
                  <th key={d} style={{ color: d === todayStr ? C.accent : C.muted, padding: "0 3px 8px", fontWeight: 500, textAlign: "center", minWidth: 26 }}>
                    {new Date(d + "T12:00:00").toLocaleDateString("en-US", { weekday: "narrow" })}
                  </th>
                ))}
                <th style={{ color: C.muted, padding: "0 0 8px 8px", fontWeight: 500 }}>🔥</th>
              </tr>
            </thead>
            <tbody>
              {habits.map(h => (
                <tr key={h.id}>
                  <td style={{ padding: "5px 0", fontSize: 12, color: C.textDim, whiteSpace: "nowrap", paddingRight: 8 }}>{h.emoji} {h.name}</td>
                  {last7.map(d => {
                    const dn = (log[d] || []).includes(h.id);
                    return <td key={d} style={{ textAlign: "center", padding: "5px 3px" }}><div style={{ width: 16, height: 16, borderRadius: 4, background: dn ? h.color : C.border, margin: "0 auto", transition: "background .2s" }} /></td>;
                  })}
                  <td style={{ padding: "5px 0 5px 8px", fontWeight: 700, color: streak(h.id) > 0 ? C.accent : C.muted, fontSize: 12 }}>{streak(h.id)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>

      {/* Checklist */}
      <Card>
        <div style={{ ...syne, fontSize: 15, fontWeight: 700, marginBottom: 12 }}>Today</div>
        <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
          {habits.map(h => {
            const dn = done.includes(h.id);
            return (
              <div key={h.id} onClick={() => toggle(h.id)}
                style={{ display: "flex", alignItems: "center", gap: 12, padding: "12px 14px", background: dn ? `rgba(90,158,82,.08)` : C.bg, borderRadius: 10, cursor: "pointer", border: `1px solid ${dn ? "rgba(90,158,82,.25)" : C.border}`, transition: "all .2s", userSelect: "none" }}>
                <div style={{ width: 20, height: 20, borderRadius: 6, border: `2px solid ${dn ? C.accent : C.border}`, background: dn ? C.accent : "transparent", display: "flex", alignItems: "center", justifyContent: "center", transition: "all .2s", flexShrink: 0 }}>
                  {dn && <span style={{ fontSize: 11, color: "#fff", fontWeight: 700 }}>✓</span>}
                </div>
                <span style={{ fontSize: 15 }}>{h.emoji}</span>
                <span style={{ fontSize: 13, fontWeight: 500, color: dn ? C.greenXlt : C.text, flex: 1, textDecoration: dn ? "line-through" : "none", transition: "color .2s" }}>{h.name}</span>
                <button className="btn" onClick={e => { e.stopPropagation(); remHabit(h.id); }} style={{ background: "none", color: C.muted, fontSize: 14, padding: "2px 4px" }}>×</button>
              </div>
            );
          })}
        </div>
        {showAdd ? (
          <div style={{ display: "flex", gap: 8, marginTop: 12 }}>
            <Inp value={newH} onChange={setNewH} placeholder="New habit…" style={{ flex: 1 }} onKeyDown={e => e.key === "Enter" && addHabit()} />
            <Btn onClick={addHabit}>Add</Btn>
            <Btn onClick={() => setShowAdd(false)} variant="outline">Cancel</Btn>
          </div>
        ) : (
          <button className="btn" onClick={() => setShowAdd(true)}
            style={{ width: "100%", marginTop: 12, padding: "10px", background: "none", border: `1px dashed ${C.border}`, color: C.muted, borderRadius: 8, fontSize: 13, transition: "border-color .2s, color .2s" }}>
            + Add habit
          </button>
        )}
      </Card>
    </div>
  );
};

// ═══════════════════════════════════════════════════════════════════════════
// SETTINGS TAB
// ═══════════════════════════════════════════════════════════════════════════
const SettingsTab = ({ profile, onUpdate }) => {
  const [form, setForm] = useState(profile || {});
  const [saved, setSaved] = useState(false);
  const set = (k, v) => setForm(f => ({ ...f, [k]: v }));

  const save = () => {
    S.set("ff_profile", form); onUpdate(form);
    setSaved(true); setTimeout(() => setSaved(false), 2000);
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
      <Card>
        <div style={{ ...syne, fontSize: 17, fontWeight: 700, marginBottom: 16 }}>Your Profile</div>
        <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
          <div><Label>Name</Label><Inp value={form.name || ""} onChange={v => set("name", v)} placeholder="First name" /></div>
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 10 }}>
            <div><Label>Age</Label><Inp type="number" value={form.age || ""} onChange={v => set("age", v)} placeholder="25" /></div>
            <div><Label>Weight (lbs)</Label><Inp type="number" value={form.weight || ""} onChange={v => set("weight", v)} placeholder="175" /></div>
          </div>
          <div><Label>Height</Label><Inp value={form.height || ""} onChange={v => set("height", v)} placeholder="5'10&quot;" /></div>
          <div>
            <Label>Gender</Label>
            <div style={{ display: "flex", gap: 8 }}>
              {["male", "female", "other"].map(g => (
                <button key={g} className="btn" onClick={() => set("gender", g)}
                  style={{ flex: 1, padding: "9px 0", borderRadius: 8, border: `1px solid ${form.gender === g ? C.accent : C.border}`, background: form.gender === g ? `rgba(90,158,82,.15)` : C.bg, color: form.gender === g ? C.accent : C.light, fontSize: 12, fontWeight: 600, textTransform: "capitalize" }}>
                  {g}
                </button>
              ))}
            </div>
          </div>
          <div><Label>Main Goal</Label>
            <Sel value={form.goal || "Build muscle"} onChange={v => set("goal", v)}>
              {["Build muscle","Lose fat","Improve endurance","Increase strength","General fitness"].map(g => <option key={g}>{g}</option>)}
            </Sel>
          </div>
          <div><Label>Experience Level</Label>
            <Sel value={form.level || "Beginner"} onChange={v => set("level", v)}>
              {["Beginner","Intermediate","Advanced"].map(l => <option key={l}>{l}</option>)}
            </Sel>
          </div>
          <div><Label>Equipment</Label>
            <Sel value={form.equipment || "Full gym"} onChange={v => set("equipment", v)}>
              {["Full gym","Dumbbells only","Bodyweight only","Home gym","Resistance bands"].map(e => <option key={e}>{e}</option>)}
            </Sel>
          </div>
          <Btn onClick={save} full>{saved ? "Saved ✓" : "Save Changes"}</Btn>
        </div>
      </Card>
      <Card>
        <div style={{ ...syne, fontSize: 15, fontWeight: 700, marginBottom: 8 }}>Data</div>
        <div style={{ fontSize: 13, color: C.muted, marginBottom: 14, lineHeight: 1.6 }}>All your data is stored locally on this device. Nothing leaves your browser.</div>
        <Btn onClick={() => { if (window.confirm("Clear all data and start over?")) { localStorage.clear(); window.location.reload(); } }} variant="danger" full>Reset All Data</Btn>
      </Card>
    </div>
  );
};

// ═══════════════════════════════════════════════════════════════════════════
// ROOT
// ═══════════════════════════════════════════════════════════════════════════
const TABS = [
  { id: "workout",   label: "Plan",      icon: "⚡" },
  { id: "nutrition", label: "Nutrition", icon: "🥗" },
  { id: "progress",  label: "Progress",  icon: "📈" },
  { id: "habits",    label: "Habits",    icon: "🔥" },
  { id: "settings",  label: "You",       icon: "👤" },
];

export default function App() {
  const [profile, setProfile] = useState(() => S.get("ff_profile", null));
  const [tab, setTab] = useState("workout");

  if (!profile) return (
    <>
      <style>{GS}</style>
      <OnboardingScreen onDone={p => setProfile(p)} />
    </>
  );

  return (
    <>
      <style>{GS}</style>
      <div style={{ minHeight: "100vh", background: C.bg, maxWidth: 480, margin: "0 auto" }}>
        {/* Header */}
        <div style={{ padding: "16px 20px 12px", borderBottom: `1px solid ${C.border}`, display: "flex", justifyContent: "space-between", alignItems: "center", position: "sticky", top: 0, background: C.bg, zIndex: 10 }}>
          <div>
            <div style={{ ...syne, fontSize: 17, fontWeight: 800, color: C.accent, letterSpacing: "-0.3px" }}>FORMFORGE</div>
          </div>
          <div style={{ fontSize: 12, color: C.muted }}>
            {profile?.name ? `Hey, ${profile.name.split(" ")[0]} 👋` : new Date().toLocaleDateString("en-US", { month: "short", day: "numeric" })}
          </div>
        </div>

        {/* Tabs */}
        <div style={{ display: "flex", borderBottom: `1px solid ${C.border}`, background: C.bg, position: "sticky", top: 49, zIndex: 9 }}>
          {TABS.map(t => (
            <button key={t.id} className="btn" onClick={() => setTab(t.id)}
              style={{ flex: 1, padding: "10px 2px 8px", background: "none", color: tab === t.id ? C.accent : C.muted, fontSize: 9, fontWeight: 600, letterSpacing: "0.5px", textTransform: "uppercase", borderBottom: `2px solid ${tab === t.id ? C.accent : "transparent"}`, display: "flex", flexDirection: "column", alignItems: "center", gap: 3, transition: "color .2s" }}>
              <span style={{ fontSize: 17 }}>{t.icon}</span>
              {t.label}
            </button>
          ))}
        </div>

        {/* Content */}
        <div style={{ padding: "16px 14px 48px" }}>
          {tab === "workout"   && <WorkoutTab   profile={profile} />}
          {tab === "nutrition" && <NutritionTab profile={profile} />}
          {tab === "progress"  && <ProgressTab  profile={profile} />}
          {tab === "habits"    && <HabitsTab    profile={profile} />}
          {tab === "settings"  && <SettingsTab  profile={profile} onUpdate={setProfile} />}
        </div>
      </div>
    </>
  );
}
