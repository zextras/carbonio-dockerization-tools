#!/bin/bash

#!/bin/bash

jq -s '{components: .}' $(find /opt/zextras/web/iris/ -name component.json) >/opt/zextras/web/iris/components.json
jq -s '{components: .}' $(find /opt/zextras/admin/iris/ -name component.json) >/opt/zextras/admin/iris/components.json

./entrypoint.sh