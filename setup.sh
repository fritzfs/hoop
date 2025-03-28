#!/bin/bash
# setup file
# - fritz, Mar 27 2025

username="franjo.stipanovic"
hoop_path=/Users/franjo.stipanovic/.hoop/
hoop_dev_url=https://hoop.tradelocker.dev
hoop_pro_url=https://hoop.tradelocker.pro
start_ip=2
end_ip=10
interface="lo0"

if [[ $EUID -ne 0 ]]; then
  echo "This script must be run as root (sudo)." >&2
  exit 1
fi

echo ""
echo "Creating loopback aliases from 127.0.0.2 to 127.0.0.10 ..."
echo ""

for i in $(seq $start_ip $end_ip); do
  ip="127.0.0.$i"
  if ifconfig "$interface" | grep -q "$ip"; then
    echo "IP $ip already exists, skipping..."
  else
    sudo ifconfig "$interface" alias "$ip" up
    echo "Added $ip to $interface"
  fi
done

echo ""
echo "Creating databases files in /tmp/hop/dev and /tmp/hoop/pro ..."
echo ""

rm -rf /tmp/hoop/dev
sudo -u $username $hoop_path/bin/hoop config create --api-url $hoop_dev_url
sudo -u $username $hoop_path/bin/hoop login
mkdir -p /tmp/hoop/dev/
sudo -u $username $hoop_path/bin/hoop admin get connections -o json | jq '.[] .name' | tr -d '"' | while IFS= read -r line; do
  touch "/tmp/hoop/dev/$(echo "$line" | tr -d '/:*?"<>|')"
done
sudo -u $username $hoop_path/bin/hoop config view --raw | grep token | cut -f2 -d= >$hoop_path/.token.dev

rm -rf /tmp/hoop/pro
sudo -u $username $hoop_path/bin/hoop config create --api-url $hoop_pro_url
sudo -u $username $hoop_path/bin/hoop login
mkdir -p /tmp/hoop/pro/
sudo -u $username $hoop_path/bin/hoop admin get connections -o json | jq '.[] .name' | tr -d '"' | while IFS= read -r line; do
  touch "/tmp/hoop/pro/$(echo "$line" | tr -d '/:*?"<>|')"
done
sudo -u $username $hoop_path/bin/hoop config view --raw | grep token | cut -f2 -d= >$hoop_path/.token.pro
