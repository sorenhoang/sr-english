CREATE TABLE passages (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  title TEXT NOT NULL,
  body TEXT NOT NULL,
  cefr_level TEXT NOT NULL,
  topics TEXT[] NOT NULL DEFAULT '{}',
  kinds TEXT[] NOT NULL DEFAULT '{}',
  source TEXT NOT NULL,
  source_url TEXT NOT NULL,
  license TEXT NOT NULL,
  audio_url TEXT,
  published_at TIMESTAMPTZ,
  tokens JSONB NOT NULL DEFAULT '[]',
  sentences JSONB NOT NULL DEFAULT '[]',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ,
  deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX idx_passages_source_url_unique ON passages(source_url) WHERE deleted_at IS NULL;
CREATE INDEX idx_passages_level_active ON passages(cefr_level) WHERE deleted_at IS NULL;
CREATE INDEX idx_passages_deleted ON passages(deleted_at) WHERE deleted_at IS NOT NULL;

CREATE TABLE passage_progress (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL,
  passage_id UUID NOT NULL,
  completed_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ,
  deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX idx_passage_progress_user_id_passage_id_unique ON passage_progress(user_id, passage_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_passage_progress_passage_id ON passage_progress(passage_id);

CREATE TRIGGER trg_passages_updated_at BEFORE UPDATE ON passages FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER trg_passage_progress_updated_at BEFORE UPDATE ON passage_progress FOR EACH ROW EXECUTE FUNCTION update_updated_at();
