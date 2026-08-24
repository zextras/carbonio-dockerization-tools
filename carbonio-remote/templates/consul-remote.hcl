ui_config {
  enabled = true
}

datacenter = "dc1"
node_name  = "carbonio-catalog"
server     = false

client_addr    = "0.0.0.0"
bind_addr      = "0.0.0.0"
advertise_addr = "ADVERTISE_ADDR"
retry_join     = ["REMOTE_HOST"]
encrypt        = "POs+wTCCZjxT/GBKTxBocdlKPrCrsgKl8ox0RFQFQU8="

ports {
  grpc     = -1
  grpc_tls = 8502
}

acl {
  enabled        = true
  default_policy = "allow"
  tokens {
    agent = "CONSUL_TOKEN"
  }
}

tls {
  defaults {
    ca_file        = "/consul/tls/consul-agent-ca.pem"
    cert_file      = "/consul/tls/client.pem"
    key_file       = "/consul/tls/client-key.pem"
    verify_outgoing = true
  }
  internal_rpc {
    verify_server_hostname = false
  }
}
