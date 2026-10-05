By querying "dig @127.0.0.1 -p 8600 carbonio-mailbox.service.consul" you get 
the IP address of mailbox in the local datacenter.

So if a container uses Consul as DNS it can resolve the mailbox ip using 
Consul DNS.

This method does not have mTLS, but allows loadbalancing between healthy 
services.
This load balancing strategy is also discussed here: https://www.hashicorp.com/en/blog/load-balancing-strategies-for-consul