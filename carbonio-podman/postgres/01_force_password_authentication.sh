#!/bin/bash
sed -i 's/trust/scram-sha-256/g' "$PGDATA/pg_hba.conf"
pg_ctl reload -D "$PGDATA"