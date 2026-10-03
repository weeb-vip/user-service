-- Follow graph and the outbox that announces changes to it.
--
-- user_follows is one directed edge per row. status is 'pending' while the
-- followee still has to approve (they set follow_approval_required) and
-- 'accepted' once they have, or immediately when approval is not required.
-- The primary key makes a pair unique in one direction; following back is a
-- second row the other way round.
--
-- Both indexes lead with the side a page looks up by: followee for "who
-- follows me" and for fan-out, follower for "who do I follow". status is next
-- so a pending-requests page and an accepted-followers page each read one
-- index range.
--
-- outbox_events is the transactional outbox from go-outbox-lib. Its shape is
-- the library's outbox.Schema, copied here so this migration chain can be
-- verified without the library. The partial index is the relay's only read.
ALTER TABLE users ADD COLUMN IF NOT EXISTS follow_approval_required boolean NOT NULL DEFAULT false;

CREATE TABLE IF NOT EXISTS user_follows (
    follower_id varchar(100) NOT NULL REFERENCES users(id),
    followee_id varchar(100) NOT NULL REFERENCES users(id),
    status      varchar(16)  NOT NULL,
    created_at  timestamptz  NOT NULL DEFAULT now(),
    accepted_at timestamptz,
    PRIMARY KEY (follower_id, followee_id),
    CONSTRAINT user_follows_not_self CHECK (follower_id <> followee_id),
    CONSTRAINT user_follows_status CHECK (status IN ('pending', 'accepted'))
);
CREATE INDEX IF NOT EXISTS idx_user_follows_followee ON user_follows (followee_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_follows_follower ON user_follows (follower_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS outbox_events (
    id           uuid PRIMARY KEY,
    subject      text        NOT NULL,
    payload      jsonb       NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz
);
CREATE INDEX IF NOT EXISTS idx_outbox_events_unpublished ON outbox_events (created_at, id) WHERE published_at IS NULL;
