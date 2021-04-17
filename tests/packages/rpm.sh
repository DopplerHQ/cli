#!/bin/bash

set -e

rpm --import 'https://packages.doppler.com/public/cli/gpg.DE2A7741A397C129.key'
curl -sLf --tlsv1.2 --proto "=https" 'https://packages.doppler.com/public/cli/config.rpm.txt?distro=el' > /etc/yum.repos.d/bintray-dopplerhq-doppler.repo
yum install -y doppler
doppler -v
