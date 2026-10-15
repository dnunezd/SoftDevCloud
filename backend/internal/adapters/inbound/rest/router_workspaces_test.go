package rest_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dnunezd/SoftDevCloud/backend/internal/adapters/inbound/rest/dto"
	"github.com/dnunezd/SoftDevCloud/backend/internal/domain"
	"github.com/dnunezd/SoftDevCloud/backend/internal/ports"
)

const (
	periodID    = "5d6e7f80-91a2-4bce-8f30-4b5c6d7e8f90"
	workspaceID = "6e7f8091-a2b3-4cdf-9041-5c6d7e8f90a1"
)

func activePeriod() domain.AcademicPeriod {
	return domain.AcademicPeriod{
		ID:               periodID,
		Code:             "2026-10",
		StartDate:        time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
		EndDate:          time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC),
		CreationDeadline: time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC),
		Status:           domain.PeriodActive,
	}
}

func courseOf(professorID string, status domain.WorkspaceStatus) domain.Workspace {
	w := domain.Workspace{
		ID: workspaceID, PeriodID: periodID, ProfessorID: professorID, Type: domain.WorkspaceCourse,
		Name: "Arquitectura de Software", Observations: "Grupo nocturno", Status: status,
		StartDate: time.Date(2026, time.February, 2, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, time.June, 12, 0, 0, 0, 0, time.UTC),
	}
	if status == domain.WorkspaceClosed {
		closedAt := testNow
		w.ClosedAt = &closedAt
	}
	return w
}

// RF-03.2: solo el administrador define y cierra los períodos académicos.
func TestAdminPeriods(t *testing.T) {
	api := newTestAPI(t)
	token := api.tokenFor(t, adminLaptop)
	asAdmin := func() string {
		api.activeSession(adminLaptop, domain.RoleAdmin)
		return token
	}
	const body = `{"code":"2026-10","start_date":"2026-02-01","end_date":"2026-06-15","creation_deadline":"2026-02-28"}`

	t.Run("crea un período activo", func(t *testing.T) {
		api.periods.EXPECT().Create(mock.Anything, mock.MatchedBy(func(p domain.AcademicPeriod) bool {
			return p.Code == "2026-10" && p.Status == domain.PeriodActive
		})).RunAndReturn(func(_ context.Context, p domain.AcademicPeriod) (domain.AcademicPeriod, error) {
			p.ID = periodID
			return p, nil
		}).Once()

		rec := api.do(http.MethodPost, "/api/v1/admin/periods", body, asAdmin())

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		assert.Equal(t, "/api/v1/periods/"+periodID, rec.Header().Get("Location"))
		assert.Equal(t, dto.PeriodResponse{
			ID: periodID, Code: "2026-10", StartDate: "2026-02-01", EndDate: "2026-06-15",
			CreationDeadline: "2026-02-28", Status: "ACTIVO",
		}, decode[dto.PeriodResponse](t, rec))
	})

	t.Run("un profesor no puede crear períodos", func(t *testing.T) {
		api.activeSession(profLaptop, domain.RoleProfessor)

		assertError(t, api.do(http.MethodPost, "/api/v1/admin/periods", body, api.tokenFor(t, profLaptop)), http.StatusForbidden, "forbidden")
	})

	t.Run("datos inválidos con detalle por campo", func(t *testing.T) {
		rec := api.do(http.MethodPost, "/api/v1/admin/periods", `{"code":"","start_date":"ayer"}`, asAdmin())

		response := assertError(t, rec, http.StatusUnprocessableEntity, "validation_failed")
		assert.Contains(t, response.Error.Details, "code")
		assert.Contains(t, response.Error.Details, "start_date")
		assert.Contains(t, response.Error.Details, "end_date")
	})

	t.Run("código duplicado", func(t *testing.T) {
		api.periods.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.AcademicPeriod{}, domain.ErrPeriodCodeExists).Once()

		assertError(t, api.do(http.MethodPost, "/api/v1/admin/periods", body, asAdmin()), http.StatusConflict, "period_code_already_exists")
	})

	t.Run("JSON mal formado", func(t *testing.T) {
		assertError(t, api.do(http.MethodPost, "/api/v1/admin/periods", `{"code":`, asAdmin()), http.StatusBadRequest, "invalid_request")
	})

	t.Run("cierra un período", func(t *testing.T) {
		api.periods.EXPECT().GetByID(mock.Anything, periodID).Return(activePeriod(), nil).Once()
		api.periods.EXPECT().MarkClosed(mock.Anything, periodID).Return(nil).Once()

		rec := api.do(http.MethodPost, "/api/v1/admin/periods/"+periodID+"/close", "", asAdmin())

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Equal(t, "CERRADO", decode[dto.PeriodResponse](t, rec).Status)
	})

	t.Run("un período cerrado no se cierra otra vez", func(t *testing.T) {
		closed := activePeriod()
		closed.Status = domain.PeriodClosed
		api.periods.EXPECT().GetByID(mock.Anything, periodID).Return(closed, nil).Once()

		assertError(t, api.do(http.MethodPost, "/api/v1/admin/periods/"+periodID+"/close", "", asAdmin()), http.StatusConflict, "period_closed")
	})

	t.Run("cerrar un período inexistente", func(t *testing.T) {
		api.periods.EXPECT().GetByID(mock.Anything, "no-existe").Return(domain.AcademicPeriod{}, domain.ErrPeriodNotFound).Once()

		assertError(t, api.do(http.MethodPost, "/api/v1/admin/periods/no-existe/close", "", asAdmin()), http.StatusNotFound, "period_not_found")
	})
}

func TestPeriodsReadableByProfessors(t *testing.T) {
	api := newTestAPI(t)

	t.Run("el profesor lista los períodos", func(t *testing.T) {
		api.activeSession(profLaptop, domain.RoleProfessor)
		api.periods.EXPECT().List(mock.Anything).Return([]domain.AcademicPeriod{activePeriod()}, nil).Once()

		rec := api.do(http.MethodGet, "/api/v1/periods", "", api.tokenFor(t, profLaptop))

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		body := decode[dto.PeriodListResponse](t, rec)
		require.Len(t, body.Items, 1)
		assert.Equal(t, "2026-10", body.Items[0].Code)
	})

	t.Run("el profesor consulta un período", func(t *testing.T) {
		api.activeSession(profLaptop, domain.RoleProfessor)
		api.periods.EXPECT().GetByID(mock.Anything, periodID).Return(activePeriod(), nil).Once()

		rec := api.do(http.MethodGet, "/api/v1/periods/"+periodID, "", api.tokenFor(t, profLaptop))

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Equal(t, periodID, decode[dto.PeriodResponse](t, rec).ID)
	})

	t.Run("período inexistente", func(t *testing.T) {
		api.activeSession(profLaptop, domain.RoleProfessor)
		api.periods.EXPECT().GetByID(mock.Anything, "no-existe").Return(domain.AcademicPeriod{}, domain.ErrPeriodNotFound).Once()

		assertError(t, api.do(http.MethodGet, "/api/v1/periods/no-existe", "", api.tokenFor(t, profLaptop)), http.StatusNotFound, "period_not_found")
	})

	t.Run("un monitor no consulta períodos", func(t *testing.T) {
		api.activeSession(anaLaptop, domain.RoleMonitor)

		assertError(t, api.do(http.MethodGet, "/api/v1/periods", "", api.tokenFor(t, anaLaptop)), http.StatusForbidden, "forbidden")
	})

	t.Run("sin token", func(t *testing.T) {
		assertError(t, api.do(http.MethodGet, "/api/v1/periods", "", ""), http.StatusUnauthorized, "unauthenticated")
	})
}

// RF-03.1 a RF-03.7: cursos y proyectos del profesor.
func TestWorkspaces(t *testing.T) {
	api := newTestAPI(t)
	asProfessor := func() string {
		api.activeSession(profLaptop, domain.RoleProfessor)
		return api.tokenFor(t, profLaptop)
	}
	asOtherProfessor := func() string {
		api.activeSession(otherProf, domain.RoleProfessor)
		return api.tokenFor(t, otherProf)
	}
	asAdmin := func() string {
		api.activeSession(adminLaptop, domain.RoleAdmin)
		return api.tokenFor(t, adminLaptop)
	}
	const body = `{"type":"CURSO","name":"Arquitectura de Software","period_id":"` + periodID +
		`","start_date":"2026-02-02","end_date":"2026-06-12","observations":"Grupo nocturno"}`

	t.Run("el profesor crea un curso y queda como responsable", func(t *testing.T) {
		api.periods.EXPECT().GetByID(mock.Anything, periodID).Return(activePeriod(), nil).Once()
		api.workspaces.EXPECT().Create(mock.Anything, mock.MatchedBy(func(w domain.Workspace) bool {
			return w.ProfessorID == profID && w.Status == domain.WorkspaceActive && w.Type == domain.WorkspaceCourse
		})).RunAndReturn(func(_ context.Context, w domain.Workspace) (domain.Workspace, error) {
			w.ID = workspaceID
			return w, nil
		}).Once()

		rec := api.do(http.MethodPost, "/api/v1/workspaces", body, asProfessor())

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		assert.Equal(t, "/api/v1/workspaces/"+workspaceID, rec.Header().Get("Location"))
		response := decode[dto.WorkspaceResponse](t, rec)
		assert.Equal(t, profID, response.ProfessorID)
		assert.Equal(t, "ACTIVO", response.Status)
		assert.Equal(t, "2026-02-02", response.StartDate)
		assert.Nil(t, response.ClosedAt)
	})

	t.Run("fuera de la ventana de creación del período", func(t *testing.T) {
		closed := activePeriod()
		closed.Status = domain.PeriodClosed
		api.periods.EXPECT().GetByID(mock.Anything, periodID).Return(closed, nil).Once()

		assertError(t, api.do(http.MethodPost, "/api/v1/workspaces", body, asProfessor()), http.StatusConflict, "creation_window_closed")
	})

	t.Run("el período debe existir", func(t *testing.T) {
		api.periods.EXPECT().GetByID(mock.Anything, periodID).Return(domain.AcademicPeriod{}, domain.ErrPeriodNotFound).Once()

		response := assertError(t, api.do(http.MethodPost, "/api/v1/workspaces", body, asProfessor()), http.StatusUnprocessableEntity, "validation_failed")
		assert.Contains(t, response.Error.Details, "period_id")
	})

	t.Run("datos inválidos con detalle por campo", func(t *testing.T) {
		rec := api.do(http.MethodPost, "/api/v1/workspaces", `{"type":"SEMILLERO","name":" "}`, asProfessor())

		response := assertError(t, rec, http.StatusUnprocessableEntity, "validation_failed")
		for _, field := range []string{"type", "name", "period_id", "start_date", "end_date"} {
			assert.Contains(t, response.Error.Details, field)
		}
	})

	t.Run("solo los profesores crean espacios", func(t *testing.T) {
		assertError(t, api.do(http.MethodPost, "/api/v1/workspaces", body, asAdmin()), http.StatusForbidden, "forbidden")
	})

	t.Run("el profesor lista únicamente sus espacios", func(t *testing.T) {
		api.workspaces.EXPECT().List(mock.Anything, ports.WorkspaceFilter{
			ProfessorID: profID, PeriodID: periodID, Status: domain.WorkspaceActive, Limit: 10,
		}).Return(ports.WorkspacePage{Workspaces: []domain.Workspace{courseOf(profID, domain.WorkspaceActive)}, Total: 1}, nil).Once()

		rec := api.do(http.MethodGet, "/api/v1/workspaces?period_id="+periodID+"&status=activo&limit=10", "", asProfessor())

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		page := decode[dto.WorkspaceListResponse](t, rec)
		assert.Equal(t, 1, page.Total)
		assert.Equal(t, 10, page.Limit)
		require.Len(t, page.Items, 1)
		assert.Equal(t, workspaceID, page.Items[0].ID)
	})

	t.Run("el administrador lista todos los espacios", func(t *testing.T) {
		api.workspaces.EXPECT().List(mock.Anything, ports.WorkspaceFilter{Limit: 50}).Return(ports.WorkspacePage{}, nil).Once()

		assert.Equal(t, http.StatusOK, api.do(http.MethodGet, "/api/v1/workspaces", "", asAdmin()).Code)
	})

	t.Run("parámetros de listado inválidos", func(t *testing.T) {
		assertError(t, api.do(http.MethodGet, "/api/v1/workspaces?limit=abc", "", asProfessor()), http.StatusBadRequest, "invalid_request")
		assertError(t, api.do(http.MethodGet, "/api/v1/workspaces?status=VIEJO", "", asProfessor()), http.StatusUnprocessableEntity, "validation_failed")
	})

	t.Run("un monitor no lista espacios", func(t *testing.T) {
		api.activeSession(anaLaptop, domain.RoleMonitor)

		assertError(t, api.do(http.MethodGet, "/api/v1/workspaces", "", api.tokenFor(t, anaLaptop)), http.StatusForbidden, "forbidden")
	})

	t.Run("el profesor consulta su espacio", func(t *testing.T) {
		api.workspaces.EXPECT().GetByID(mock.Anything, workspaceID).Return(courseOf(profID, domain.WorkspaceActive), nil).Once()

		rec := api.do(http.MethodGet, "/api/v1/workspaces/"+workspaceID, "", asProfessor())

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Equal(t, "Arquitectura de Software", decode[dto.WorkspaceResponse](t, rec).Name)
	})

	t.Run("un profesor no ve los espacios de otro", func(t *testing.T) {
		api.workspaces.EXPECT().GetByID(mock.Anything, workspaceID).Return(courseOf(profID, domain.WorkspaceActive), nil).Once()

		assertError(t, api.do(http.MethodGet, "/api/v1/workspaces/"+workspaceID, "", asOtherProfessor()), http.StatusNotFound, "workspace_not_found")
	})

	t.Run("el administrador ve cualquier espacio", func(t *testing.T) {
		api.workspaces.EXPECT().GetByID(mock.Anything, workspaceID).Return(courseOf(profID, domain.WorkspaceActive), nil).Once()

		assert.Equal(t, http.StatusOK, api.do(http.MethodGet, "/api/v1/workspaces/"+workspaceID, "", asAdmin()).Code)
	})

	t.Run("un espacio cerrado se consulta en modo de solo lectura", func(t *testing.T) {
		api.workspaces.EXPECT().GetByID(mock.Anything, workspaceID).Return(courseOf(profID, domain.WorkspaceClosed), nil).Once()

		rec := api.do(http.MethodGet, "/api/v1/workspaces/"+workspaceID, "", asProfessor())

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		response := decode[dto.WorkspaceResponse](t, rec)
		assert.Equal(t, "CERRADO", response.Status)
		assert.NotNil(t, response.ClosedAt)
	})

	t.Run("espacio inexistente", func(t *testing.T) {
		api.workspaces.EXPECT().GetByID(mock.Anything, "no-existe").Return(domain.Workspace{}, domain.ErrWorkspaceNotFound).Once()

		assertError(t, api.do(http.MethodGet, "/api/v1/workspaces/no-existe", "", asProfessor()), http.StatusNotFound, "workspace_not_found")
	})

	t.Run("el profesor responsable cierra su espacio", func(t *testing.T) {
		api.workspaces.EXPECT().GetByID(mock.Anything, workspaceID).Return(courseOf(profID, domain.WorkspaceActive), nil).Once()
		api.workspaces.EXPECT().MarkClosed(mock.Anything, workspaceID, testNow).Return(nil).Once()

		rec := api.do(http.MethodPost, "/api/v1/workspaces/"+workspaceID+"/close", "", asProfessor())

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		response := decode[dto.WorkspaceResponse](t, rec)
		assert.Equal(t, "CERRADO", response.Status)
		require.NotNil(t, response.ClosedAt)
		assert.True(t, response.ClosedAt.Equal(testNow))
	})

	t.Run("un espacio cerrado no se cierra otra vez", func(t *testing.T) {
		api.workspaces.EXPECT().GetByID(mock.Anything, workspaceID).Return(courseOf(profID, domain.WorkspaceClosed), nil).Once()

		assertError(t, api.do(http.MethodPost, "/api/v1/workspaces/"+workspaceID+"/close", "", asProfessor()), http.StatusConflict, "workspace_closed")
	})

	t.Run("otro profesor no cierra el espacio", func(t *testing.T) {
		api.workspaces.EXPECT().GetByID(mock.Anything, workspaceID).Return(courseOf(profID, domain.WorkspaceActive), nil).Once()

		assertError(t, api.do(http.MethodPost, "/api/v1/workspaces/"+workspaceID+"/close", "", asOtherProfessor()), http.StatusNotFound, "workspace_not_found")
	})

	t.Run("el administrador no cierra espacios", func(t *testing.T) {
		assertError(t, api.do(http.MethodPost, "/api/v1/workspaces/"+workspaceID+"/close", "", asAdmin()), http.StatusForbidden, "forbidden")
	})
}
