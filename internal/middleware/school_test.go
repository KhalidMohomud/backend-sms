package middleware

import (
	"backendapi/internal/authz"
	"backendapi/internal/model"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequireSchoolUsesAccountSchoolAndRejectsOverride(t *testing.T) {
	gin.SetMode(gin.TestMode)
	schoolID := uint64(3)
	principal := authz.Principal{UserID: 9, SchoolID: &schoolID, Role: model.RoleSchoolAdmin}

	tests := []struct {
		name       string
		header     string
		wantStatus int
	}{
		{name: "account school", wantStatus: http.StatusOK},
		{name: "matching header", header: "3", wantStatus: http.StatusOK},
		{name: "different school", header: "4", wantStatus: http.StatusForbidden},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/", setTestPrincipal(principal), RequireSchool(), func(c *gin.Context) {
				got, ok := SchoolID(c)
				if !ok || got != schoolID {
					t.Fatalf("SchoolID() = %d, %v; want %d, true", got, ok, schoolID)
				}
				c.Status(http.StatusOK)
			})
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if test.header != "" {
				req.Header.Set("X-School-ID", test.header)
			}
			response := httptest.NewRecorder()
			r.ServeHTTP(response, req)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
		})
	}
}

func TestRequireSchoolRequiresSuperAdminHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	principal := authz.Principal{UserID: 1, Role: model.RoleSuperAdmin}

	for _, schoolID := range []uint64{0, 12} {
		r := gin.New()
		r.GET("/", setTestPrincipal(principal), RequireSchool(), func(c *gin.Context) { c.Status(http.StatusOK) })
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if schoolID != 0 {
			req.Header.Set("X-School-ID", strconv.FormatUint(schoolID, 10))
		}
		response := httptest.NewRecorder()
		r.ServeHTTP(response, req)
		want := http.StatusBadRequest
		if schoolID != 0 {
			want = http.StatusOK
		}
		if response.Code != want {
			t.Fatalf("school %d status = %d, want %d", schoolID, response.Code, want)
		}
	}
}

func TestRequirePermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		principal  authz.Principal
		wantStatus int
	}{
		{name: "granted", principal: authz.Principal{Permissions: []string{"manage_users"}}, wantStatus: http.StatusOK},
		{name: "denied", principal: authz.Principal{}, wantStatus: http.StatusForbidden},
		{name: "superadmin", principal: authz.Principal{Role: model.RoleSuperAdmin}, wantStatus: http.StatusOK},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/", setTestPrincipal(test.principal), RequirePermission("manage_users"), func(c *gin.Context) { c.Status(http.StatusOK) })
			response := httptest.NewRecorder()
			r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
		})
	}
}

func setTestPrincipal(principal authz.Principal) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(PrincipalKey, principal)
		c.Next()
	}
}
