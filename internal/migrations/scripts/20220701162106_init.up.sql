CREATE TABLE IF NOT EXISTS users
(
    id         VARCHAR(100) PRIMARY KEY,
    first_name VARCHAR(255) NOT NULL,
    last_name  VARCHAR(255) NOT NULL,
    username   varchar(255) NOT NULL,
    language   varchar(3)   NOT NULL,
    created_at timestamp    NOT NULL,
    updated_at timestamp    NOT NULL
);
