-- A user's self-added word exists once per headword.
CREATE UNIQUE INDEX idx_senses_owner_headword_unique ON senses(owner_user_id, headword)
  WHERE owner_user_id IS NOT NULL AND deleted_at IS NULL;
