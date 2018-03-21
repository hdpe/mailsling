CREATE TABLE list_recipient_attributes (
  list_recipient_id INTEGER NOT NULL,
  `key` VARCHAR(128) NOT NULL,
  `value` VARCHAR(4096) NOT NULL,
  PRIMARY KEY (list_recipient_id, `key`),
  CONSTRAINT fk_list_recipient_attributes_list_recipient_id FOREIGN KEY (list_recipient_id) REFERENCES list_recipients (id)
);