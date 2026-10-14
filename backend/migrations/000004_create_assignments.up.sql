-- RF-04, RF-05 y RN-10 a RN-19: vinculaciones y su registro de rectificaciones.
-- El profesor responsable no se duplica aquí: se obtiene de workspaces.professor_id (RN-11).

CREATE TABLE assignments (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      uuid        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    workspace_id uuid        NOT NULL REFERENCES workspaces (id) ON DELETE RESTRICT,
    role         text        NOT NULL CHECK (role IN ('MONITOR', 'ASISTENTE_GRADUADO')),
    weekly_hours integer     NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    -- Una persona tiene como máximo una vinculación por curso o proyecto.
    CONSTRAINT assignments_user_workspace_key UNIQUE (user_id, workspace_id),
    -- Tope por rol de cada vinculación (RN-14, RN-16). Los límites que suman
    -- varias vinculaciones los valida el dominio dentro de una transacción.
    CONSTRAINT assignments_weekly_hours_check CHECK (
        weekly_hours > 0
        AND ((role = 'ASISTENTE_GRADUADO' AND weekly_hours <= 22) OR (role = 'MONITOR' AND weekly_hours <= 12))
    )
);

CREATE INDEX assignments_workspace_id_idx ON assignments (workspace_id);

-- Antes y después de cada rectificación del administrador (RF-04.5).
CREATE TABLE assignment_audit (
    id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    assignment_id uuid        NOT NULL REFERENCES assignments (id) ON DELETE RESTRICT,
    admin_id      uuid        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    before        jsonb       NOT NULL,
    after         jsonb       NOT NULL,
    changed_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX assignment_audit_assignment_id_idx ON assignment_audit (assignment_id);
