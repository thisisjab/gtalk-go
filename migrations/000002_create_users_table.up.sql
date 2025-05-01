CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
    version BIGINT NOT NULL DEFAULT 1,
    email TEXT NOT NULL UNIQUE,
    email_verified_at TIMESTAMPTZ,
    display_name TEXT NOT NULL,
    bio TEXT
);
