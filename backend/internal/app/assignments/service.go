// Package assignments implementa la vinculación de personas a cursos y
// proyectos (RF-04, RF-05) y su rectificación por el administrador (RF-15.3).
//
// La validación de carga horaria se ejecuta dentro de una transacción con
// bloqueo de la persona, para evitar que dos vinculaciones concurrentes
// superen los límites.
package assignments
