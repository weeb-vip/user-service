-- Postgres baseline for user-service.
--
-- Generated from the production schema dumped out of PlanetScale, not from
-- this repository's MySQL migrations. The migrations describe what was
-- intended; the dump describes what is actually there, and on anime-api those
-- two disagreed -- a table its migrations create had been dropped from
-- production years ago and nothing noticed.
--
-- Worth knowing while reading this: every service shares one MySQL database
-- today. These tables sit alongside anime-api's, with a separate
-- __migrations_* tracker per service. Whether that stays true in Postgres is
-- an open decision; this file only covers the tables this service owns.

CREATE TABLE users (
  id varchar(100) NOT NULL,
  first_name varchar(255) NOT NULL,
  last_name varchar(255) NOT NULL,
  username varchar(255) NOT NULL,
  language varchar(3) NOT NULL,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL,
  email varchar(255),
  profile_image_url varchar(500),
  PRIMARY KEY (id)
);


