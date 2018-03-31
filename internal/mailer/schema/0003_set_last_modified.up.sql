UPDATE list_recipients SET last_modified = NOW();

ALTER TABLE list_recipients MODIFY last_modified TIMESTAMP NOT NULL;
