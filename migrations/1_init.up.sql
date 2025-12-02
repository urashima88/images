CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS images (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id UUID NOT NULL,
    image_id TEXT NOT NULL CHECK (image_id <> ''),
    width INTEGER NOT NULL CHECK (width > 0),
    height INTEGER NOT NULL CHECK (height > 0),
    extension VARCHAR NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    score INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS images_profile_id_idx ON images (profile_id);
CREATE INDEX IF NOT EXISTS images_image_id_idx ON images (image_id);
CREATE INDEX IF NOT EXISTS images_created_at_idx ON images (created_at);
CREATE INDEX IF NOT EXISTS images_score_idx ON images (score);

CREATE TABLE IF NOT EXISTS tags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL UNIQUE CHECK (name <> ''),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX tags_name_idx ON tags (name);
CREATE UNIQUE INDEX tags_name_unique ON tags (name);

CREATE TABLE image_tags (
    image_id UUID NOT NULL,
    tag_id UUID NOT NULL,

    PRIMARY KEY (image_id, tag_id),

    CONSTRAINT fk_image_tags_images FOREIGN KEY (image_id) REFERENCES images(id) ON DELETE CASCADE,
    CONSTRAINT fk_image_tags_tags FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS image_tags_image_id_idx ON image_tags (image_id);
CREATE INDEX IF NOT EXISTS image_tags_tag_id_idx ON image_tags (tag_id);
