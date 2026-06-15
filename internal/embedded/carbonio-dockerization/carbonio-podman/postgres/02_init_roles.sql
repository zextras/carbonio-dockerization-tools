-- Roles only. Databases are created by each service's sidecar bootstrap script.


-- Advanced
CREATE ROLE carbonio_advanced WITH LOGIN SUPERUSER encrypted password 'password';
CREATE ROLE "carbonio-mailbox-adm" WITH LOGIN SUPERUSER encrypted password 'password';
CREATE DATABASE "carbonio-mailbox-adm" owner "carbonio-mailbox-adm";

-- WSC
CREATE ROLE "carbonio_adm" WITH LOGIN SUPERUSER encrypted password 'password';
CREATE DATABASE "carbonio_adm" owner "carbonio_adm";

-- Files
CREATE ROLE "carbonio-files-adm" WITH LOGIN SUPERUSER encrypted password 'password';
CREATE DATABASE "carbonio-files-adm" owner "carbonio-files-adm";

-- Docs connector
CREATE ROLE "carbonio-docs-connector-adm" WITH LOGIN SUPERUSER encrypted password 'password';
CREATE DATABASE "carbonio-docs-connector-adm" owner "carbonio-docs-connector-adm";