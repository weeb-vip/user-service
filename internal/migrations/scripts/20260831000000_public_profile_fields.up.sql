-- Fields for a public user page and its customization.
--
-- banner_image_url is the wide header image, stored like profile_image_url: the
-- path to the object, not a full URL, so the CDN base can change without a
-- rewrite. bio is a short free-text line. accent_color themes the page and is a
-- token name rather than a raw colour, so a page cannot set something off the
-- palette. lists_public gates the one piece of the page that is not public by
-- default -- the header is always visible, the watch/read lists only when the
-- user opts in.
ALTER TABLE users ADD COLUMN IF NOT EXISTS banner_image_url varchar(512);
ALTER TABLE users ADD COLUMN IF NOT EXISTS bio varchar(300);
ALTER TABLE users ADD COLUMN IF NOT EXISTS accent_color varchar(32);
ALTER TABLE users ADD COLUMN IF NOT EXISTS lists_public boolean NOT NULL DEFAULT false;
