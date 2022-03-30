CREATE DATABASE blog;

\c blog

CREATE TABLE IF NOT EXISTS post (
    id SERIAL PRIMARY KEY,
    title VARCHAR (500) NOT NULL,
    body TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tag (
    id SERIAL PRIMARY KEY,
    name VARCHAR (50) NOT NULL,
    post_id INT NOT NULL,
    FOREIGN KEY (post_id)
        REFERENCES post (id)
);