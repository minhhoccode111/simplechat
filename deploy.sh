#!/bin/bash
set -e

BINARY=simplechat
SERVER=mhc
REMOTE_DIR=/opt/simplechat

echo "== Building for linux amd64..."
GOOS=linux GOARCH=amd64 go build -o $BINARY .

echo "== Copying files..."
scp $BINARY index.html simplechat.nginx.conf simplechat.service .env.prod $SERVER:/tmp/

echo "== Deploying..."
ssh $SERVER "
sudo mkdir -p $REMOTE_DIR
sudo mv /tmp/$BINARY /usr/local/bin/
sudo mv /tmp/index.html $REMOTE_DIR/
sudo mv /tmp/.env.prod $REMOTE_DIR/.env
sudo mv /tmp/simplechat.nginx.conf /etc/nginx/sites-available/simplechat
sudo ln -sf /etc/nginx/sites-available/simplechat /etc/nginx/sites-enabled/
sudo mv /tmp/simplechat.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable simplechat
sudo systemctl restart simplechat
sudo systemctl reload nginx
"

echo "== Done."
