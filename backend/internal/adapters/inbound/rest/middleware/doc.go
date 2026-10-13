// Package middleware contiene los middleware HTTP transversales.
//
//   - logging.go: identificador de petición y log de acceso estructurado.
//   - auth.go: valida el token y carga la identidad en el contexto.
//   - rbac.go: exige roles por grupo de rutas.
//   - errors.go: traduce errores de dominio a respuestas HTTP uniformes.
package middleware
