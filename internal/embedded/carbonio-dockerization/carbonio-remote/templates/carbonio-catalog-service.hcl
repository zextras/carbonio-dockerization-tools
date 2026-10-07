services {
  name = "carbonio-test"
  port = 8000

  connect {
    sidecar_service {
      proxy {
        upstreams {
          destination_name = "carbonio-mailbox"
          local_bind_port  = 20000
        }
      }
    }
  }
}
