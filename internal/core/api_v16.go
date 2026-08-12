package core

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
)

func measurementSystemFor(db Database, userID string) string {
	if p, ok := db.ThemePreferences[userID]; ok && strings.EqualFold(p.MeasurementSystem, "metric") {
		return "metric"
	}
	return "imperial"
}

func formatWeightForUser(db Database, userID string, kg float64) string {
	if measurementSystemFor(db, userID) == "metric" {
		return fmt.Sprintf("%.1f kg", kg)
	}
	return fmt.Sprintf("%.1f lb", kg*2.2046226218)
}

func formatHeightForUser(db Database, userID string, cm float64) string {
	if measurementSystemFor(db, userID) == "metric" {
		return fmt.Sprintf("%.1f cm", cm)
	}
	totalInches := cm / 2.54
	feet := int(totalInches) / 12
	inches := totalInches - float64(feet*12)
	return fmt.Sprintf("%d ft %.1f in", feet, inches)
}

func wantsSourceLinks(message string) bool {
	m := strings.ToLower(message)
	return containsAny(m, "source", "sources", "citation", "citations", "link", "links", "study", "studies", "research paper", "where did", "evidence for")
}

func wantsExactQuote(message string) bool {
	m := strings.ToLower(message)
	return containsAny(m, "exact quote", "quote", "what did", "verbatim", "word for word")
}

func approvedSourceLinks(db Database, userID, query string, limit int) []CoachSource {
	selected := map[string]bool{}
	for _, sel := range preferencesFor(db, userID).Influences {
		selected[sel.ProfileID] = true
	}
	words := knowledgeWords(query)
	type scored struct {
		src   CoachSource
		score int
	}
	var candidates []scored
	for _, src := range db.CoachSources {
		if len(selected) > 0 && !selected[src.ProfileID] {
			continue
		}
		if strings.TrimSpace(src.SourceURL) == "" {
			continue
		}
		haystack := strings.ToLower(src.Title + " " + src.Summary + " " + src.Excerpt)
		score := 1
		for word := range words {
			if strings.Contains(haystack, word) {
				score += 2
			}
		}
		candidates = append(candidates, scored{src: src, score: score})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return candidates[i].src.Title < candidates[j].src.Title
		}
		return candidates[i].score > candidates[j].score
	})
	if limit <= 0 {
		limit = 5
	}
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	out := make([]CoachSource, 0, len(candidates))
	for _, item := range candidates {
		out = append(out, item.src)
	}
	return out
}

func (s *Server) knowledgeStatus(w http.ResponseWriter, cu *contextUser) {
	var approved int
	var verifiedQuotes int
	_ = s.Store.Read(func(db Database) error {
		for _, src := range db.CoachSources {
			if strings.TrimSpace(src.Summary) != "" {
				approved++
			}
			if src.QuoteVerified && strings.TrimSpace(src.Quote) != "" && strings.TrimSpace(src.SourceURL) != "" {
				verifiedQuotes++
			}
		}
		return nil
	})
	jsonOut(w, 200, map[string]any{
		"builtInChunks":   len(fitnessKnowledgeVault),
		"domains":         fitnessKnowledgeDomains(),
		"approvedSources": approved,
		"verifiedQuotes":  verifiedQuotes,
		"capabilities": []string{
			"training programming", "hypertrophy", "strength", "cardio", "nutrition", "supplements", "recovery", "pain-aware substitutions", "body composition", "behavior change", "evidence literacy",
		},
		"limitations": "FormForge is broad and source-aware, but it is not omniscient and does not replace medical diagnosis, licensed dietetic care, or current primary research review.",
	})
}

func groundingFromAgentSteps(steps []AgentStep) []AIGrounding {
	seen := map[string]bool{}
	out := []AIGrounding{}
	for _, step := range steps {
		if step.Tool != "web_fetch" || strings.TrimSpace(step.SourceURL) == "" || seen[step.SourceURL] {
			continue
		}
		seen[step.SourceURL] = true
		label := strings.TrimSpace(step.Input)
		if label == "" {
			label = step.SourceURL
		}
		out = append(out, AIGrounding{Kind: "web_source", Label: "Web source: " + label, URL: step.SourceURL})
		if len(out) == 5 {
			break
		}
	}
	return out
}

func approvedLinkBlock(db Database, userID, query string) string {
	sources := approvedSourceLinks(db, userID, query, 5)
	if len(sources) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("APPROVED LINKS RELEVANT TO THIS QUESTION:\n")
	for _, src := range sources {
		fmt.Fprintf(&b, "- %s: %s\n", src.Title, src.SourceURL)
	}
	return strings.TrimSpace(b.String())
}
