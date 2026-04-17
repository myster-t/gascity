package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gastownhall/gascity/internal/beads"
	"github.com/gastownhall/gascity/internal/config"
)

// newSlingTestServer creates a test handler wrapping a Server that has a
// fake runner injected (captures commands without executing real shell
// processes).
func newSlingTestServer(t *testing.T) (http.Handler, *fakeMutatorState) {
	t.Helper()
	state := newFakeMutatorState(t)
	state.cfg.Rigs[0].Prefix = "gc" // match MemStore's auto-generated prefix
	srv := New(state)
	srv.SlingRunnerFunc = func(_ string, _ string, _ map[string]string) (string, error) {
		return "", nil // no-op runner
	}
	return newTestCityHandlerWith(t, state, srv), state
}

func TestSlingWithBead(t *testing.T) {
	h, state := newSlingTestServer(t)
	store := state.stores["myrig"]
	b, err := store.Create(beads.Bead{Title: "test task", Type: "task"})
	if err != nil {
		t.Fatal(err)
	}

	body := `{"target":"myrig/worker","bead":"` + b.ID + `"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newPostRequest(cityURL(state, "/sling"), strings.NewReader(body)))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["status"] != "slung" {
		t.Fatalf("status = %q, want %q", resp["status"], "slung")
	}
	if resp["mode"] != "direct" {
		t.Fatalf("mode = %q, want %q", resp["mode"], "direct")
	}
}

func TestSlingMissingTarget(t *testing.T) {
	h, state := newSlingTestServer(t); _ = state
	body := `{"bead":"abc"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newPostRequest(cityURL(state, "/sling"), strings.NewReader(body)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestSlingTargetNotFound(t *testing.T) {
	h, state := newSlingTestServer(t); _ = state
	body := `{"target":"nonexistent","bead":"abc"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newPostRequest(cityURL(state, "/sling"), strings.NewReader(body)))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestSlingMissingBeadAndFormula(t *testing.T) {
	h, state := newSlingTestServer(t); _ = state
	body := `{"target":"myrig/worker"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newPostRequest(cityURL(state, "/sling"), strings.NewReader(body)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestSlingBeadAndFormulaMutuallyExclusive(t *testing.T) {
	h, state := newSlingTestServer(t); _ = state
	body := `{"target":"myrig/worker","bead":"abc","formula":"xyz"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newPostRequest(cityURL(state, "/sling"), strings.NewReader(body)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestSlingRejectsVarsWithoutFormula(t *testing.T) {
	h, state := newSlingTestServer(t); _ = state
	body := `{"target":"myrig/worker","bead":"BD-42","vars":{"issue":"BD-42"}}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newPostRequest(cityURL(state, "/sling"), strings.NewReader(body)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}
}

func TestSlingRejectsScopeWithoutFormula(t *testing.T) {
	h, state := newSlingTestServer(t); _ = state
	body := `{"target":"myrig/worker","bead":"BD-42","scope_kind":"city","scope_ref":"test-city"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newPostRequest(cityURL(state, "/sling"), strings.NewReader(body)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}
}

func TestSlingRejectsPartialScope(t *testing.T) {
	h, state := newSlingTestServer(t); _ = state
	body := `{"target":"myrig/worker","formula":"mol-review","scope_kind":"city"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newPostRequest(cityURL(state, "/sling"), strings.NewReader(body)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}
}

func TestSlingPoolTarget(t *testing.T) {
	h, state := newSlingTestServer(t)
	state.cfg.Agents = []config.Agent{
		{
			Name:              "polecat",
			Dir:               "myrig",
			MinActiveSessions: intPtr(0), MaxActiveSessions: intPtr(3),
		},
	}
	store := state.stores["myrig"]
	b, err := store.Create(beads.Bead{Title: "test task", Type: "task"})
	if err != nil {
		t.Fatal(err)
	}

	body := `{"target":"myrig/polecat","bead":"` + b.ID + `"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newPostRequest(cityURL(state, "/sling"), strings.NewReader(body)))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["status"] != "slung" {
		t.Fatalf("status = %q, want slung", resp["status"])
	}
}
