package core

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
)

type aiUsageSummary struct {
	Date               string `json:"date"`
	PlanTier           string `json:"planTier"`
	InputTokens        int    `json:"inputTokens"`
	OutputTokens       int    `json:"outputTokens"`
	TotalTokens        int    `json:"totalTokens"`
	CostMicros         int64  `json:"costMicros"`
	DailyTokenCap      int    `json:"dailyTokenCap"`
	DailyCostCapMicros int64  `json:"dailyCostCapMicros"`
	OnlineAllowed      bool   `json:"onlineAllowed"`
}

func estimateTokens(text string) int {
	n := len([]rune(text))/4 + 1
	if n < 1 {
		return 1
	}
	return n
}

func estimateOnlineCostMicros(model string, input, output int) int64 {
	// Conservative local accounting estimate, not a billing assertion. Admin caps are safety rails.
	m := strings.ToLower(model)
	inPerMillion, outPerMillion := int64(500000), int64(1500000)
	if strings.Contains(m, "mini") {
		inPerMillion, outPerMillion = 200000, 800000
	}
	if strings.Contains(m, "local") {
		return 0
	}
	return (int64(input)*inPerMillion + int64(output)*outPerMillion) / 1000000
}

func dailyAIUsage(db Database, user User) aiUsageSummary {
	date := todayISO()
	out := aiUsageSummary{Date: date, PlanTier: user.PlanTier}
	if out.PlanTier == "" {
		out.PlanTier = "free"
	}
	if out.PlanTier == "pro" || user.Role == "admin" {
		out.OnlineAllowed = true
		out.DailyTokenCap = db.Settings.AIDailyTokenCap
		out.DailyCostCapMicros = db.Settings.AIDailyCostCapMicros
	} else {
		out.DailyTokenCap = db.Settings.FreeDailyTokenCap
		out.DailyCostCapMicros = db.Settings.FreeDailyCostCapMicros
		out.OnlineAllowed = out.DailyTokenCap > 0 && out.DailyCostCapMicros > 0
	}
	for _, x := range db.AIUsage {
		if x.UserID == user.ID && x.Date == date {
			out.InputTokens += x.InputTokens
			out.OutputTokens += x.OutputTokens
			out.CostMicros += x.CostMicros
		}
	}
	out.TotalTokens = out.InputTokens + out.OutputTokens
	return out
}

func (s *Server) aiUsageStatus(w http.ResponseWriter, cu *contextUser) {
	var out aiUsageSummary
	_ = s.Store.Read(func(db Database) error { out = dailyAIUsage(db, cu.User); return nil })
	jsonOut(w, 200, out)
}

func onlineBudgetError(db Database, user User, estimatedInput int) error {
	u := dailyAIUsage(db, user)
	if !u.OnlineAllowed {
		return fmt.Errorf("Online AI is available on the paid tier. Offline coaching remains available without limits")
	}
	reserve := estimatedInput + 1000
	if u.DailyTokenCap > 0 && u.TotalTokens+reserve > u.DailyTokenCap {
		return fmt.Errorf("Your daily online-AI token cap has been reached. Use Offline mode or try again tomorrow")
	}
	estCost := estimateOnlineCostMicros(db.Settings.AIModel, estimatedInput, 1000)
	if u.DailyCostCapMicros > 0 && u.CostMicros+estCost > u.DailyCostCapMicros {
		return fmt.Errorf("Your daily online-AI cost cap has been reached. Use Offline mode or try again tomorrow")
	}
	return nil
}

func aiGroundingFor(db Database, userID string) []AIGrounding {
	prefs := db.CoachPreferences[userID]
	selected := map[string]bool{}
	for _, c := range prefs.Influences {
		selected[c.ProfileID] = true
	}
	out := []AIGrounding{}
	for _, src := range db.CoachSources {
		if selected[src.ProfileID] && strings.TrimSpace(src.Summary) != "" {
			label := "Grounded in " + src.Title
			out = append(out, AIGrounding{Kind: "source", Label: label, SourceID: src.ID, URL: src.SourceURL})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Label < out[j].Label })
	if len(out) > 4 {
		out = out[:4]
	}
	if len(out) == 0 {
		out = []AIGrounding{{Kind: "general_knowledge", Label: "General fitness knowledge and your FormForge records"}}
	}
	return out
}

func (s *Server) deleteAIHistoryItem(w http.ResponseWriter, r *http.Request, cu *contextUser, id string) {
	found := false
	_ = s.Store.Update(func(db *Database) error {
		n := make([]ChatMessage, 0, len(db.ChatMessages))
		for _, m := range db.ChatMessages {
			if m.ID == id && m.UserID == cu.User.ID {
				found = true
				continue
			}
			n = append(n, m)
		}
		db.ChatMessages = n
		if found {
			u := db.Users[cu.User.ID]
			s.audit(db, &u, "ai.history.delete", id, clientIP(r), nil)
		}
		return nil
	})
	if !found {
		jsonError(w, 404, "not_found", "Chat message not found.")
		return
	}
	jsonOut(w, 200, map[string]any{"ok": true})
}
