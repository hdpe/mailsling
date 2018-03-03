CREATE TABLE recipients (
  id INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT,
  email VARCHAR(254) NOT NULL,
  UNIQUE KEY uq_recipients_email (email)
);

CREATE TABLE list_recipients (
  id INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT,
  list_id VARCHAR(128) NULL,
  recipient_id INTEGER NOT NULL,
  status VARCHAR(32) NOT NULL,
  last_modified TIMESTAMP NULL,
  UNIQUE KEY uq_list_recipients_list_id_recipient_id (list_id, recipient_id),
  CONSTRAINT fk_list_recipients_recipient_id FOREIGN KEY (recipient_id) REFERENCES recipients (id)
);