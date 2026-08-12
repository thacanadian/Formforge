package core

import (
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"
)

func siblingClient(tc *testClient) *testClient {
	jar, _ := cookiejar.New(nil)
	return &testClient{c: &http.Client{Transport: tc.c.Transport, Jar: jar}, base: tc.base}
}

func TestLegalPagesConsentModerationBlockingAndDeletion(t *testing.T) {
	store, _, admin, closeFn := newTestApp(t)
	defer closeFn()
	admin.req(t, "POST", "/api/auth/setup", map[string]any{
		"name": "Owner", "email": "owner@example.com", "password": "StrongPassword123",
		"profile": map[string]any{"name": "Owner", "age": 30, "heightCm": 180, "weightKg": 80},
	}, 201)

	for _, path := range []string{"/legal", "/legal/terms", "/legal/privacy", "/legal/community", "/legal/subscription", "/legal/health-ai", "/legal/security", "/account-deletion"} {
		raw, headers := admin.reqRaw(t, "GET", path, nil, 200)
		if !strings.Contains(headers.Get("Content-Type"), "text/html") || !strings.Contains(string(raw), "FORMFORGE") {
			t.Fatalf("legal page %s did not render", path)
		}
	}

	alice := admin.req(t, "POST", "/api/admin/users", map[string]any{
		"name": "Alice", "email": "alice@example.com", "password": "AlicePassword123", "role": "user",
		"profile": map[string]any{"name": "Alice", "age": 22, "heightCm": 165, "weightKg": 62},
	}, 201)
	bob := admin.req(t, "POST", "/api/admin/users", map[string]any{
		"name": "Bob", "email": "bob@example.com", "password": "BobPassword123", "role": "user",
		"profile": map[string]any{"name": "Bob", "age": 24, "heightCm": 178, "weightKg": 78},
	}, 201)
	aliceID, bobID := alice["id"].(string), bob["id"].(string)

	ac := siblingClient(admin)
	ac.req(t, "POST", "/api/auth/login", map[string]any{"email": "alice@example.com", "password": "AlicePassword123"}, 200)
	ac.req(t, "POST", "/api/social/nudges", map[string]any{"toUserId": bobID, "message": "Train today?"}, 403)
	ac.req(t, "POST", "/api/legal/accept", map[string]any{"terms": true, "privacy": true, "community": true, "ageConfirmed": true}, 200)
	status := ac.req(t, "GET", "/api/legal/status", nil, 200)
	if ok, _ := status["eligibleForCommunity"].(bool); !ok {
		t.Fatal("community should be enabled after consent")
	}
	ac.req(t, "POST", "/api/social/nudges", map[string]any{"toUserId": bobID, "message": "kill yourself"}, 400)
	ac.req(t, "POST", "/api/social/blocks", map[string]any{"userId": bobID}, 201)
	ac.req(t, "POST", "/api/social/nudges", map[string]any{"toUserId": bobID, "message": "Train today?"}, 400)
	blocks := ac.req(t, "GET", "/api/social/blocks", nil, 200)["_array"].([]any)
	if len(blocks) != 1 {
		t.Fatal("block was not stored")
	}
	blockID := blocks[0].(map[string]any)["id"].(string)
	ac.req(t, "DELETE", "/api/social/blocks/"+blockID, nil, 200)
	nudge := ac.req(t, "POST", "/api/social/nudges", map[string]any{"toUserId": bobID, "message": "Want to train today?"}, 201)

	bc := siblingClient(admin)
	bc.req(t, "POST", "/api/auth/login", map[string]any{"email": "bob@example.com", "password": "BobPassword123"}, 200)
	bc.req(t, "POST", "/api/legal/accept", map[string]any{"terms": true, "privacy": true, "community": true, "ageConfirmed": true}, 200)
	report := bc.req(t, "POST", "/api/social/reports", map[string]any{"targetType": "nudge", "targetId": nudge["id"], "category": "harassment", "details": "Unwanted repeated contact."}, 201)
	if got := bc.req(t, "GET", "/api/social/nudges", nil, 200)["_array"].([]any); len(got) != 0 {
		t.Fatal("open report should hide content for reporter")
	}
	reports := admin.req(t, "GET", "/api/admin/moderation/reports", nil, 200)["_array"].([]any)
	if len(reports) != 1 {
		t.Fatal("report queue missing report")
	}
	admin.req(t, "PUT", "/api/admin/moderation/reports/"+report["id"].(string), map[string]any{"action": "remove_content", "reason": "Test moderation removal"}, 200)

	userReport := bc.req(t, "POST", "/api/social/reports", map[string]any{"targetType": "user", "targetId": aliceID, "category": "harassment", "details": "Repeated unwanted behavior."}, 201)
	reports = admin.req(t, "GET", "/api/admin/moderation/reports", nil, 200)["_array"].([]any)
	if len(reports) != 2 {
		t.Fatal("user report was not added to moderation queue")
	}
	admin.req(t, "PUT", "/api/admin/moderation/reports/"+userReport["id"].(string), map[string]any{"action": "suspend_7d", "reason": "Test community suspension"}, 200)
	ac.req(t, "POST", "/api/social/nudges", map[string]any{"toUserId": bobID, "message": "This should be blocked by suspension."}, 403)

	anon := siblingClient(admin)
	for i := 0; i < 5; i++ {
		anon.req(t, "POST", "/api/account-deletion/request", map[string]any{"email": "alice@example.com", "password": "wrong-password", "confirmation": "DELETE"}, 401)
	}
	anon.req(t, "POST", "/api/account-deletion/request", map[string]any{"email": "alice@example.com", "password": "wrong-password", "confirmation": "DELETE"}, 429)

	bc.req(t, "POST", "/api/nutrition", map[string]any{"date": "2026-08-04", "name": "Meal", "calories": 500, "protein": 30}, 201)
	bc.req(t, "DELETE", "/api/account", map[string]any{"password": "BobPassword123", "confirmation": "DELETE"}, 200)
	bc.req(t, "POST", "/api/auth/login", map[string]any{"email": "bob@example.com", "password": "BobPassword123"}, 401)
	_ = store.Read(func(db Database) error {
		if _, ok := db.Users[bobID]; ok {
			t.Fatal("deleted user remains")
		}
		if _, ok := db.Profiles[bobID]; ok {
			t.Fatal("deleted profile remains")
		}
		for _, x := range db.Nutrition {
			if x.UserID == bobID {
				t.Fatal("deleted nutrition remains")
			}
		}
		for _, x := range db.UserBlocks {
			if x.BlockerID == bobID || x.BlockedID == bobID {
				t.Fatal("deleted block remains")
			}
		}
		return nil
	})

	admin.req(t, "DELETE", "/api/account", map[string]any{"password": "StrongPassword123", "confirmation": "DELETE"}, 400)
	if aliceID == "" {
		t.Fatal("alice id missing")
	}
}
