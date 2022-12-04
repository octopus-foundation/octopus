FROM ubuntu:20.04

ENV DEBIAN_FRONTEND noninteractive
ENV LANG en_US.UTF-8
ENV ANSIBLE_HOST_KEY_CHECKING=False

RUN apt-get update && \
    apt-get -y install software-properties-common rsync && \
    add-apt-repository ppa:ansible/ansible && \
    apt-get update && \
    apt-get -y install ansible python3-venv python3-pip && mkdir -p /root/.ssh && \
    ansible-galaxy collection install amazon.aws && \
    pip install boto3 boto botocore

COPY docker-entrypoint.sh /docker-entrypoint.sh
ENTRYPOINT ["/docker-entrypoint.sh"]