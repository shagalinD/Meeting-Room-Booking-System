#!/bin/sh
set -eu

cert_dir=/etc/nginx/certs
cert_file="$cert_dir/server.crt"
key_file="$cert_dir/server.key"

mkdir -p "$cert_dir"
if [ ! -s "$cert_file" ] || [ ! -s "$key_file" ]; then
    openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
        -keyout "$key_file" \
        -out "$cert_file" \
        -subj "/CN=localhost" \
        -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
    chmod 600 "$key_file"
fi

exec "$@"
