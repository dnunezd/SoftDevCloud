# Entrega 1 · Reporte de deuda técnica

> **Grupo XX** · _Nombre completo 1 · Nombre completo 2 · Nombre completo 3 · Nombre completo 4_
>
> Versión asociada: tag `entrega-1`.

Este reporte sigue la sección 8 de los lineamientos. Registra las decisiones, limitaciones y compromisos **del código implementado** que afectan la calidad o la evolución de la solución, por qué existen, qué riesgo introducen y cómo se propone tratarlos. Al cierre de la entrega se completa con los hallazgos de SonarQube.

Solo se registran deudas de componentes que ya existen. Los compromisos de diseño de módulos aún no implementados (almacenamiento de archivos, cola de reportes) se describen en la [arquitectura](Entrega-1-Arquitectura), sección 15, y se registrarán aquí cuando esos módulos se construyan.

## 1. Campos de cada registro

**Tabla 1.** Campos obligatorios de cada deuda técnica.

| Campo | Contenido esperado |
|---|---|
| ID | Identificador estable entre entregas (DT-NN) |
| Descripción | Problema concreto: qué es y dónde está |
| Componente afectado | Módulo, paquete, servicio o parte de la arquitectura |
| Origen | Restricción de tiempo, simplificación deliberada, decisión provisional, limitación de despliegue, problema heredado |
| Relación | ADR, prueba de carga, hallazgo de SonarQube o dificultad de despliegue con la que se relaciona |
| Impacto técnico | Mantenibilidad, rendimiento, escalabilidad, confiabilidad, seguridad, pruebas, operación o comprensión |
| Prioridad | Alta / Media / Baja |
| Riesgo si no se atiende | Consecuencia concreta |
| Acción propuesta | Mitigación, reducción o resolución, y en qué entrega |
| Estado | Identificada / En tratamiento / Parcialmente resuelta / Resuelta |

## 2. Deudas registradas

**Tabla 2.** Registro de deudas técnicas de la Entrega 1.

| ID | Descripción | Componente | Origen | Relación | Impacto | Prioridad | Riesgo si no se atiende | Acción propuesta | Estado |
|---|---|---|---|---|---|---|---|---|---|
| DT-03 | Los tokens de acceso no se podían revocar antes de vencer: no había cierre de sesión ni forma de invalidar un token robado. En una primera versión, además, los roles viajaban en el token y un rol retirado seguía vigente hasta 8 h | `adapters/outbound/security/token.go`, `middleware/auth.go` | Simplificación para mantener la API sin estado (requisito de E3) | ADR-08 | Seguridad | Media | Un token filtrado permitía actuar como el usuario hasta su vencimiento | **Implementado.** Tabla `sessions`: cada inicio de sesión crea una fila y su ID viaja en el token (`sid`). En cada petición se verifica que la sesión siga activa y se leen los roles vigentes, en una sola consulta. `POST /auth/logout` cierra solo la sesión del dispositivo; `DELETE /admin/users/{id}/sessions` cierra todas las de un usuario. Las sesiones viven en la BD compartida, así que se sigue cumpliendo E3 | **Resuelta** |
| DT-05 | El administrador inicial se crea al arrancar, desde las variables `BOOTSTRAP_ADMIN_*` | `cmd/api`, `bootstrap/container.go` | Decisión provisional: solo un administrador puede crear usuarios, así que sin uno inicial nadie podría entrar al sistema | — | Seguridad / operación | Baja | Si no se cambia la contraseña de ejemplo de `.env.example`, queda una credencial por defecto conocida | Forzar el cambio de contraseña en el primer inicio de sesión, o crear el administrador con un comando de *seed* separado del arranque | Identificada |
| DT-06 | `POST /auth/login` no limita los intentos fallidos. Además, un correo inexistente responde más rápido que una contraseña incorrecta (no ejecuta bcrypt), lo que permite inferir por tiempo qué correos existen | `app/auth`, `adapters/inbound/rest` | Fuera del alcance de E1 (RNF-03 pide un mecanismo "básico") | ADR-08 | Seguridad | Media | Ataques de fuerza bruta o enumeración de cuentas | Comparar contra un hash ficticio cuando el correo no existe. Límite de intentos por cuenta y por IP (Nginx `limit_req` o middleware). **Ojo con E2:** un límite por IP afectaría el escenario obligatorio de "autenticación concurrente", porque la herramienta de carga envía desde una sola IP; habrá que excluirla o medir sin el límite y documentarlo | Identificada |

**Identificadores retirados:** DT-01 (archivos en volumen local), DT-02 (Redis como punto único de fallo) y DT-04 (sin *dead-letter queue*) se retiraron porque describían módulos que aún no existen. Sus números no se reutilizan, para conservar la trazabilidad.

## 3. Hallazgos de SonarQube y su lectura técnica

_Pendiente al cierre de la entrega. Resumir bugs, vulnerabilidades, code smells, duplicación y cobertura. Para cada hallazgo relevante, explicar su impacto real en el proyecto y si se convierte en un registro de la Tabla 2. No basta con exportar las métricas (lineamientos, sección 8)._

## 4. Evolución entre iteraciones

En la Entrega 1:

- **Resueltas:** DT-03, dentro de la misma iteración en que se identificó.
- **Nuevas y abiertas:** DT-05 y DT-06.

A partir de la Entrega 2, esta sección distingue deudas nuevas, acumuladas y resueltas.

## 5. Priorización consolidada

**Tabla 3.** Priorización de las deudas abiertas.

| Prioridad | ID | Recomendación |
|---|---|---|
| Media | DT-06 | Antes de E2, corregir la diferencia de tiempos con el hash ficticio: es un cambio pequeño. El límite de intentos puede esperar a definir el plan de pruebas de carga, para no distorsionar el escenario de autenticación concurrente |
| Baja | DT-05 | Puede diferirse. Mientras tanto, el procedimiento de despliegue debe exigir una contraseña propia en `BOOTSTRAP_ADMIN_PASSWORD`, nunca la de ejemplo |
