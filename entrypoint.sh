#!/bin/sh
# entrypoint.sh

exec /bin/server daemon -p "$PORT" -i "$COMMUNITY_IP" -c "$CLUSTER_IP" -v "$CLUSTER_PORT" -j "$IPFS_IP" -b "$IPFS_PORT" -d "$DISCOVERY_ADDRESS" 