CREATE TABLE users (
  id INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT,
  email VARCHAR(254) NOT NULL,
  status VARCHAR(32) NOT NULL,
  welcome_time TIMESTAMP NULL
);

INSERT INTO users (email, status) VALUES ('ryan@mission-remission.com', 'new');
