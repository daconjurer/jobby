#!/usr/bin/env bash
set -euo pipefail

# Waits for MongoDB and initiates the single-node replica set required for change streams.
# Exits only once a member reports PRIMARY (migrate and change streams need a writable primary).

mongosh_cmd() {
  mongosh --host mongodb:27017 \
    -u jobby_admin -p jobby_admin_pass \
    --authenticationDatabase admin \
    --quiet \
    "$@"
}

replica_has_primary() {
  mongosh_cmd --eval '
    try {
      const s = rs.status();
      if (s.ok !== 1) quit(1);
      if (s.members.some((m) => m.stateStr === "PRIMARY")) quit(0);
      quit(1);
    } catch (e) {
      quit(1);
    }
  '
}

init_replica_set() {
  mongosh_cmd --eval '
    try {
      const s = rs.status();
      if (s.ok === 1) quit(0);
    } catch (e) {}
    rs.initiate({_id: "rs0", members: [{_id: 0, host: "mongodb:27017"}]});
    quit(0);
  '
}

if replica_has_primary; then
  exit 0
fi

init_replica_set

until replica_has_primary; do
  sleep 1
done
