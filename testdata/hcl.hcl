// a line comment
# another line comment
/* a block
   comment */

job "batch" {
  datacenters = ["dc1", "dc2"]
  type        = "batch"
  priority    = 50
  weight      = 1.5
  big         = 10mb
  enabled     = true
  disabled    = false
  nothing     = null

  meta {
    owner = "team"
    note  = 'single quoted'
  }

  group "workers" {
    count = 3

    task "run" {
      driver = "docker"

      config {
        image   = "alpine:3"
        command = "/bin/sh"
        args    = ["-c", "echo hi"]
      }

      resources {
        cpu    = 500
        memory = 256
      }

      template {
        data = <<EOF
a here document
with $dollars and ${braces}
EOF
      }

      quoted = <<-'END'
  indented and quoted
  END
    }
  }
}
