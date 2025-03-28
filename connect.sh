#!/bin/bash
# connect
# - fritz, Mar 27 2025

username="franjo.stipanovic"
hoop_path=/Users/franjo.stipanovic/.hoop/
hoop_dev_url=https://hoop.tradelocker.dev
hoop_dev_grpcurl=https://hoopgrpc.tradelocker.dev:443 
hoop_pro_url=https://hoop.tradelocker.pro
hoop_pro_grpcurl=https://hoopgrpc.tradelocker.pro:443 
start_ip=2
end_ip=10

if [[ $EUID -ne 0 ]]; then
    echo "This script must be run as root (sudo)." >&2
    exit 1
fi

if [[ -z "$1" ]]; then
    echo "Usage: $0 /tmp/hoop/<ENV>/<DATABASE>"
    exit 1
fi

if [[ "$1" == /tmp/hoop/dev/* ]]; then
    hoop_grpcurl=$hoop_dev_grpcurl
    hoop_apiurl=$hoop_dev_url
    hoop_token=$(cat $hoop_path/.token.dev)
elif [[ "$1" == /tmp/hoop/pro/* ]]; then
    hoop_grpcurl=$hoop_pro_grpcurl
    hoop_apiurl=$hoop_pro_url
    hoop_token=$(cat $hoop_path/.token.pro)
else
    echo "Error: Invalid path. Must be in /tmp/hoop/dev/ or /tmp/hoop/pro/" >&2
    exit 1
fi

db=$(basename "$1")
hostname=$(basename "$1").local

echo ""
echo "Creating hostname: $hostname"
echo ""

# Remove existing hostname entries from /etc/hosts
sed -i '' "/[[:space:]]$hostname$/d" /etc/hosts

used_ips=$(awk '/^127\.0\.0\./ {print $1}' /etc/hosts)

# Find the first available IP
for i in $(seq $start_ip $end_ip); do
    candidate_ip="127.0.0.$i"
    if ! grep -q "^$candidate_ip" <<< "$used_ips"; then
        available_ip="$candidate_ip"
        break
    fi
done

# If no available IP found, exit
if [[ -z "$available_ip" ]]; then
    echo "No available IPs in the range" >&2
    exit 1
fi

echo "$available_ip $hostname" | tee -a /etc/hosts

# Flush DNS
dscacheutil -flushcache

# Test if hoop session is valid, if not, login
sudo -u $username HOOP_GRPCURL="$hoop_grpcurl" HOOP_APIURL="$hoop_apiurl" HOOP_TOKEN="$hoop_token" $hoop_path/bin/hoop exec $db -i "select 1" > /dev/null 2>&1
if [[ $? -ne 0 ]]; then
    echo "Hoop session not found, initiating ..."
    sudo -u $username HOOP_GRPCURL="$hoop_grpcurl" HOOP_APIURL="$hoop_apiurl" HOOP_TOKEN="$hoop_token" $hoop_path/bin/hoop login
fi

echo ""
echo -e "\033[32mAfter connect, use $hostname to connect!\033[0m"
echo ""

# hoop connect
sudo -u $username HOOP_GRPCURL="$hoop_grpcurl" HOOP_APIURL="$hoop_apiurl" HOOP_TOKEN="$hoop_token" $hoop_path/bin/hoop connect -a $available_ip $db