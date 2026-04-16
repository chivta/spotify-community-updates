CREATE TABLE users (
    id BIGINT PRIMARY KEY
);

CREATE TABLE updates (
    id SERIAL PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL
);

---- create above / drop below ----

DROP TABLE users;