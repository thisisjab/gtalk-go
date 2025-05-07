CREATE TYPE conversation_type AS ENUM ('private', 'group');

CREATE TABLE IF NOT EXISTS conversations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    type conversation_type NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
    version INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS group_metadata (
    conversation_id UUID PRIMARY KEY REFERENCES conversations (ID) ON DELETE CASCADE,
    name TEXT NOT NULL,
    owner_id UUID NOT NULL REFERENCES users (ID) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS conversation_participants (
    conversation_id UUID NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
    -- Check each user_id & conversation_id is unique
    CONSTRAINT unique_participant UNIQUE (conversation_id, user_id)
);

CREATE TYPE message_type AS ENUM ('text', 'image', 'video', 'audio', 'file');

CREATE TABLE IF NOT EXISTS conversation_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    conversation_id UUID NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    sender_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    type message_type NOT NULL,
    replied_message_id UUID REFERENCES conversation_messages (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
    version INTEGER NOT NULL DEFAULT 1
);
