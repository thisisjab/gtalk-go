CREATE TABLE IF NOT EXISTS conversations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    name TEXT,
    type TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
    version INTEGER NOT NULL DEFAULT 1,
    -- Conversation can be of type 'private' or 'group'
    CHECK (type IN ('private', 'group')),
    -- Private conversation has no name
    CHECK (
        name IS NULL
        OR type = 'group'
    )
);

CREATE TABLE IF NOT EXISTS conversation_participants (
    conversation_id UUID NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
    -- Check each user_id & conversation_id is unique
    CONSTRAINT unique_participant UNIQUE (conversation_id, user_id)
);

CREATE TABLE IF NOT EXISTS conversation_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    conversation_id UUID NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    sender_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    type TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
    version INTEGER NOT NULL DEFAULT 1,
    -- Message can be of type 'text', 'image', 'video', 'audio', 'file'
    CHECK (
        type IN ('text', 'image', 'video', 'audio', 'file')
    )
);
