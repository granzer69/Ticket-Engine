CREATE USER IF NOT EXISTS 'ticket'@'%' IDENTIFIED BY 'ticket';
GRANT ALL PRIVILEGES ON ticketdb.* TO 'ticket'@'%';
GRANT ALL PRIVILEGES ON ticketdb_integration.* TO 'ticket'@'%';
CREATE DATABASE IF NOT EXISTS ticketdb_integration;
FLUSH PRIVILEGES;
