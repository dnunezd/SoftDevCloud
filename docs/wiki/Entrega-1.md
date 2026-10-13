# Entrega 1 · Backend REST con procesamiento asíncrono

> **Grupo XX** · _Nombre completo 1 · Nombre completo 2 · Nombre completo 3 · Nombre completo 4_
>
> Tag: `entrega-1` · Video: _enlace pendiente_

## Alcance

Según `07-alcance.md`: backend construido conforme a la especificación, `Dockerfile` y `docker-compose.yml` para la ejecución local, pipeline de GitHub Actions para validar las pruebas, documento de arquitectura y documento de deuda técnica. Quedan fuera la detección de ausencia de reporte, las notificaciones internas, la revisión o aprobación de tareas, el reinicio de contraseñas, la desactivación de usuarios y el frontend.

## Entregables

**Tabla 1.** Entregables mínimos de la Entrega 1 y su ubicación.

| Entregable | Ubicación | Estado |
|---|---|---|
| Backend según la especificación | `backend/` | En construcción (ver Tabla 2) |
| `Dockerfile` | `backend/Dockerfile` | Propuesto |
| `docker-compose.yml` | raíz del repositorio | Propuesto |
| Pipeline de GitHub Actions (pruebas + SonarQube) | `.github/workflows/ci.yml` | Propuesto |
| Documento de arquitectura | [Entrega-1-Arquitectura](Entrega-1-Arquitectura) | Propuesta en revisión |
| Documento de deuda técnica | [Entrega-1-Deuda-Tecnica](Entrega-1-Deuda-Tecnica) | Plantilla con deudas conocidas |
| Video de sustentación (10-20 min) | _enlace no listado_ | Pendiente |

## Estado de implementación

**Tabla 2.** Avance por requerimiento funcional.

| Requerimiento | Estado | Endpoints | Pruebas |
|---|---|---|---|
| RF-01 Autenticación y acceso | Implementado, con sesiones por dispositivo (resuelve DT-03) | `POST /auth/login`, `POST /auth/logout`, `GET /me`, `GET/DELETE /admin/users/{id}/sessions`, middleware de autenticación y RBAC | Dominio, caso de uso, JWT/bcrypt, API e integración con PostgreSQL |
| RF-02 Gestión de usuarios | Implementado | `POST/GET /admin/users`, `GET /admin/users/{id}`, `PUT /admin/users/{id}/roles` | Dominio, caso de uso, API e integración con PostgreSQL |
| RF-03 a RF-08, RF-10 a RF-12, RF-14, RF-15 | Pendiente | — | — |
| RF-09, RF-13 | Fuera del alcance de E1 | — | — |

## Evidencia del despliegue funcional

_Pendiente. Incluir, con una breve explicación de cada una:_

- _salida de `docker compose ps` con todos los servicios en estado `healthy` o `running`;_
- _respuesta de `GET /healthz`;_
- _un recorrido funcional: login → crear período y espacio → vincular → registrar tarea → generar lote de reportes → descargar PDF;_
- _ejecución verde del pipeline de CI y el panel de SonarQube._

## Guion mínimo del video

Elementos que exige la sección 3 de los lineamientos y cómo se cubren en esta entrega:

1. Identificación del grupo, sus integrantes y la entrega.
2. Descripción general de la solución.
3. Arquitectura: capas, componentes, modelo de datos y flujo asíncrono. Al ser la primera entrega no hay cambios frente a una iteración anterior; conviene decirlo.
4. Demostración funcional de los endpoints principales y de las reglas de negocio: un rechazo por carga horaria, un reporte tardío y una restricción de visibilidad.
5. Evidencia del despliegue local con Docker Compose y del pipeline de CI.
6. Pruebas de carga: **no se exigen en la Entrega 1**. Decirlo explícitamente y presentar cómo la arquitectura se preparó para medirlas (arquitectura, sección 13).
7. Resultados y análisis: cobertura de pruebas, hallazgos de SonarQube y deuda técnica.

## Cambios respecto a la entrega anterior

No aplica: la Entrega 1 es la línea base del proyecto.

## Lista de verificación antes de entregar

Basada en la sección 15 de los lineamientos:

- [ ] El repositorio corresponde al proyecto completo del grupo.
- [ ] El profesor y los tutores tienen acceso al repositorio y a la Wiki.
- [ ] La entrega está identificada con el tag `entrega-1` (y su *release*).
- [ ] La Wiki contiene la documentación completa y actualizada de la iteración.
- [ ] El número del grupo y los nombres completos de los integrantes aparecen en el README, en la Wiki, en los documentos y en el video.
- [ ] El enlace al video funciona y tiene permisos de acceso adecuados.
- [ ] Pruebas de carga: no aplican en E1, y se indica explícitamente.
- [ ] La arquitectura y el despliegue están documentados.
- [ ] No hay credenciales, secretos ni configuraciones sensibles en el repositorio (solo `.env.example`).
- [ ] Están presentes los entregables de `07-alcance.md` (Tabla 1).
- [ ] El envío se marcó formalmente en Coursera.
