## Web UI
The frontend consists of a single webserver but multiple projects.  
So the composed ui is based on a multistage build copying compiled files from
static images inside a nginx image and hosting them: this approach means that
all frontend projects can upload images in the registry and everyone can compose their
ui with different versions of each project without having to build locally/install npm&deps etc.

## Nginx routing
This is the official Carbonio Nginx, so routing is based on NsLookup for the 
mailbox, and other services are managed with upstreams.  
In this case however  Nginx is integrated with consul-template and Consul, 
so upstreams are dynamically resolved based on service registration on 
Consul (service discovery).  
In addition some services that are not present  in the official Carbonio 
installation were added (user-management, preview) and several servers for 
specific service routing were added.