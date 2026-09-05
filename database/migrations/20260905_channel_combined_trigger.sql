BEGIN;

ALTER TABLE channel_policies DROP CONSTRAINT IF EXISTS channel_policies_mode_check;
ALTER TABLE channel_policies ADD CONSTRAINT channel_policies_mode_check
    CHECK (mode IN ('disabled', 'silent_listen', 'mention_only', 'mention_or_keyword', 'keyword', 'auto_reply'));

COMMIT;
