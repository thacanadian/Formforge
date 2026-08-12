package main

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"formforge/internal/core"
)

func TestChooseAvailablePortSkipsOccupiedPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	_, raw, _ := net.SplitHostPort(ln.Addr().String())
	occupied, _ := strconv.Atoi(raw)
	chosen, err := chooseAvailablePort("127.0.0.1", occupied, 5)
	if err != nil {
		t.Fatal(err)
	}
	if chosen == occupied {
		t.Fatalf("selected occupied port %d", occupied)
	}
}

func TestExistingServerRequiresFormForgeIdentity(t *testing.T) {
	fake := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer fake.Close()
	if existingServer(strings.TrimRight(fake.URL, "/")) {
		t.Fatal("accepted an unrelated HTTPS service as FormForge")
	}

	real := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "product": "formforge"})
	}))
	defer real.Close()
	if !existingServer(strings.TrimRight(real.URL, "/")) {
		t.Fatal("did not recognize a FormForge health response")
	}
}

func TestApplyEnvironmentConfiguration(t *testing.T) {
	dir := t.TempDir()
	store, err := core.OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	key, _, err := core.EnsureMasterKey(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("OPENAI_API_KEY", "test-render-secret")
	t.Setenv("FORMFORGE_AI_MODE", "auto")
	t.Setenv("FORMFORGE_AI_MODEL", "gpt-4o-mini")
	t.Setenv("FORMFORGE_AGENT_ENABLED", "true")
	t.Setenv("FORMFORGE_AGENT_MAX_STEPS", "7")
	if err := applyEnvironmentConfiguration(store, key); err != nil {
		t.Fatal(err)
	}
	var settings core.Settings
	if err := store.Read(func(db core.Database) error { settings = db.Settings; return nil }); err != nil {
		t.Fatal(err)
	}
	plain, err := core.DecryptSecret(key, settings.AIAPIKeyEncrypted)
	if err != nil || plain != "test-render-secret" {
		t.Fatalf("environment API key was not encrypted and saved: %q %v", plain, err)
	}
	if settings.AIMode != "auto" || settings.AIModel != "gpt-4o-mini" || !settings.AgentEnabled || settings.AgentMaxSteps != 7 {
		t.Fatalf("unexpected environment settings: %#v", settings)
	}
}
