#!/bin/bash

podman exec -i carbonio-mailbox-app zmprov <<'EOF'
  ca shared@carbonio.localhost assext displayName "Shared Dept"
  ca shared2@carbonio.localhost assext displayName "Shared Dept 2"
  ca delegated@carbonio.localhost assext
  ca shared@test.com assext displayName "Shared Dept"
  ca shared2@test.com assext displayName "Shared Dept 2"
  ca delegated@test.com assext
EOF

podman exec -i carbonio-mailbox-app zmmailbox -z <<'EOF'
  sm shared@carbonio.localhost
  mfg / account delegated@carbonio.localhost rwidx
  sm delegated@carbonio.localhost
  cm /shared@carbonio.localhost shared2@carbonio.localhost /
  sm shared2@carbonio.localhost
  mfg / account delegated@carbonio.localhost rwidx
  sm delegated@carbonio.localhost
  cm /shared2@carbonio.localhost shared2@carbonio.localhost /
EOF