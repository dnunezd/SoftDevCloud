// Package domain contiene el núcleo del sistema: entidades, value objects y
// reglas de negocio.
//
// Restricción de arquitectura: este paquete solo depende de la biblioteca
// estándar. No importa Gin, pgx, Asynq ni SDKs externos. Las reglas se expresan
// como funciones o métodos puros para poder probarlas de forma unitaria.
package domain
