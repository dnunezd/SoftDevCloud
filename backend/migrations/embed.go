// Package migrations embebe los scripts SQL versionados para que el binario los
// aplique al arrancar (golang-migrate). Cada módulo agrega su propia migración
// a medida que se implementa.
package migrations

import "embed"

// FS contiene los archivos *.sql de este directorio.
//
//go:embed *.sql
var FS embed.FS
