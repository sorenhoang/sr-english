CREATE EXTENSION IF NOT EXISTS "uuid-ossp" SCHEMA public;

CREATE OR REPLACE FUNCTION update_updated_at() RETURNS trigger AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  auth_sub TEXT NOT NULL,
  email TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ,
  deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX idx_users_auth_sub_unique ON users(auth_sub) WHERE deleted_at IS NULL;

CREATE TABLE user_settings (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL,
  level TEXT NOT NULL,
  daily_new INTEGER NOT NULL,
  timezone TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ,
  deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX idx_user_settings_user_id_unique ON user_settings(user_id) WHERE deleted_at IS NULL;

-- One row per (headword, part of speech). owner_user_id is set only for words a user added themselves.
CREATE TABLE senses (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  headword TEXT NOT NULL,
  pos TEXT NOT NULL,
  cefr_level TEXT,
  variants TEXT[] NOT NULL DEFAULT '{}',
  source TEXT NOT NULL,
  owner_user_id UUID,
  ipa_uk TEXT,
  ipa_us TEXT,
  vi_meaning TEXT,
  en_definition TEXT,
  example_en TEXT,
  example_vi TEXT,
  reviewed BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ,
  deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX idx_senses_headword_pos_unique ON senses(headword, pos) WHERE owner_user_id IS NULL AND deleted_at IS NULL;
CREATE INDEX idx_senses_level_active ON senses(cefr_level) WHERE deleted_at IS NULL;
CREATE INDEX idx_senses_owner_user_id ON senses(owner_user_id) WHERE owner_user_id IS NOT NULL;

CREATE TABLE cards (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL,
  sense_id UUID NOT NULL,
  due TIMESTAMPTZ NOT NULL,
  stability DOUBLE PRECISION NOT NULL DEFAULT 0,
  difficulty DOUBLE PRECISION NOT NULL DEFAULT 0,
  elapsed_days INTEGER NOT NULL DEFAULT 0,
  scheduled_days INTEGER NOT NULL DEFAULT 0,
  reps INTEGER NOT NULL DEFAULT 0,
  lapses INTEGER NOT NULL DEFAULT 0,
  state SMALLINT NOT NULL DEFAULT 0,
  last_review TIMESTAMPTZ,
  context_sentence TEXT,
  context_passage_id UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ,
  deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX idx_cards_user_id_sense_id_unique ON cards(user_id, sense_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_cards_user_id_due_active ON cards(user_id, due) WHERE deleted_at IS NULL;
CREATE INDEX idx_cards_sense_id ON cards(sense_id);

CREATE TABLE reviews (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL,
  card_id UUID NOT NULL,
  sense_id UUID NOT NULL,
  rating SMALLINT NOT NULL,
  is_new BOOLEAN NOT NULL,
  reviewed_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ,
  deleted_at TIMESTAMPTZ
);
CREATE INDEX idx_reviews_user_id_reviewed_at_active ON reviews(user_id, reviewed_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_reviews_card_id ON reviews(card_id);

CREATE TRIGGER trg_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER trg_user_settings_updated_at BEFORE UPDATE ON user_settings FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER trg_senses_updated_at BEFORE UPDATE ON senses FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER trg_cards_updated_at BEFORE UPDATE ON cards FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER trg_reviews_updated_at BEFORE UPDATE ON reviews FOR EACH ROW EXECUTE FUNCTION update_updated_at();
