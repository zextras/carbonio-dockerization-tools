keygen:
	keytool -genkey -keyalg RSA -dname "cn=Unknown, ou=Unknown, o=Unknown, c=Unknown" \
   	-keystore ssl/keystore -keysize 2048 -validity 3650 \
   	-keypass testkey1 -storepass teststore1
export-cert:
	keytool -exportcert \
      -alias mykey \
      -keystore ssl/keystore \
      -storepass teststore1 \
      -rfc \
      -file ssl/server-cert.pem
certgen:
	 openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 \
    -nodes -keyout ssl/nginx.key \
    -out ssl/nginx.crt -subj "/CN=carbonio.localhost" \
    -addext "subjectAltName=DNS:carbonio.localhost,DNS:*.carbonio.localhost,IP:10.0.0.1"
start-advanced:
	docker compose -f docker-compose.yaml -f docker-compose-advanced.yaml -f docker-compose.macos.override.yaml \
	-f monitoring.yaml \
	-f docker-compose-advanced.macos.override.yaml up -d
stop-advanced:
	docker compose -f docker-compose.yaml -f docker-compose-advanced.yaml -f docker-compose.macos.override.yaml \
	-f monitoring.yaml \
	-f docker-compose-advanced.macos.override.yaml down
delete-advanced:
	docker compose -f docker-compose.yaml -f docker-compose-advanced.yaml -f docker-compose.macos.override.yaml \
	-f monitoring.yaml \
	-f docker-compose-advanced.macos.override.yaml down -v

start-cluster:
	docker compose -f docker-compose.yaml -f docker-compose-cluster.yaml -f docker-compose.macos.override.yaml \
 	-f monitoring.yaml -f monitoring-cluster.yaml \
	-f docker-compose-cluster.macos.override.yaml up -d
stop-cluster:
	docker compose -f docker-compose.yaml -f docker-compose-cluster.yaml -f docker-compose.macos.override.yaml \
	-f monitoring.yaml -f monitoring-cluster.yaml \
	-f docker-compose-cluster.macos.override.yaml down
delete-cluster:
	docker compose -f docker-compose.yaml -f docker-compose-cluster.yaml -f docker-compose.macos.override.yaml \
	-f monitoring.yaml -f monitoring-cluster.yaml \
	-f docker-compose-cluster.macos.override.yaml down -v

start-advanced-cluster:
	docker compose -f docker-compose.yaml -f docker-compose-cluster.yaml -f docker-compose-advanced.yaml \
	-f docker-compose-cluster.yaml -f docker-compose-advanced-cluster.yaml \
	-f docker-compose.macos.override.yaml \
	-f docker-compose-advanced.macos.override.yaml -f docker-compose-cluster.macos.override.yaml up -d
stop-advanced-cluster:
	docker compose -f docker-compose.yaml -f docker-compose-cluster.yaml -f docker-compose-advanced.yaml \
	-f docker-compose-cluster.yaml -f docker-compose-advanced-cluster.yaml \
	-f docker-compose.macos.override.yaml \
	-f docker-compose-advanced.macos.override.yaml -f docker-compose-cluster.macos.override.yaml down
delete-advanced-cluster:
	docker compose -f docker-compose.yaml -f docker-compose-cluster.yaml -f docker-compose-advanced.yaml \
	-f docker-compose-mailreplica.yaml \
	-f docker-compose-cluster.yaml -f docker-compose-advanced-cluster.yaml \
	-f docker-compose.macos.override.yaml \
	-f docker-compose-advanced.macos.override.yaml -f docker-compose-cluster.macos.override.yaml down -v

start-mailreplica:
	docker compose -f docker-compose.yaml -f docker-compose-cluster.yaml -f docker-compose-advanced.yaml \
	-f docker-compose-advanced-cluster.yaml \
	-f docker-compose-mailreplica.yaml \
	-f docker-compose.macos.override.yaml \
	-f docker-compose-advanced.macos.override.yaml -f docker-compose-cluster.macos.override.yaml up -d
stop-mailreplica:
	docker compose -f docker-compose.yaml -f docker-compose-cluster.yaml -f docker-compose-advanced.yaml \
	-f docker-compose-advanced-cluster.yaml \
	-f docker-compose-mailreplica.yaml \
	-f docker-compose.macos.override.yaml \
	-f docker-compose-advanced.macos.override.yaml -f docker-compose-cluster.macos.override.yaml down
delete-mailreplica:
	docker compose -f docker-compose.yaml -f docker-compose-cluster.yaml -f docker-compose-advanced.yaml \
	-f docker-compose-advanced-cluster.yaml \
	-f docker-compose-mailreplica.yaml \
	-f docker-compose.macos.override.yaml \
	-f docker-compose-advanced.macos.override.yaml -f docker-compose-cluster.macos.override.yaml down -v

pull:
	docker compose -f docker-compose.yaml -f docker-compose-cluster.yaml -f docker-compose-advanced.yaml \
    	-f docker-compose-mailreplica.yaml \-f docker-compose.macos.override.yaml \
    	-f docker-compose-advanced.macos.override.yaml -f docker-compose-cluster.macos.override.yaml pull

stop:
	docker rm -f $(docker ps -aq) || true

remove-volumes:
	docker volume rm $(docker volume ls -q) || true
