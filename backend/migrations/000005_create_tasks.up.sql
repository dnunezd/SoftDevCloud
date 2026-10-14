-- RF-06 a RF-08 y RN-20 a RN-34: tareas semanales y sus adjuntos.
-- El estado "sin reporte" no se almacena: se deriva de la ausencia de tareas (RN-36).

CREATE TABLE tasks (
    id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    assignment_id uuid        NOT NULL REFERENCES assignments (id) ON DELETE RESTRICT,
    -- La semana se identifica por la fecha de su lunes (RN-29).
    week_start    date        NOT NULL CHECK (EXTRACT(ISODOW FROM week_start) = 1),
    title         text        NOT NULL CHECK (length(btrim(title)) > 0),
    description   text        NOT NULL DEFAULT '',
    status        text        NOT NULL CHECK (status IN ('ABIERTO', 'EN_DESARROLLO', 'FINALIZADO')),
    hours_spent   integer     NOT NULL CHECK (hours_spent >= 0),
    observations  text        NOT NULL DEFAULT '',
    -- Se fija al crear (semana ya cerrada) y no se recalcula (RN-32).
    is_late       boolean     NOT NULL DEFAULT false,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX tasks_assignment_week_idx ON tasks (assignment_id, week_start);

-- Solo metadatos: el contenido vive detrás de FileStorage y aquí queda su clave (RN-23).
CREATE TABLE task_attachments (
    id           uuid   PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id      uuid   NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    file_name    text   NOT NULL,
    content_type text   NOT NULL,
    size_bytes   bigint NOT NULL CHECK (size_bytes >= 0),
    storage_key  text   NOT NULL
);

CREATE INDEX task_attachments_task_id_idx ON task_attachments (task_id);
