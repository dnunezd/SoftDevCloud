// Package handlers contiene un handler por módulo. Cada handler solo traduce
// HTTP <-> caso de uso (lectura de la petición, invocación y respuesta). La
// lógica de negocio y las validaciones de contenido no viven aquí: los errores
// se registran con c.Error y el middleware de errores los traduce.
package handlers

import "github.com/gin-gonic/gin"

// fail registra el error para que el middleware de errores escriba la respuesta.
func fail(c *gin.Context, err error) {
	_ = c.Error(err)
}
