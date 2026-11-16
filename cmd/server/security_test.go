package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"monitor-infra/internal/storage"
)

// Run only after the coordinator releases QA:
// CGO_ENABLED=1 go test ./cmd/server -run '^TestSecurity' -count=1
// No listener is opened: requests go through the real registered ServeMux.

type securityFixture struct {
	server    *Server
	machineID int64
}

func securityToken(t *testing.T) string {
	t.Helper()
	var token [32]byte
	if _, err := rand.Read(token[:]); err != nil {
		t.Fatal("cannot generate ephemeral test token")
	}
	return hex.EncodeToString(token[:])
}

func newSecurityFixture(t *testing.T, token string, insecureDev bool) securityFixture {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "security.db")
	store, err := storage.New(dbPath, 1)
	if err != nil {
		t.Fatal("cannot initialize isolated SQLite storage")
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error("cannot close isolated SQLite storage")
		}
	})
	id, err := store.SaveMetrics(&storage.MetricPayload{
		Hostname:   "security-seeded-agent",
		IP:         "192.0.2.1",
		GroupName:  "test",
		SwarmRole:  "none",
		CPUPercent: 12.5,
	})
	if err != nil {
		t.Fatal("cannot seed isolated metrics")
	}
	config := &Config{Token: token, DBPath: dbPath, RetentionDays: 1}
	// Decode the new option instead of referencing a new symbol, allowing this
	// same test file to compile on the original HEAD for before/after evidence.
	// Original Config ignores this field; the behavioral cases still expose its
	// unconditional token-empty bypass rather than failing at compilation.
	if insecureDev {
		if err := json.Unmarshal([]byte(`{"InsecureDev":true}`), config); err != nil {
			t.Fatal("cannot configure local insecure development mode")
		}
	}
	server := &Server{config: config, storage: store, mux: http.NewServeMux()}
	server.registerRoutes()
	return securityFixture{server: server, machineID: id}
}

func (f securityFixture) readRoutes() []struct {
	path string
	code int
} {
	return []struct {
		path string
		code int
	}{
		{"/", http.StatusOK},
		{"/api/machines", http.StatusOK},
		{fmt.Sprintf("/api/machines/%d", f.machineID), http.StatusOK},
		{fmt.Sprintf("/api/machines/%d/metrics?hours=1", f.machineID), http.StatusOK},
		{"/api/stats", http.StatusOK},
		{"/api/metrics", http.StatusMethodNotAllowed},
		{"/api/", http.StatusNotFound},
		{"/api/unknown", http.StatusNotFound},
		{"/api/health/", http.StatusNotFound},
		{"/api/health/extra", http.StatusNotFound},
	}
}

func securityRequest(method, route, remote, authorization string, body []byte) *http.Request {
	req := httptest.NewRequest(method, route, bytes.NewReader(body))
	req.RemoteAddr = remote
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func securityBasic(username, password string) string {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth(username, password)
	return req.Header.Get("Authorization")
}

func securityServe(t *testing.T, server *Server, req *http.Request, want int) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	server.mux.ServeHTTP(recorder, req)
	if recorder.Code != want {
		// Do not print headers, body or Config: they could contain a test token
		// if the code regresses. Only safe method/path/status reach test output.
		t.Errorf("%s %s returned %d; want %d", req.Method, req.URL.Path, recorder.Code, want)
	}
	return recorder
}

func TestSecurityAllReadRoutesRequireAuthentication(t *testing.T) {
	token := securityToken(t)
	fixture := newSecurityFixture(t, token, false)
	credentials := []struct {
		name string
		auth string
	}{
		{"missing", ""},
		{"wrong-bearer", "Bearer " + securityToken(t)},
		{"short-bearer", "Bearer wrong"},
		{"bare-token", token},
		{"malformed-basic", "Basic !!!"},
		{"wrong-basic-password", securityBasic("monitor", securityToken(t))},
		{"wrong-basic-user", securityBasic("other", token)},
		{"empty-basic-password", securityBasic("monitor", "")},
	}
	for _, route := range fixture.readRoutes() {
		t.Run(route.path, func(t *testing.T) {
			for _, credential := range credentials {
				t.Run(credential.name, func(t *testing.T) {
					req := securityRequest(http.MethodGet, route.path, "198.51.100.10:4567", credential.auth, nil)
					response := securityServe(t, fixture.server, req, http.StatusUnauthorized)
					if strings.Contains(response.Body.String(), token) {
						t.Error("authentication failure must not disclose configured token")
					}
					if route.path == "/" && credential.name == "missing" &&
						!strings.HasPrefix(response.Header().Get("WWW-Authenticate"), "Basic ") {
						t.Error("dashboard must provide a Basic authentication challenge")
					}
				})
			}
		})
	}
}

func TestSecurityReadAuthenticationPositiveControls(t *testing.T) {
	token := securityToken(t)
	fixture := newSecurityFixture(t, token, false)
	for _, route := range fixture.readRoutes() {
		t.Run(route.path, func(t *testing.T) {
			req := securityRequest(http.MethodGet, route.path, "198.51.100.10:4567", securityBasic("monitor", token), nil)
			securityServe(t, fixture.server, req, route.code)
			req = securityRequest(http.MethodGet, route.path, "198.51.100.10:4567", "Bearer "+token, nil)
			securityServe(t, fixture.server, req, route.code)
		})
	}
	req := securityRequest(http.MethodHead, "/", "198.51.100.10:4567", securityBasic("monitor", token), nil)
	securityServe(t, fixture.server, req, http.StatusOK)
}

func TestSecurityHealthProbeRemainsPublicAndMinimal(t *testing.T) {
	configuredToken := securityToken(t)
	for _, configuration := range []struct {
		name  string
		token string
	}{
		{"empty-token", ""},
		{"whitespace-token", " \t "},
		{"configured-token", configuredToken},
	} {
		for _, insecureDev := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/dev=%t", configuration.name, insecureDev), func(t *testing.T) {
				fixture := newSecurityFixture(t, configuration.token, insecureDev)
				for _, remote := range []string{"127.0.0.1:4567", "198.51.100.10:4567"} {
					for _, authorization := range []string{
						"",
						"Bearer wrong",
						"Bearer " + configuredToken,
						"Basic !!!",
						securityBasic("monitor", "wrong"),
						securityBasic("monitor", configuredToken),
					} {
						req := securityRequest(http.MethodGet, "/api/health", remote, authorization, nil)
						response := securityServe(t, fixture.server, req, http.StatusOK)
						// The public probe may expose liveness only, including when
						// machines exist. Never print an unexpected response body.
						if strings.TrimSpace(response.Body.String()) != `{"status":"ok"}` {
							t.Error("public health must return only the minimal status object")
						}
						if !strings.HasPrefix(response.Header().Get("Content-Type"), "application/json") {
							t.Error("public health must return JSON")
						}
						if response.Header().Get("WWW-Authenticate") != "" {
							t.Error("public probe must not challenge for authentication")
						}
					}
					// Infrastructure probes use GET today; HEAD must remain public too.
					req := securityRequest(http.MethodHead, "/api/health", remote, "", nil)
					securityServe(t, fixture.server, req, http.StatusOK)
				}
			})
		}
	}
}

func TestSecurityBlankTokenFailsClosedByDefault(t *testing.T) {
	for _, token := range []string{"", " \t "} {
		name := "empty"
		if token != "" {
			name = "whitespace"
		}
		t.Run(name, func(t *testing.T) {
			fixture := newSecurityFixture(t, token, false)
			for _, remote := range []string{"127.0.0.1:4567", "[::1]:4567", "198.51.100.10:4567"} {
				for _, route := range fixture.readRoutes() {
					for _, authorization := range []string{"", "Bearer " + token, securityBasic("monitor", token)} {
						req := securityRequest(http.MethodGet, route.path, remote, authorization, nil)
						securityServe(t, fixture.server, req, http.StatusUnauthorized)
					}
				}
				req := securityRequest(http.MethodPost, "/api/metrics", remote, "", []byte(`{"hostname":"blocked-agent"}`))
				securityServe(t, fixture.server, req, http.StatusUnauthorized)
			}
		})
	}
}

func TestSecurityInsecureDevIsExplicitAndLimitedToLoopback(t *testing.T) {
	fixture := newSecurityFixture(t, "", true)
	for _, remote := range []string{"127.0.0.1:4567", "[::1]:4567"} {
		for _, route := range fixture.readRoutes() {
			req := securityRequest(http.MethodGet, route.path, remote, "", nil)
			securityServe(t, fixture.server, req, route.code)
		}
	}
	for _, remote := range []string{"198.51.100.10:4567", "[2001:db8::1]:4567", "localhost:4567", "invalid", ""} {
		for _, route := range fixture.readRoutes() {
			req := securityRequest(http.MethodGet, route.path, remote, "", nil)
			// Forwarding headers are untrusted and cannot turn a remote peer into loopback.
			req.Header.Set("X-Forwarded-For", "127.0.0.1")
			req.Header.Set("X-Real-IP", "127.0.0.1")
			securityServe(t, fixture.server, req, http.StatusUnauthorized)
		}
		req := securityRequest(http.MethodPost, "/api/metrics", remote, "", []byte(`{"hostname":"blocked-agent"}`))
		securityServe(t, fixture.server, req, http.StatusUnauthorized)
	}
	// The explicit dev flag also overrides a configured token, but only locally.
	configuredToken := newSecurityFixture(t, securityToken(t), true)
	for _, route := range configuredToken.readRoutes() {
		req := securityRequest(http.MethodGet, route.path, "127.0.0.1:4567", "", nil)
		securityServe(t, configuredToken.server, req, route.code)
		req = securityRequest(http.MethodGet, route.path, "198.51.100.10:4567", "Bearer "+configuredToken.server.config.Token, nil)
		securityServe(t, configuredToken.server, req, http.StatusUnauthorized)
	}
}

func TestSecurityAgentMetricsStillRequireBearerAndPersist(t *testing.T) {
	token := securityToken(t)
	fixture := newSecurityFixture(t, token, false)
	body := []byte(`{"hostname":"security-new-agent","ip":"192.0.2.2","group":"test","swarm_role":"none","cpu_percent":37.5}`)
	for _, authorization := range []string{"", "Bearer wrong", securityBasic("monitor", token)} {
		req := securityRequest(http.MethodPost, "/api/metrics", "198.51.100.10:4567", authorization, body)
		securityServe(t, fixture.server, req, http.StatusUnauthorized)
	}
	machines, err := fixture.server.storage.GetMachinesWithMetrics()
	if err != nil || len(machines) != 1 {
		t.Fatal("unauthorized metric posts must not create machines")
	}
	req := securityRequest(http.MethodPost, "/api/metrics", "198.51.100.10:4567", "Bearer "+token, body)
	response := securityServe(t, fixture.server, req, http.StatusCreated)
	var receipt struct {
		Status    string `json:"status"`
		MachineID int64  `json:"machine_id"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &receipt); err != nil || receipt.Status != "ok" || receipt.MachineID <= 0 {
		t.Fatal("authenticated agent must receive a valid creation receipt")
	}
	machine, err := fixture.server.storage.GetMachineByID(receipt.MachineID)
	if err != nil || machine == nil || machine.Hostname != "security-new-agent" {
		t.Fatal("authenticated metric post must persist the agent")
	}
	history, err := fixture.server.storage.GetMetricsHistory(receipt.MachineID, 1)
	if err != nil || len(history) != 1 {
		t.Fatal("authenticated metric post must persist one metrics record")
	}
	// Verify a browser's Basic credentials cannot authorize unsafe API methods.
	for _, method := range []string{http.MethodPut, http.MethodPatch, http.MethodDelete} {
		req := securityRequest(method, "/api/metrics", "198.51.100.10:4567", securityBasic("monitor", token), body)
		securityServe(t, fixture.server, req, http.StatusUnauthorized)
	}
}

func TestSecurityAgentBootstrapRemainsPublic(t *testing.T) {
	for _, token := range []string{"", securityToken(t)} {
		fixture := newSecurityFixture(t, token, false)
		for _, route := range []struct {
			path string
			code int
		}{
			{"/install.sh", http.StatusOK},
			{"/download/not-a-real-agent", http.StatusNotFound},
		} {
			req := securityRequest(http.MethodGet, route.path, "198.51.100.10:4567", "", nil)
			securityServe(t, fixture.server, req, route.code)
		}
		for _, route := range []string{"/download/agent-linux-amd64", "/download/agent-linux-arm64"} {
			req := securityRequest(http.MethodGet, route, "198.51.100.10:4567", "", nil)
			recorder := httptest.NewRecorder()
			fixture.server.mux.ServeHTTP(recorder, req)
			// A fresh clone may have no binaries. Both states must remain public;
			// this test deliberately does not write or build release artifacts.
			if recorder.Code != http.StatusOK && recorder.Code != http.StatusNotFound {
				t.Errorf("public download returned %d; want 200 or 404", recorder.Code)
			}
		}
	}
}
