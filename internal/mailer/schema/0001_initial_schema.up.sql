CREATE TABLE recipients (
  id INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT,
  email VARCHAR(254) NOT NULL,
  status VARCHAR(32) NOT NULL,
  last_modified TIMESTAMP NULL
);