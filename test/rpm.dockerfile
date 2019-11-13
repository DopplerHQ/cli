FROM centos:latest

RUN yum update -y && yum install -y wget sudo

RUN sudo wget https://bintray.com/dopplerhq/doppler-rpm/rpm -O bintray-dopplerhq-doppler-rpm.repo && mv bintray-dopplerhq-doppler-rpm.repo /etc/yum.repos.d/ \
    && sudo yum update -y \
    && sudo yum install -y doppler

ENTRYPOINT ["doppler"]
