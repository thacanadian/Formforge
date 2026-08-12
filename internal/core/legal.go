package core

import (
	"fmt"
	"html"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const legalEffectiveDate = "August 4, 2026"

type LegalConfig struct {
	BusinessName      string   `json:"businessName"`
	OperatorLegalName string   `json:"operatorLegalName"`
	EntityDescription string   `json:"entityDescription"`
	SupportEmail      string   `json:"supportEmail"`
	PrivacyEmail      string   `json:"privacyEmail"`
	ModerationEmail   string   `json:"moderationEmail"`
	MailingAddress    string   `json:"mailingAddress"`
	MinimumAge        int      `json:"minimumAge"`
	MonthlyPrice      string   `json:"monthlyPrice"`
	Jurisdiction      string   `json:"jurisdiction"`
	EffectiveDate     string   `json:"effectiveDate"`
	Configured        bool     `json:"configured"`
	Warnings          []string `json:"warnings,omitempty"`
}

func LegalConfigFromEnv() LegalConfig {
	get := func(name, fallback string) string {
		if v := strings.TrimSpace(os.Getenv(name)); v != "" {
			return v
		}
		return fallback
	}
	age := 18
	if v, err := strconv.Atoi(strings.TrimSpace(os.Getenv("FORMFORGE_MINIMUM_AGE"))); err == nil && v >= 18 && v <= 120 {
		age = v
	}
	c := LegalConfig{
		BusinessName:      get("FORMFORGE_BUSINESS_NAME", "FormForge"),
		OperatorLegalName: get("FORMFORGE_LEGAL_NAME", "OPERATOR LEGAL NAME REQUIRED"),
		EntityDescription: get("FORMFORGE_ENTITY_DESCRIPTION", "an individual sole proprietor"),
		SupportEmail:      get("FORMFORGE_SUPPORT_EMAIL", "support@example.com"),
		PrivacyEmail:      get("FORMFORGE_PRIVACY_EMAIL", get("FORMFORGE_SUPPORT_EMAIL", "privacy@example.com")),
		ModerationEmail:   get("FORMFORGE_MODERATION_EMAIL", get("FORMFORGE_SUPPORT_EMAIL", "moderation@example.com")),
		MailingAddress:    get("FORMFORGE_MAILING_ADDRESS", "MAILING ADDRESS REQUIRED"),
		MinimumAge:        age,
		MonthlyPrice:      get("FORMFORGE_MONTHLY_PRICE", "$20.00 per month"),
		Jurisdiction:      get("FORMFORGE_GOVERNING_LAW", "Texas, United States"),
		EffectiveDate:     legalEffectiveDate,
	}
	if strings.Contains(c.OperatorLegalName, "REQUIRED") {
		c.Warnings = append(c.Warnings, "Set FORMFORGE_LEGAL_NAME before public launch.")
	}
	if strings.Contains(c.MailingAddress, "REQUIRED") {
		c.Warnings = append(c.Warnings, "Set FORMFORGE_MAILING_ADDRESS before public launch.")
	}
	if strings.HasSuffix(c.SupportEmail, "example.com") || strings.HasSuffix(c.PrivacyEmail, "example.com") || strings.HasSuffix(c.ModerationEmail, "example.com") {
		c.Warnings = append(c.Warnings, "Replace example.com contact emails before public launch.")
	}
	c.Configured = len(c.Warnings) == 0
	return c
}

func (s *Server) legalConfig() LegalConfig {
	if s.Legal.BusinessName == "" {
		s.Legal = LegalConfigFromEnv()
	}
	return s.Legal
}

func (s *Server) legalVersions() map[string]string {
	v := map[string]string{"terms": "2.0", "privacy": "1.0", "community": "1.0", "subscription": "1.0"}
	_ = s.Store.Read(func(db Database) error {
		if db.Settings.TermsVersion != "" {
			v["terms"] = db.Settings.TermsVersion
		}
		if db.Settings.PrivacyVersion != "" {
			v["privacy"] = db.Settings.PrivacyVersion
		}
		if db.Settings.CommunityVersion != "" {
			v["community"] = db.Settings.CommunityVersion
		}
		if db.Settings.SubscriptionVersion != "" {
			v["subscription"] = db.Settings.SubscriptionVersion
		}
		return nil
	})
	return v
}

func (s *Server) getTerms(w http.ResponseWriter) {
	c := s.legalConfig()
	v := s.legalVersions()
	jsonOut(w, 200, map[string]any{
		"version": v["terms"], "versions": v, "text": legalPlainText(c), "takedownContact": c.ModerationEmail,
		"config": c, "links": legalLinks(),
	})
}

func (s *Server) legalStatus(w http.ResponseWriter, cu *contextUser) {
	v := s.legalVersions()
	u := cu.User
	p := Profile{}
	_ = s.Store.Read(func(db Database) error { u = db.Users[u.ID]; p = db.Profiles[u.ID]; return nil })
	accepted := map[string]bool{
		"terms":     u.TermsAcceptedVersion == v["terms"],
		"privacy":   u.PrivacyAcceptedVersion == v["privacy"],
		"community": u.CommunityAcceptedVersion == v["community"],
		"age":       u.AgeConfirmedAt != "" && p.Age >= s.legalConfig().MinimumAge,
	}
	restriction := map[string]any{"restricted": communityRestricted(u), "banned": u.CommunityBanned, "suspendedUntil": u.CommunitySuspendedUntil, "reason": u.CommunityBanReason}
	jsonOut(w, 200, map[string]any{"versions": v, "accepted": accepted, "eligibleForCommunity": accepted["terms"] && accepted["privacy"] && accepted["community"] && accepted["age"] && !communityRestricted(u), "restriction": restriction, "config": s.legalConfig(), "links": legalLinks()})
}

func (s *Server) acceptTerms(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	var in struct{ Terms, Privacy, Community, AgeConfirmed bool }
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	// Backward-compatible clients used an empty POST to accept the complete legal set.
	if !in.Terms && !in.Privacy && !in.Community && !in.AgeConfirmed {
		in.Terms, in.Privacy, in.Community, in.AgeConfirmed = true, true, true, true
	}
	if !in.Terms || !in.Privacy || !in.AgeConfirmed {
		jsonError(w, 400, "consent_required", "Terms, privacy notice, and age confirmation are required.")
		return
	}
	v := s.legalVersions()
	var p Profile
	_ = s.Store.Read(func(db Database) error { p = db.Profiles[cu.User.ID]; return nil })
	if p.Age < s.legalConfig().MinimumAge {
		jsonError(w, 403, "age_restricted", fmt.Sprintf("FormForge public accounts require users to be at least %d.", s.legalConfig().MinimumAge))
		return
	}
	now := nowISO()
	_ = s.Store.Update(func(db *Database) error {
		u := db.Users[cu.User.ID]
		u.TermsAcceptedVersion, u.TermsAcceptedAt = v["terms"], now
		u.PrivacyAcceptedVersion, u.PrivacyAcceptedAt = v["privacy"], now
		u.AgeConfirmedAt = now
		if in.Community {
			u.CommunityAcceptedVersion, u.CommunityAcceptedAt = v["community"], now
		}
		u.UpdatedAt = now
		db.Users[u.ID] = u
		s.audit(db, &u, "legal.accept", strings.Join([]string{v["terms"], v["privacy"], v["community"]}, "/"), clientIP(r), map[string]any{"community": in.Community, "ageConfirmed": true})
		return nil
	})
	jsonOut(w, 200, map[string]any{"ok": true, "versions": v})
}

func legalLinks() map[string]string {
	return map[string]string{"index": "/legal", "terms": "/legal/terms", "privacy": "/legal/privacy", "community": "/legal/community", "subscription": "/legal/subscription", "healthAI": "/legal/health-ai", "security": "/legal/security", "accountDeletion": "/account-deletion"}
}

func legalPlainText(c LegalConfig) string {
	return fmt.Sprintf(`FORMFORGE LEGAL SUMMARY

Operator: %s, doing business as %s (%s).
Support: %s. Privacy: %s. Community safety: %s.

FormForge is a fitness organization and educational coaching service, not medical care, diagnosis, treatment, or emergency assistance. AI output can be wrong. Stop activity and seek qualified care for severe, sudden, or worsening symptoms.

The service collects account details, fitness profile data, workouts, nutrition, progress, optional photos and health imports, AI conversations, community activity, and security logs as described in the Privacy Notice. It does not sell personal information or use health data for targeted advertising. Online AI is optional and may send the user prompt and relevant fitness context to the configured model provider.

Community participation requires acceptance of the Community Standards. Harassment, hate, sexual exploitation, threats, dangerous encouragement, impersonation, privacy violations, and spam are prohibited. Users can report content and block users; moderators may remove content, suspend community access, or ban accounts.

Paid access is expected to be offered at %s. Store purchases are billed, renewed, cancelled, and refunded under the applicable Apple App Store or Google Play terms. Deleting a FormForge account does not itself cancel an external store subscription.

Users may delete accounts from the Security page or the public account deletion page. Active account data is deleted, subject to limited retention for security, fraud prevention, legal obligations, transaction records, and encrypted backup rotation.

These documents are practical operator templates and are not a substitute for advice from a qualified attorney or tax professional.`, c.OperatorLegalName, c.BusinessName, c.EntityDescription, c.SupportEmail, c.PrivacyEmail, c.ModerationEmail, c.MonthlyPrice)
}

func (s *Server) serveLegalPage(w http.ResponseWriter, r *http.Request) {
	slug := strings.Trim(strings.TrimPrefix(r.URL.Path, "/legal"), "/")
	if r.URL.Path == "/account-deletion" {
		slug = "account-deletion"
	}
	c := s.legalConfig()
	title, body := legalPageContent(slug, c)
	if title == "" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	warn := ""
	if !c.Configured {
		warn = `<div class="warning"><strong>Operator setup incomplete.</strong> ` + html.EscapeString(strings.Join(c.Warnings, " ")) + ` These documents must not be published as final until completed.</div>`
	}
	fmt.Fprintf(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>%s — %s</title><style>%s</style></head><body><header><a href="/">FORMFORGE</a><nav><a href="/legal/terms">Terms</a><a href="/legal/privacy">Privacy</a><a href="/legal/community">Community</a><a href="/legal/subscription">Subscription</a><a href="/account-deletion">Delete account</a></nav></header><main>%s<h1>%s</h1><p class="meta">Effective and last updated %s</p>%s</main><footer>Operated by %s, %s.<br>%s · <a href="mailto:%s">%s</a></footer></body></html>`, html.EscapeString(title), html.EscapeString(c.BusinessName), legalCSS, warn, html.EscapeString(title), html.EscapeString(c.EffectiveDate), body, html.EscapeString(c.OperatorLegalName), html.EscapeString(c.EntityDescription), html.EscapeString(c.MailingAddress), html.EscapeString(c.SupportEmail), html.EscapeString(c.SupportEmail))
}

const legalCSS = `:root{color-scheme:dark;--bg:#08090a;--card:#101214;--line:#402b16;--accent:#ff922b;--text:#f3eee7;--muted:#aaa39b}*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--text);font:16px/1.65 system-ui,sans-serif}header,main,footer{max-width:920px;margin:auto;padding:24px}header{display:flex;justify-content:space-between;gap:20px;align-items:center;border-bottom:1px solid var(--line)}header>a{font-weight:900;color:var(--accent);text-decoration:none;letter-spacing:.12em}nav{display:flex;gap:14px;flex-wrap:wrap}a{color:#ffad5c}main{padding-top:48px;padding-bottom:64px}h1{font-size:clamp(2rem,5vw,3.7rem);line-height:1.05;margin:0 0 8px;color:var(--accent)}h2{margin-top:36px}h3{margin-top:24px}section{background:var(--card);border:1px solid var(--line);border-radius:14px;padding:22px;margin:18px 0}.meta,.muted{color:var(--muted)}.warning{border:1px solid #ef7d32;background:#2b160d;padding:14px;border-radius:10px;margin-bottom:24px}.danger{border-left:4px solid #d84b4b}.deletion-form{display:grid;gap:12px;max-width:540px}input,button{font:inherit;padding:12px;border-radius:8px;border:1px solid var(--line);background:#08090a;color:var(--text)}button{background:var(--accent);color:#160b02;font-weight:800;cursor:pointer}footer{border-top:1px solid var(--line);color:var(--muted);font-size:.9rem}`

func legalPageContent(slug string, c LegalConfig) (string, string) {
	e := html.EscapeString
	contact := fmt.Sprintf(`<a href="mailto:%s">%s</a>`, e(c.SupportEmail), e(c.SupportEmail))
	switch slug {
	case "", "index":
		return "Legal & Safety Center", `<section><h2>Policies</h2><p>These documents govern use of FormForge and explain how fitness, health, AI, billing, and community data are handled.</p><ul><li><a href="/legal/terms">Terms of Use</a></li><li><a href="/legal/privacy">Privacy Notice</a></li><li><a href="/legal/community">Community Standards</a></li><li><a href="/legal/subscription">Subscription Terms</a></li><li><a href="/legal/health-ai">Health and AI Notice</a></li><li><a href="/legal/security">Security and Breach Notice</a></li><li><a href="/account-deletion">Account and Data Deletion</a></li></ul></section>`
	case "terms":
		return "Terms of Use", fmt.Sprintf(`<section><h2>1. Agreement and eligibility</h2><p>These Terms are an agreement between you and %s, doing business as %s. You must be at least %d years old and legally able to enter this agreement. By creating an account or using the service, you agree to these Terms and the Privacy Notice.</p><h2>2. Fitness service—not medical care</h2><p>FormForge provides fitness organization, educational information, and automated coaching. It is not a medical provider and does not diagnose, treat, cure, or prevent disease or injury. Do not use it for emergencies. Stop training and seek qualified care for chest pain, fainting, severe shortness of breath, neurological symptoms, severe or worsening pain, or other urgent symptoms.</p><h2>3. Accounts</h2><p>Provide accurate information, protect your password and devices, and notify us of suspected unauthorized access. You are responsible for activity under your account. One person may not impersonate another or use another person’s health information without authorization.</p><h2>4. Acceptable use</h2><p>You may not break laws, probe or disrupt the service, evade limits, upload malware, scrape other users, misuse health information, infringe intellectual-property or privacy rights, or use AI output to provide unlicensed medical care. Community activity is also governed by the Community Standards.</p><h2>5. AI and third-party information</h2><p>AI output, food information, creator summaries, and generated plans may be incomplete or incorrect. Verify important information. Editorial creator profiles do not imply endorsement, affiliation, licensing, or that a creator authored a response. Exact quotations require a verified source.</p><h2>6. Your content</h2><p>You retain ownership of content you submit. You grant us a limited license to host, process, display, and moderate it only as necessary to operate, secure, improve, and enforce the service. Do not submit content you lack authority to use.</p><h2>7. Suspension and termination</h2><p>We may remove content or restrict accounts for safety, legal, security, payment, or policy reasons. Community restrictions do not necessarily block private fitness tools. You may delete your account through the Security page or public deletion page.</p><h2>8. Paid service</h2><p>Paid access is described in the Subscription Terms. App-store billing is controlled by the applicable store. Deleting your account does not automatically cancel an Apple or Google subscription.</p><h2>9. Disclaimers and limitation</h2><p>To the maximum extent allowed by law, the service is provided “as is” and “as available.” We do not promise uninterrupted availability or particular fitness, weight, health, or financial results. Nothing in these Terms excludes rights or liabilities that cannot legally be excluded.</p><h2>10. Governing law and contact</h2><p>These Terms are governed by the laws of %s, without overriding mandatory consumer protections. Contact %s or write to %s.</p></section>`, e(c.OperatorLegalName), e(c.BusinessName), c.MinimumAge, e(c.Jurisdiction), contact, e(c.MailingAddress))
	case "privacy":
		return "Privacy Notice", fmt.Sprintf(`<section><h2>1. Who controls your data</h2><p>%s, doing business as %s, operates FormForge. Privacy questions: <a href="mailto:%s">%s</a>.</p><h2>2. Data we collect</h2><ul><li>Account and profile data: name, email, password hash, age confirmation, preferences, and subscription status.</li><li>Fitness data: goals, measurements, workouts, exercise performance, nutrition, habits, check-ins, recovery, pain flags, and progress.</li><li>Optional sensitive data: progress photos, heart-rate or wearable imports, body-fat information, and notes you enter.</li><li>AI data: prompts, responses, source links, usage totals, and relevant profile or fitness context used to answer a request.</li><li>Community data: nudges, shared workouts, reports, blocks, and moderation records.</li><li>Security and technical data: IP address, device label, sessions, login attempts, diagnostics, audit events, and crash reports if enabled.</li><li>Billing data, if paid store access is enabled: Apple or Google may provide entitlement and transaction status; FormForge does not need to store complete payment-card numbers.</li></ul><h2>3. Why we use data</h2><p>We use data to provide and personalize the service, sync records, operate AI and community features, secure accounts, prevent abuse, fulfill purchases if paid access is enabled, respond to support and deletion requests, comply with law, and improve reliability. We do not sell personal information and do not use health or fitness data for targeted advertising.</p><h2>4. When data is shared</h2><p>Data may be processed by hosting and infrastructure providers such as Render; by an AI provider such as OpenAI only when online AI is enabled and requested; by Apple or Google if store billing is enabled; and by public-data providers such as Open Food Facts for searches. We may disclose data when legally required, to protect users or the service, or during a business transfer subject to appropriate safeguards. Community content is visible only to the users allowed by the feature.</p><h2>5. AI choices</h2><p>Offline coaching does not send prompts to an external model provider. Online AI may send the prompt and relevant fitness context to the configured AI provider. Do not put secrets or third-party medical records in prompts.</p><h2>6. Retention and deletion</h2><p>We retain active-account data while the account is used. On account deletion, active records and uploaded photo files are removed. Limited transaction, fraud-prevention, security, moderation, tax, and legal records may be retained where reasonably necessary. By default, encrypted local backup rotation keeps the 14 newest backups plus one weekly backup for up to eight weeks. Manually downloaded or separately copied backups remain under the control of the operator or person who copied them and must be securely deleted under the operator’s retention process. We do not promise that information already shared with another user can be erased from that user’s independent records or screenshots.</p><h2>7. Security</h2><p>Controls include encrypted secrets and photos, HTTPS in hosted use, password hashing, CSRF protection, session controls, rate limits, audit logs, and backups. No system is perfectly secure. See the Security and Breach Notice.</p><h2>8. Your choices and privacy requests</h2><p>You can review and update profile data, export records, disable optional online AI, block users, report content, revoke sessions, and delete your account. Depending on applicable law, you may also request access, correction, deletion, or a portable copy of personal data and may opt out of covered sale, targeted advertising, or certain profiling. FormForge does not sell personal information or use health or fitness data for targeted advertising. Submit a request through the in-app controls or contact <a href="mailto:%s">%s</a>; identity verification may be required.</p><h2>9. Texas privacy notice</h2><p>Texas residents may have rights under the Texas Data Privacy and Security Act when it applies. FormForge provides access, correction, export, and deletion tools even when a small-business exemption may apply. We do not sell sensitive data. Privacy complaints or appeals should first be sent to the privacy contact above.</p><h2>10. Children</h2><p>FormForge is not directed to children. Public accounts are limited to users age %d or older. We do not knowingly collect data from younger users.</p><h2>11. Changes</h2><p>Material changes will be posted and may require renewed consent.</p></section>`, e(c.OperatorLegalName), e(c.BusinessName), e(c.PrivacyEmail), e(c.PrivacyEmail), e(c.PrivacyEmail), e(c.PrivacyEmail), c.MinimumAge)
	case "community":
		return "Community Standards", fmt.Sprintf(`<section><h2>Be useful, honest, and safe</h2><p>Community tools exist for workout accountability—not anonymous public posting. You must accept these standards before posting.</p><h2>Not allowed</h2><ul><li>Harassment, bullying, hate, dehumanizing attacks, or discriminatory slurs.</li><li>Sexual exploitation, sexual content involving minors, non-consensual sexual content, or sexual solicitation.</li><li>Credible threats, instructions to harm another person, encouragement of self-harm, or glorification of dangerous conduct.</li><li>Impersonation, fraud, spam, scams, malware, or repeated unwanted contact.</li><li>Sharing another person’s private, identifying, health, or intimate information without permission.</li><li>Illegal content, copyright infringement, or using community features to sell regulated products.</li><li>Medical diagnosis or dangerous advice presented as professional care.</li></ul><h2>Moderation</h2><p>Automated filters may reject obvious abusive or sexual text, but no filter catches everything. Users can report content and block other users. Moderators may remove content, warn users, suspend community access, or ban accounts. Reports are reviewed in context; deliberately false or retaliatory reporting is prohibited.</p><h2>Emergency reports</h2><p>FormForge is not an emergency service. If someone appears in immediate danger, contact local emergency services rather than relying on an in-app report.</p><h2>Contact</h2><p>Report in the app or contact <a href="mailto:%s">%s</a>.</p></section>`, e(c.ModerationEmail), e(c.ModerationEmail))
	case "subscription":
		return "Subscription Terms", fmt.Sprintf(`<section><h2>FormForge Pro</h2><p>The planned standard price is %s, shown in local currency where available. The purchase screen must display the exact price, billing period, included features, and any trial before you confirm.</p><h2>Automatic renewal</h2><p>Subscriptions renew automatically unless cancelled through the Apple App Store or Google Play settings before the next renewal. Charges are processed by the store account used to subscribe. Store price, tax, currency, grace-period, refund, and family-sharing rules may apply.</p><h2>Cancellation</h2><p>Cancellation stops future renewals; access ordinarily continues through the paid period. Deleting FormForge or deleting your FormForge account does not by itself cancel an external store subscription. Use the store’s subscription-management screen.</p><h2>Trials and promotions</h2><p>Any trial length and post-trial price will be shown before purchase. Unless cancelled before the trial ends, the subscription converts to paid renewal as disclosed by the store.</p><h2>Refunds</h2><p>For Apple or Google purchases, request refunds through that store. We cannot promise refunds outside the store’s rules. Mandatory consumer rights still apply.</p><h2>Changes</h2><p>Price changes will be handled through store-required notice and consent processes. Material reductions in recurring service will be disclosed.</p><h2>Contact</h2><p>Billing support: %s.</p></section>`, e(c.MonthlyPrice), contact)
	case "health-ai":
		return "Health and AI Notice", `<section class="danger"><h2>Not medical advice</h2><p>FormForge is a fitness and education tool. It does not diagnose injuries, prescribe treatment, replace a clinician, or provide emergency assistance. AI may misunderstand context or produce incorrect information.</p><h2>Use safely</h2><ul><li>Consult a qualified professional before starting a new program when you have a medical condition, are pregnant, take medication affecting exercise, or have unexplained symptoms.</li><li>Stop exercise for severe, sudden, sharp, or worsening symptoms.</li><li>Do not use calorie, body-fat, recovery, wearable, or AI estimates as clinical measurements.</li><li>Do not delay professional care because of an app response.</li></ul><h2>AI transparency</h2><p>When you use AI Coach or FormForge Agent, you are interacting with an artificial-intelligence system. Offline answers are generated from bundled rules and reference modules. Online answers are generated by a configured third-party model and may send relevant profile and fitness context to that provider. Sources are shown when available, but citations do not guarantee that an answer is correct.</p><h2>Creator profiles</h2><p>Editorial profiles are independent summaries. They are not endorsements, exact voice replicas, official programs, or evidence that a creator reviewed an answer.</p></section>`
	case "security":
		return "Security and Breach Notice", fmt.Sprintf(`<section><h2>Security program</h2><p>FormForge uses HTTPS for hosted traffic, password hashing, secure sessions, CSRF controls, login throttling, encrypted secrets and progress photos, audit logging, and encrypted backups. The operator must keep Render, domain, app-store, email, and AI-provider accounts protected with multifactor authentication.</p><h2>Your responsibilities</h2><p>Use a unique password, protect devices, enable available 2FA, and sign out or revoke lost-device sessions. Do not share recovery keys or API keys.</p><h2>Security incident response</h2><p>If we discover unauthorized access to identifying health information or other protected data, we will investigate, contain the incident, preserve evidence, and provide notices required by applicable law. This can include direct notice to affected people, reports required by Texas breach law, and notices required by the FTC Health Breach Notification Rule when applicable. Contact <a href="mailto:%s">%s</a> with suspected vulnerabilities or account compromise. Do not publicly disclose personal data or exploit a vulnerability.</p></section>`, e(c.PrivacyEmail), e(c.PrivacyEmail))
	case "account-deletion":
		return "Account and Data Deletion", fmt.Sprintf(`<section><h2>Delete from the app</h2><p>Sign in, open <strong>Security</strong>, choose <strong>Delete my account</strong>, enter your password, and confirm. Standard-user deletion is immediate and signs out all sessions.</p><h2>Delete from the web</h2><p>This form deletes the FormForge account associated with the supplied credentials. It does not cancel an Apple or Google subscription. If you cannot authenticate, contact support for an identity-verification and deletion process.</p><form class="deletion-form" id="delete-form"><label>Email<input name="email" type="email" required autocomplete="email"></label><label>Password<input name="password" type="password" required autocomplete="current-password"></label><label>Type DELETE<input name="confirmation" required pattern="DELETE"></label><button>Delete account and active data</button></form><p id="delete-result" class="muted"></p><script src="/legal-delete.js" defer></script><h2>What is deleted</h2><p>Active profile, workout, nutrition, progress, health import, AI chat, photo, community, session, and preference records are removed. Content sent to another user is removed from the live service where reasonably possible.</p><h2>Limited retention</h2><p>We may retain limited tax, transaction, security, fraud-prevention, moderation, legal, and backup records for legitimate requirements. By default, encrypted local backups retain the 14 newest copies plus one weekly copy for up to eight weeks; separately copied or downloaded backups require manual secure deletion. Contact <a href="mailto:%s">%s</a> for assistance.</p><h2>Administrator accounts</h2><p>The sole service administrator cannot self-delete while responsible for the installation. Transfer administration or contact support for full service closure.</p></section>`, e(c.SupportEmail), e(c.SupportEmail))
	default:
		return "", ""
	}
}
