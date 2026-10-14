-- RF-14 y RN-41 a RN-46: lotes de reportes semanales y un reporte por persona.

CREATE TABLE report_batches (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    professor_id uuid        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    week_start   date        NOT NULL CHECK (EXTRACT(ISODOW FROM week_start) = 1),
    -- Se recalcula a partir del estado de los reportes del lote.
    status       text        NOT NULL DEFAULT 'EN_PROCESO' CHECK (status IN ('EN_PROCESO', 'COMPLETADO', 'COMPLETADO_CON_ERRORES')),
    requested_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX report_batches_professor_week_idx ON report_batches (professor_id, week_start);

CREATE TABLE weekly_reports (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id     uuid        NOT NULL REFERENCES report_batches (id) ON DELETE RESTRICT,
    user_id      uuid        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    -- Copia de la semana del lote; la igualdad entre ambas la garantiza el dominio.
    week_start   date        NOT NULL CHECK (EXTRACT(ISODOW FROM week_start) = 1),
    status       text        NOT NULL DEFAULT 'PENDIENTE' CHECK (status IN ('PENDIENTE', 'PROCESANDO', 'COMPLETADO', 'FALLIDO')),
    storage_key  text,
    error        text,
    generated_at timestamptz,
    -- Un reporte por persona en cada lote (RF-14.2). Este índice también cubre las consultas por batch_id.
    CONSTRAINT weekly_reports_batch_user_key UNIQUE (batch_id, user_id),
    -- Un reporte completado siempre tiene su PDF almacenado (RF-14.5).
    CONSTRAINT weekly_reports_completed_check CHECK (status <> 'COMPLETADO' OR storage_key IS NOT NULL)
);

CREATE INDEX weekly_reports_user_id_idx ON weekly_reports (user_id);
