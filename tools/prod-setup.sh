#!/bin/bash

# Get the current user's home directory
USER_HOME=$(eval echo ~${SUDO_USER:-$USER})
OLYMPSIS_DIR="/etc/olympsis/nginx"

# 1. Create /etc/olympsis if it doesn't exist
if [ ! -d "/etc/olympsis" ]; then
    echo "Creating /etc/olympsis..."
    sudo mkdir -p /etc/olympsis
fi

# 2. Create /etc/olympsis/nginx if it doesn't exist
if [ ! -d "$OLYMPSIS_DIR" ]; then
    echo "Creating $OLYMPSIS_DIR..."
    sudo mkdir -p "$OLYMPSIS_DIR"
fi

# 3. Create symbolic link for server.conf
SERVER_CONF_SRC="$USER_HOME/server/nginx/server.conf"
SERVER_CONF_LINK="$OLYMPSIS_DIR/server.conf"

if [ ! -L "$SERVER_CONF_LINK" ]; then
    echo "Linking server.conf..."
    sudo ln -s "$SERVER_CONF_SRC" "$SERVER_CONF_LINK"
else
    echo "server.conf link already exists."
fi

# 4. Create symbolic link for proxy_headers.conf
PROXY_HEADERS_SRC="$USER_HOME/server/nginx/proxy_headers.conf"
PROXY_HEADERS_LINK="$OLYMPSIS_DIR/proxy_headers.conf"

if [ ! -L "$PROXY_HEADERS_LINK" ]; then
    echo "Linking proxy_headers.conf..."
    sudo ln -s "$PROXY_HEADERS_SRC" "$PROXY_HEADERS_LINK"
else
    echo "proxy_headers.conf link already exists."
fi

echo "Done."