ALTER TABLE list_recipients MODIFY last_modified TIMESTAMP NULL;

UPDATE list_recipients SET last_modified = NULL;
