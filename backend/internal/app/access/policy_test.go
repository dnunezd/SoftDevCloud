package access_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dnunezd/SoftDevCloud/backend/internal/app/access"
	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
)

// RF-12.1, RN-03 y RN-05: el profesor ve y opera solo sus espacios; el
// administrador ve todos pero no opera como profesor responsable.
func TestWorkspacePolicies(t *testing.T) {
	workspace := domain.Workspace{ID: "ws-1", ProfessorID: "prof-1"}

	tests := []struct {
		name       string
		principal  access.Principal
		wantView   bool
		wantManage bool
	}{
		{"profesor responsable", access.Principal{UserID: "prof-1", Roles: []domain.Role{domain.RoleProfessor}}, true, true},
		{"otro profesor", access.Principal{UserID: "prof-2", Roles: []domain.Role{domain.RoleProfessor}}, false, false},
		{"administrador", access.Principal{UserID: "admin-1", Roles: []domain.Role{domain.RoleAdmin}}, true, false},
		{"monitor", access.Principal{UserID: "mon-1", Roles: []domain.Role{domain.RoleMonitor}}, false, false},
		{"mismo usuario sin rol de profesor", access.Principal{UserID: "prof-1", Roles: []domain.Role{domain.RoleMonitor}}, false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantView, tc.principal.CanViewWorkspace(workspace), "ver")
			assert.Equal(t, tc.wantManage, tc.principal.CanManageWorkspace(workspace), "operar")
		})
	}
}
