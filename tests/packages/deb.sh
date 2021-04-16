#!/bin/bash

set -e

apt-get update > /dev/null 2>&1
apt-get install -y apt-transport-https ca-certificates curl gnupg sudo > /dev/null 2>&1
curl -sLf --tlsv1.2 'https://packages.doppler.com/public/cli/gpg.DE2A7741A397C129.key' | sudo apt-key add -
echo "deb https://packages.doppler.com/public/cli/deb/debian any-version main" | sudo tee /etc/apt/sources.list.d/dopplerhq-doppler.list
apt-get update > /dev/null 2>&1
apt-get install -y doppler
doppler -v
