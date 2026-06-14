#!/usr/bin/env bash
set -euo pipefail

# Waits for MongoDB and initiates the single-node replica set required for change streams.
until mongosh --host mongodb:27017 -u jobby_admin -p jobby_admin_pass --authenticationDatabase admin --quiet --eval '
  try { if (rs.status().ok === 1) { quit(0); } } catch (e) {}
  rs.initiate({_id: "rs0", members: [{_id: 0, host: "mongodb:27017"}]});
  quit(rs.status().ok === 1 ? 0 : 1);
'; do
  sleep 1
done
