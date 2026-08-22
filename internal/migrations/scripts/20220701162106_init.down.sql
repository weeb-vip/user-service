-- Was "DROP TABLE IF EXISTS credentials;", which is auth's table, not this
-- service's. Almost certainly copied from auth, which has a migration with this
-- exact timestamp creating credentials.
--
-- Not a harmless mistake: both services point at the same database, so running
-- this dropped auth's credentials -- every stored login -- and left users, the
-- table this migration actually creates, in place. It stayed invisible because
-- down migrations are almost never run.
DROP TABLE IF EXISTS users;
