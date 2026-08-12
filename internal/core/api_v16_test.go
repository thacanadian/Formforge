package core

import (
	"strings"
	"testing"
)

func TestAppearanceUnitsAndKnowledgeStatus(t *testing.T) {
	_, _, admin, closeFn := newTestApp(t)
	defer closeFn()
	admin.req(t, "POST", "/api/auth/setup", map[string]any{
		"name":     "Owner",
		"email":    "owner-units@example.com",
		"password": "StrongPassword123",
		"profile": map[string]any{
			"name": "Owner", "age": 25, "gender": "male", "heightCm": 188, "weightKg": 95,
			"goal": "Build muscle", "experience": "intermediate", "daysPerWeek": 4, "equipment": "Full gym",
		},
		"appearancePreferences": map[string]any{
			"preset": "midnight", "measurementSystem": "imperial", "navigationMode": "focused",
		},
	}, 201)
	theme := admin.req(t, "GET", "/api/theme", nil, 200)
	current := theme["current"].(map[string]any)
	if current["measurementSystem"] != "imperial" || current["navigationMode"] != "focused" {
		t.Fatalf("unexpected preferences: %#v", current)
	}
	updated := admin.req(t, "PUT", "/api/theme", map[string]any{
		"preset": "arctic", "accent": "#0077ff", "background": "#f2f6fb", "surface": "#ffffff", "text": "#152033",
		"radius": 16, "density": "comfortable", "measurementSystem": "metric", "navigationMode": "full",
	}, 200)
	if updated["measurementSystem"] != "metric" || updated["navigationMode"] != "full" {
		t.Fatalf("theme update did not preserve preferences: %#v", updated)
	}
	status := admin.req(t, "GET", "/api/knowledge/status", nil, 200)
	if status["builtInChunks"].(float64) < 50 {
		t.Fatalf("knowledge vault is unexpectedly small: %#v", status)
	}
	if len(status["domains"].([]any)) < 10 {
		t.Fatalf("knowledge domains missing: %#v", status)
	}
}

func TestKnowledgeRetrievalAndUnitFormatting(t *testing.T) {
	chunks := fitnessKnowledgeSearch("hypertrophy volume proximity to failure", 5)
	if len(chunks) == 0 || chunks[0].Domain != "Hypertrophy" {
		t.Fatalf("unexpected knowledge retrieval: %#v", chunks)
	}
	db := NewDatabase()
	db.ThemePreferences["u"] = ThemePreferences{MeasurementSystem: "imperial"}
	if got := formatWeightForUser(db, "u", 100); got != "220.5 lb" {
		t.Fatalf("imperial conversion = %q", got)
	}
	db.ThemePreferences["u"] = ThemePreferences{MeasurementSystem: "metric"}
	if got := formatWeightForUser(db, "u", 100); got != "100.0 kg" {
		t.Fatalf("metric conversion = %q", got)
	}
}

func TestSourceRequestsAndVerifiedQuotes(t *testing.T) {
	if !wantsSourceLinks("Can you link the studies and sources?") {
		t.Fatal("source/link request was not detected")
	}
	if wantsSourceLinks("Give me a simple chest workout") {
		t.Fatal("ordinary coaching request was misclassified as a source request")
	}
	if !wantsExactQuote("Give me the exact quote word for word") {
		t.Fatal("exact quote request was not detected")
	}

	db := NewDatabase()
	db.CoachPreferences["u"] = CoachPreferences{
		UserID:     "u",
		Influences: []CoachSelection{{ProfileID: "jeff-nippard", Weight: 100}},
	}
	db.CoachSources = append(db.CoachSources, CoachSource{
		ID:            "source-1",
		ProfileID:     "jeff-nippard",
		Title:         "Verified test source",
		Kind:          "article",
		SourceURL:     "https://example.com/source",
		Summary:       "A test summary about progressive overload.",
		Quote:         "Test verified wording.",
		QuoteVerified: true,
	})
	reply := verifiedQuoteReply(db, "u")
	if !strings.Contains(reply, "Test verified wording.") || !strings.Contains(reply, "https://example.com/source") {
		t.Fatalf("verified quote reply omitted quote or source: %q", reply)
	}
	links := approvedSourceLinks(db, "u", "progressive overload", 3)
	if len(links) != 1 || links[0].SourceURL != "https://example.com/source" {
		t.Fatalf("approved source lookup failed: %#v", links)
	}
}
