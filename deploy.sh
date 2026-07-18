#!/bin/bash
set -e

BINARY=simplechat
SERVER=mhc
REMOTE_DIR=/opt/simplechat

echo "== Building for linux amd64..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $BINARY .

echo "== Copying files..."
scp $BINARY index.html simplechat.service simplechat.minhhoccode111.com .env.prod $SERVER:/tmp/

echo "== Deploying..."
ssh $SERVER "
sudo mkdir -p $REMOTE_DIR
sudo mv /tmp/$BINARY $REMOTE_DIR/
sudo mv /tmp/index.html $REMOTE_DIR/
sudo mv /tmp/.env.prod $REMOTE_DIR/.env

sudo mv /tmp/simplechat.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable simplechat
sudo systemctl restart simplechat

if ! diff /tmp/simplechat.minhhoccode111.com /etc/caddy/snippets/simplechat.minhhoccode111.com 2>/dev/null; then
    sudo cp /tmp/simplechat.minhhoccode111.com /etc/caddy/snippets/simplechat.minhhoccode111.com
    sudo systemctl reload caddy
fi
rm -f /tmp/simplechat.minhhoccode111.com
"

echo "== Done."
