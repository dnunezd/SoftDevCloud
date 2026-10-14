-- RF-03 y RN-06 a RN-09: períodos académicos y espacios (cursos y proyectos).
-- Las restricciones son la segunda línea de defensa: las reglas de negocio viven
-- en el dominio y la BD evita datos corruptos si hay un error de programación.

CREATE TABLE academic_periods (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code              text NOT NULL CHECK (length(btrim(code)) > 0),
    start_date        date NOT NULL,
    end_date          date NOT NULL,
    creation_deadline date NOT NULL,
    status            text NOT NULL DEFAULT 'ACTIVO' CHECK (status IN ('ACTIVO', 'CERRADO')),
    CONSTRAINT academic_periods_code_key UNIQUE (code),
    -- La ventana de creación de espacios cae dentro del período (RN-07).
    CONSTRAINT academic_periods_dates_check CHECK (start_date <= creation_deadline AND creation_deadline <= end_date)
);

CREATE TABLE workspaces (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    type         text        NOT NULL CHECK (type IN ('CURSO', 'PROYECTO')),
    name         text        NOT NULL CHECK (length(btrim(name)) > 0),
    period_id    uuid        NOT NULL REFERENCES academic_periods (id) ON DELETE RESTRICT,
    professor_id uuid        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    start_date   date        NOT NULL,
    end_date     date        NOT NULL,
    observations text        NOT NULL DEFAULT '',
    status       text        NOT NULL DEFAULT 'ACTIVO' CHECK (status IN ('ACTIVO', 'CERRADO')),
    closed_at    timestamptz,
    CONSTRAINT workspaces_dates_check CHECK (start_date <= end_date),
    -- Estado y fecha de cierre no pueden contradecirse (RN-08).
    CONSTRAINT workspaces_closed_check CHECK ((status = 'CERRADO') = (closed_at IS NOT NULL))
);

CREATE INDEX workspaces_professor_id_idx ON workspaces (professor_id);
CREATE INDEX workspaces_period_id_idx ON workspaces (period_id);
