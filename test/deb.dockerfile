FROM debian:stable

RUN apt-get update \
    && apt-get upgrade -y \
    && apt-get dist-upgrade -y \
    && apt-get install -y sudo gnupg apt-transport-https ca-certificates

RUN sudo apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 379CE192D401AB61 \
    && echo "deb https://dl.bintray.com/dopplerhq/doppler-deb stable main" | sudo tee -a /etc/apt/sources.list \
    && sudo apt-get update \
    && sudo apt-get install -y doppler

ENTRYPOINT ["doppler"]
