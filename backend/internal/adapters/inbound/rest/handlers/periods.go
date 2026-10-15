package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/dnunezd/SoftDevCloud/backend/internal/adapters/inbound/rest/dto"
	"github.com/dnunezd/SoftDevCloud/backend/internal/adapters/inbound/rest/middleware"
	"github.com/dnunezd/SoftDevCloud/backend/internal/app/periods"
)

// Periods atiende los períodos académicos (RF-03.2): el administrador los crea
// y cierra; profesores y administrador los consultan.
type Periods struct {
	periods *periods.Service
}

// NewPeriods crea los handlers de períodos.
func NewPeriods(periodService *periods.Service) *Periods {
	return &Periods{periods: periodService}
}

// Create atiende POST /api/v1/admin/periods.
func (h *Periods) Create(c *gin.Context) {
	var req dto.CreatePeriodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, middleware.InvalidRequest(err))
		return
	}
	period, err := h.periods.Create(c.Request.Context(), req.ToInput())
	if err != nil {
		fail(c, err)
		return
	}
	c.Header("Location", "/api/v1/periods/"+period.ID)
	c.JSON(http.StatusCreated, dto.NewPeriodResponse(period))
}

// List atiende GET /api/v1/periods.
func (h *Periods) List(c *gin.Context) {
	list, err := h.periods.List(c.Request.Context())
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewPeriodListResponse(list))
}

// Get atiende GET /api/v1/periods/{id}.
func (h *Periods) Get(c *gin.Context) {
	period, err := h.periods.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewPeriodResponse(period))
}

// Close atiende POST /api/v1/admin/periods/{id}/close.
func (h *Periods) Close(c *gin.Context) {
	period, err := h.periods.Close(c.Request.Context(), c.Param("id"))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewPeriodResponse(period))
}
