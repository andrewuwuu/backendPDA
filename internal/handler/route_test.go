package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// testRouteResult tracks which handler was reached during a route test.
type testRouteResult struct {
	handler string
}

// buildTestRouter creates a Chi router that mirrors the production route
// structure but uses lightweight test handlers that record which route
// was reached. This tests Chi's radix-tree route matching in isolation
// without requiring database connections or real services.
func buildTestRouter(result *testRouteResult) chi.Router {
	r := chi.NewRouter()

	// Public health endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		result.handler = "health"
		w.WriteHeader(http.StatusOK)
	})

	r.Route("/api", func(api chi.Router) {
		api.Post("/auth/login", func(w http.ResponseWriter, r *http.Request) {
			result.handler = "login"
			w.WriteHeader(http.StatusOK)
		})

		api.Group(func(protected chi.Router) {
			// Simulate auth middleware that blocks unauthenticated requests.
			protected.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Header.Get("X-Test-Auth") == "" {
						result.handler = "auth_rejected"
						w.WriteHeader(http.StatusUnauthorized)
						return
					}
					next.ServeHTTP(w, r)
				})
			})

			protected.Post("/auth/logout", func(w http.ResponseWriter, r *http.Request) {
				result.handler = "logout"
				w.WriteHeader(http.StatusOK)
			})

			// Reading routes
			protected.Get("/pda/realtime", func(w http.ResponseWriter, r *http.Request) {
				result.handler = "pda_realtime"
				w.WriteHeader(http.StatusOK)
			})

			// Station routes — static before param
			protected.Get("/stations", func(w http.ResponseWriter, r *http.Request) {
				result.handler = "stations_list"
				w.WriteHeader(http.StatusOK)
			})
			protected.Post("/stations/sync", func(w http.ResponseWriter, r *http.Request) {
				result.handler = "stations_sync"
				w.WriteHeader(http.StatusOK)
			})
			protected.Get("/stations/{namaLokasi}", func(w http.ResponseWriter, r *http.Request) {
				result.handler = "station_get"
				w.WriteHeader(http.StatusOK)
			})

			// Formula routes — static before param
			protected.Get("/formulas", func(w http.ResponseWriter, r *http.Request) {
				result.handler = "formulas_list"
				w.WriteHeader(http.StatusOK)
			})
			protected.Get("/formulas/grouped", func(w http.ResponseWriter, r *http.Request) {
				result.handler = "formulas_grouped"
				w.WriteHeader(http.StatusOK)
			})
			protected.Get("/formulas/id/{id}", func(w http.ResponseWriter, r *http.Request) {
				result.handler = "formula_by_id"
				w.WriteHeader(http.StatusOK)
			})
			protected.Get("/formulas/{namaLokasi}", func(w http.ResponseWriter, r *http.Request) {
				result.handler = "formulas_by_station"
				w.WriteHeader(http.StatusOK)
			})

			// Alert level routes — static before param
			protected.Get("/alert-levels", func(w http.ResponseWriter, r *http.Request) {
				result.handler = "alert_levels_list"
				w.WriteHeader(http.StatusOK)
			})
			protected.Get("/alert-levels/filter/{level}", func(w http.ResponseWriter, r *http.Request) {
				result.handler = "alert_levels_filter"
				w.WriteHeader(http.StatusOK)
			})
			protected.Get("/alert-levels/{namaLokasi}", func(w http.ResponseWriter, r *http.Request) {
				result.handler = "alert_level_get"
				w.WriteHeader(http.StatusOK)
			})

			// Admin routes
			protected.Group(func(admin chi.Router) {
				admin.Use(func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.Header.Get("X-Test-Role") != "admin" {
							result.handler = "admin_rejected"
							w.WriteHeader(http.StatusForbidden)
							return
						}
						next.ServeHTTP(w, r)
					})
				})

				admin.Get("/debug/jwt", func(w http.ResponseWriter, r *http.Request) {
					result.handler = "debug_jwt"
					w.WriteHeader(http.StatusOK)
				})
			})
		})
	})

	return r
}

func TestRouteHealthPublic(t *testing.T) {
	result := &testRouteResult{}
	router := buildTestRouter(result)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /health: expected 200, got %d", w.Code)
	}
	if result.handler != "health" {
		t.Errorf("GET /health: expected handler 'health', got %q", result.handler)
	}
}

func TestRouteLoginPublic(t *testing.T) {
	result := &testRouteResult{}
	router := buildTestRouter(result)

	req := httptest.NewRequest("POST", "/api/auth/login", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("POST /api/auth/login: expected 200, got %d", w.Code)
	}
	if result.handler != "login" {
		t.Errorf("POST /api/auth/login: expected handler 'login', got %q", result.handler)
	}
}

func TestRouteProtectedWithoutAuth(t *testing.T) {
	result := &testRouteResult{}
	router := buildTestRouter(result)

	req := httptest.NewRequest("GET", "/api/pda/realtime", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("GET /api/pda/realtime (no auth): expected 401, got %d", w.Code)
	}
}

func TestRouteDebugJWTWithoutAuth(t *testing.T) {
	result := &testRouteResult{}
	router := buildTestRouter(result)

	req := httptest.NewRequest("GET", "/api/debug/jwt", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("GET /api/debug/jwt (no auth): expected 401, got %d", w.Code)
	}
}

func TestRouteFormulasGroupedVsStation(t *testing.T) {
	result := &testRouteResult{}
	router := buildTestRouter(result)

	// "grouped" should resolve to grouped route, not station param route
	req := httptest.NewRequest("GET", "/api/formulas/grouped", nil)
	req.Header.Set("X-Test-Auth", "yes")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if result.handler != "formulas_grouped" {
		t.Errorf("GET /api/formulas/grouped: expected 'formulas_grouped', got %q", result.handler)
	}
}

func TestRouteFormulasIDVsStation(t *testing.T) {
	result := &testRouteResult{}
	router := buildTestRouter(result)

	// "/formulas/id/123" should resolve as formula-by-ID, not station
	req := httptest.NewRequest("GET", "/api/formulas/id/123", nil)
	req.Header.Set("X-Test-Auth", "yes")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if result.handler != "formula_by_id" {
		t.Errorf("GET /api/formulas/id/123: expected 'formula_by_id', got %q", result.handler)
	}
}

func TestRouteFormulasByStationParam(t *testing.T) {
	result := &testRouteResult{}
	router := buildTestRouter(result)

	req := httptest.NewRequest("GET", "/api/formulas/some-station", nil)
	req.Header.Set("X-Test-Auth", "yes")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if result.handler != "formulas_by_station" {
		t.Errorf("GET /api/formulas/some-station: expected 'formulas_by_station', got %q", result.handler)
	}
}

func TestRouteAlertLevelsFilterVsStation(t *testing.T) {
	result := &testRouteResult{}
	router := buildTestRouter(result)

	req := httptest.NewRequest("GET", "/api/alert-levels/filter/normal", nil)
	req.Header.Set("X-Test-Auth", "yes")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if result.handler != "alert_levels_filter" {
		t.Errorf("GET /api/alert-levels/filter/normal: expected 'alert_levels_filter', got %q", result.handler)
	}
}

func TestRouteStationsSyncPost(t *testing.T) {
	result := &testRouteResult{}
	router := buildTestRouter(result)

	req := httptest.NewRequest("POST", "/api/stations/sync", nil)
	req.Header.Set("X-Test-Auth", "yes")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if result.handler != "stations_sync" {
		t.Errorf("POST /api/stations/sync: expected 'stations_sync', got %q", result.handler)
	}
}

func TestRouteStationsSyncGetNotAllowed(t *testing.T) {
	result := &testRouteResult{}
	router := buildTestRouter(result)

	req := httptest.NewRequest("GET", "/api/stations/sync", nil)
	req.Header.Set("X-Test-Auth", "yes")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// "sync" on GET should match the station param route, not the sync handler
	if result.handler != "station_get" {
		t.Errorf("GET /api/stations/sync: expected 'station_get' (param match), got %q", result.handler)
	}
}

func TestRouteAdminWithoutRole(t *testing.T) {
	result := &testRouteResult{}
	router := buildTestRouter(result)

	req := httptest.NewRequest("GET", "/api/debug/jwt", nil)
	req.Header.Set("X-Test-Auth", "yes")
	// No X-Test-Role header
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("GET /api/debug/jwt (no admin): expected 403, got %d", w.Code)
	}
}

func TestRouteAdminWithRole(t *testing.T) {
	result := &testRouteResult{}
	router := buildTestRouter(result)

	req := httptest.NewRequest("GET", "/api/debug/jwt", nil)
	req.Header.Set("X-Test-Auth", "yes")
	req.Header.Set("X-Test-Role", "admin")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /api/debug/jwt (admin): expected 200, got %d", w.Code)
	}
	if result.handler != "debug_jwt" {
		t.Errorf("GET /api/debug/jwt (admin): expected 'debug_jwt', got %q", result.handler)
	}
}
