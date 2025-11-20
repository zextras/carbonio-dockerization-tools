CREATE ROLE carbonio_advanced WITH LOGIN SUPERUSER encrypted password 'password';
CREATE DATABASE "core" OWNER carbonio_advanced;
CREATE DATABASE "activesync" OWNER carbonio_advanced;
CREATE DATABASE "powerstore" OWNER carbonio_advanced;
CREATE DATABASE "ha" OWNER carbonio_advanced;
CREATE DATABASE "abq" OWNER carbonio_advanced;
CREATE DATABASE "auth" OWNER carbonio_advanced;
CREATE DATABASE "backup" OWNER carbonio_advanced;