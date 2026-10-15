# SoftDevCloud

Proyecto práctico de desarrollo de software en la nube: **seguimiento semanal de monitores y asistentes graduados**.

> **Grupo XX** · _Nombre completo 1 · Nombre completo 2 · Nombre completo 3 · Nombre completo 4_

Plataforma web ligera para que los profesores hagan seguimiento semanal del trabajo de monitores y asistentes graduados en cursos y proyectos, con generación de reportes PDF mediante IA generativa.

**Stack:** Go (Gin) · PostgreSQL · Asynq + Redis · Nginx · Docker Compose · GitHub Actions · SonarQube.

## Estado

**Entrega 1 en construcción.** Implementados: RF-01 (autenticación y control de acceso por rol), RF-02 (gestión de usuarios) y RF-03 (períodos académicos definidos por el administrador, y cursos y proyectos del profesor). Además, las reglas de dominio y el esquema de base de datos de vinculaciones, tareas y reportes (RF-04 a RF-08, RF-14) ya están implementados y probados, aunque aún sin casos de uso ni endpoints. El avance por requerimiento está en la [página de la Entrega 1](docs/wiki/Entrega-1.md).

## Documentación

La documentación oficial está en la **Wiki** del repositorio. Su fuente versionada es [`docs/wiki/`](docs/wiki) y se publica automáticamente al integrar cambios en `main` (`.github/workflows/wiki.yml`).

- [Inicio de la Wiki](docs/wiki/Home.md)
- [Entrega 1: alcance, entregables y lista de verificación](docs/wiki/Entrega-1.md)
- [Entrega 1: arquitectura](docs/wiki/Entrega-1-Arquitectura.md)
- [Entrega 1: deuda técnica](docs/wiki/Entrega-1-Deuda-Tecnica.md)

## Componentes

**Tabla 1.** Servicios del despliegue local.

| Servicio | Función | Puerto |
|---|---|---|
| `nginx` | Proxy inverso, única entrada pública | `8080` (host) |
| `api` | API REST en Go | `8080` (interno) |
| `worker` | Procesa la generación asíncrona de reportes | — |
| `db` | PostgreSQL 17 | `5432` (interno) |
| `redis` | Broker de la cola (Asynq) | `6379` (interno) |

## Ejecución local

**Requisitos:** Linux (o WSL2), Docker Engine con Docker Compose v2 y Git. Para ejecutar las pruebas fuera de los contenedores se necesita además Go 1.26.

```bash
cp .env.example .env
docker compose up -d --build
```

Antes de levantar el sistema, ajusta en `.env` las contraseñas y `JWT_SECRET`. El administrador inicial se crea con `BOOTSTRAP_ADMIN_EMAIL` y `BOOTSTRAP_ADMIN_PASSWORD`. La API queda disponible en `http://localhost:8080/api/v1` y su estado se verifica en `http://localhost:8080/healthz`.

### Probar la API

Inicio de sesión con el administrador inicial (los valores salen de `.env`):

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/login -H "Content-Type: application/json" -d '{"email":"admin@example.com","password":"cambiar-en-local"}'
```

Con el `access_token` de la respuesta, crear un usuario con varios roles:

```bash
curl -s -X POST http://localhost:8080/api/v1/admin/users -H "Authorization: Bearer <access_token>" -H "Content-Type: application/json" -d '{"full_name":"Ana Perez","email":"ana@uni.edu","password":"secreta123","roles":["MONITOR","ASISTENTE_GRADUADO"]}'
```

Para cerrar la sesión del dispositivo actual (las demás sesiones del usuario siguen activas):

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/logout -H "Authorization: Bearer <access_token>"
```

> En Windows, para enviar tildes o eñes desde la terminal, guarda el JSON en un archivo UTF-8 y envíalo con `--data-binary @archivo.json`. Si no, la terminal puede mandar otra codificación y el texto llega alterado.

### Contrato de la API y Postman

- **Contrato OpenAPI 3.0:** [`docs/api/openapi.yaml`](docs/api/openapi.yaml). Se puede importar en Postman, SwaggerHub o cualquier visor de OpenAPI.
- **Colección de Postman:** [`postman/seguimiento-monitores.postman_collection.json`](postman/seguimiento-monitores.postman_collection.json). Está organizada en 7 carpetas, cada petición tiene sus pruebas, y los inicios de sesión guardan los tokens en variables.
- **Entorno:** importa `postman/local.postman_environment.json` si existe en tu máquina (contiene credenciales y **no se versiona**). Si no, importa la plantilla `postman/example.postman_environment.json` y complétala con las cuentas de tu entorno local.

Ejecutar la colección completa desde la terminal con Newman (requiere el sistema levantado):

```bash
docker run --rm --network seguimiento-monitores_default -v "$PWD/postman":/etc/newman postman/newman:alpine run seguimiento-monitores.postman_collection.json -e local.postman_environment.json --env-var baseUrl=http://nginx
```

### Pruebas

```bash
make test
```

Pruebas de integración de los repositorios (requieren una base PostgreSQL **dedicada**, porque la prueba la vacía):

```bash
TEST_DATABASE_URL="postgres://usuario:clave@localhost:5432/seguimiento_test?sslmode=disable" make test-integration
```

Los mocks de `internal/mocks` se regeneran con `make mocks` cada vez que cambia una interfaz de `internal/ports`.

## Convenciones del repositorio

- `main` está protegida: los cambios entran por *pull request* con el pipeline de CI en verde.
- Ramas de trabajo: `feature/<modulo>-<descripcion>`, `fix/...`, `docs/...`.
- Cada entrega se marca con un tag y su *release*: `entrega-1`, `entrega-2`, `entrega-3`, `entrega-4`.
- Nunca se versionan secretos: solo `.env.example`, con valores de ejemplo.
- La Wiki no se edita a mano: los cambios se hacen en `docs/wiki/`.

## Uso de herramientas de apoyo y recursos externos

Declaración según la sección 12 de los lineamientos. **El equipo debe completarla y mantenerla al día.**

- La propuesta inicial de estructura del repositorio y de los documentos de arquitectura y deuda técnica se elaboró con apoyo de un asistente de IA (Claude Code). El equipo la revisó, la ajustó y asume su defensa técnica.
- Librerías de terceros previstas: Gin, pgx, golang-migrate, Asynq, golang-jwt, x/crypto (bcrypt), go-pdf/fpdf, el SDK de Anthropic para Go y testify. Herramienta de generación: mockery. Las versiones exactas quedan registradas en `backend/go.mod`.
