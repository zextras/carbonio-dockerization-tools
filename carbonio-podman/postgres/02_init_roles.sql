-- Roles only. Databases are created by each service's sidecar bootstrap script.
CREATE ROLE carbonio_advanced WITH LOGIN SUPERUSER encrypted password 'password';

CREATE ROLE "carbonio-files-adm" WITH LOGIN SUPERUSER encrypted password 'password';
CREATE DATABASE "carbonio-files-adm" owner "carbonio-files-adm";

CREATE ROLE "carbonio-mailbox-adm" WITH LOGIN SUPERUSER encrypted password 'password';
CREATE DATABASE "carbonio-mailbox-adm" owner "carbonio-mailbox-adm";

CREATE ROLE "carbonio-docs-connector-adm" WITH LOGIN SUPERUSER encrypted password 'password';
CREATE DATABASE "carbonio-docs-connector-adm" owner "carbonio-docs-connector-adm";