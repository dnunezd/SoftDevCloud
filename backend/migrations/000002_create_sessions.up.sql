-- Sesiones (DT-03): cada inicio de sesión crea una fila y su ID viaja en el token
-- (claim sid). Cerrar sesión borra solo esa fila; las demás sesiones del usuario
-- (otros dispositivos) siguen activas. Vive en la BD compartida, así que todas
-- las instancias ven lo mismo (requisito de E3).
CREATE TABLE sessions (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    user_agent text        NOT NULL DEFAULT '',
    ip         text        NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);

CREATE INDEX sessions_user_id_idx ON sessions (user_id);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);
