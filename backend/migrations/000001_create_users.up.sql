-- RF-01 y RF-02: usuarios con credenciales locales y roles globales (RN-01).
-- El esquema crece de forma incremental: cada módulo agrega su propia migración.

CREATE TABLE users (
    id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    full_name     text        NOT NULL CHECK (length(btrim(full_name)) > 0),
    email         text        NOT NULL CHECK (email = lower(email)),
    password_hash text        NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_email_key UNIQUE (email)
);

CREATE TABLE user_roles (
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role    text NOT NULL CHECK (role IN ('ADMIN', 'PROFESOR', 'MONITOR', 'ASISTENTE_GRADUADO')),
    PRIMARY KEY (user_id, role)
);

CREATE INDEX user_roles_role_idx ON user_roles (role);
