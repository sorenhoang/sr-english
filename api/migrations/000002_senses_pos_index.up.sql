-- Serves the per-part-of-speech distractor sample in /session/today.
CREATE INDEX idx_senses_pos_public_active ON senses(pos)
  WHERE deleted_at IS NULL AND owner_user_id IS NULL AND vi_meaning IS NOT NULL;
