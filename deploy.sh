#!/bin/bash
set -e

BINARY=simplechat
SERVER=mhc
REMOTE_DIR=/opt/simplechat

echo "== Building for linux amd64..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $BINARY .

echo "== Copying files..."
scp $BINARY index.html Dockerfile simplechat.minhhoccode111.com .env.prod $SERVER:/tmp/

echo "== Deploying..."
ssh $SERVER "
sudo mkdir -p $REMOTE_DIR
sudo mv /tmp/$BINARY $REMOTE_DIR/
sudo mv /tmp/index.html $REMOTE_DIR/
sudo mv /tmp/.env.prod $REMOTE_DIR/.env

docker stop $BINARY 2>/dev/null || true
docker rm $BINARY 2>/dev/null || true
docker build -t $BINARY:latest -f /tmp/Dockerfile $REMOTE_DIR
docker run -d --name $BINARY --network=host --restart=always --env-file $REMOTE_DIR/.env $BINARY:latest

if ! diff /tmp/simplechat.minhhoccode111.com /etc/caddy/snippets/simplechat.minhhoccode111.com 2>/dev/null; then
    sudo cp /tmp/simplechat.minhhoccode111.com /etc/caddy/snippets/simplechat.minhhoccode111.com
    sudo systemctl reload caddy
fi
rm -f /tmp/simplechat.minhhoccode111.com /tmp/Dockerfile
"

echo "== Done."
