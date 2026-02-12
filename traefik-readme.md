It's possible to have a default router in traefik by doing:

traefik:
image: traefik:v3.6
command:
- "--providers.docker=true"
- "--providers.docker.defaultRule=Host(`{{ normalize .Name }}.docker.localhost`)"
- "--providers.docker.exposedbydefault=true"
- "--entrypoints.carbonio-files.address=:8082"
- "--entrypoints.carbonio-storages.address=:8083"


However you need to define services like:

carbonio-files:
image: myimage
expose:
- "10000"

so traefik uses the first exposed port.
The service will be available at http://carbonio-files.docker.localhost

This is good for single instances as there is no load balancing.
If you have another files instance then it will be available at 
http://carbonio-files-1.docker.localhost