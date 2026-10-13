package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Health atiende GET /healthz: comprueba solo el propio servicio, sin consultar
// dependencias. Responde 200 mientras el proceso esté vivo y atendiendo
// peticiones. Lo usan Docker y, desde E3, el balanceador de GCP.
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
