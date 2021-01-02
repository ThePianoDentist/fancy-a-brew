
-- extension needs to be added as admin (i.e. postgres user)
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE TABLE appusers(
    user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- why not just use firebase_token as primary-key if that's unique?
    -- well it's a long-string, rather than a space (and join) efficient UUID
    firebase_token TEXT UNIQUE NOT NULL,
    default_nickname TEXT NOT NULL,
    the_usual TEXT NOT NULL,
    -- current_making_kettle  // this is cool. cos when boot can go auto straight to kettle-page....IF close geolocation. otherwise list/add.
    -- but this wouldnt be kettle-id user drink-responds to when opening app, that could be a different kettle.
    -- open_app_with_kettle......it seems better to just have a `drink_round` table, and we look for users newest offer
    last_known_location geography(POINT,4326)
);

CREATE TABLE kettles(
    kettle_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wireless_id TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    current_maker UUID REFERENCES appusers,
    location geography(POINT,4326) NOT NULL
);

CREATE INDEX users_location_gix ON appusers USING GIST (last_known_location);
CREATE INDEX kettles_location_gix ON kettles USING GIST (location);
CREATE INDEX kettles_appusers ON kettles(current_maker);