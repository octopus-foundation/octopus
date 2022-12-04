#!/bin/bash

CMD="$@"

if [[ "$1" == "ansible-playbook" ]]; then
  if [[ -f /mnt/global_ssh_keys/id_rsa ]]; then
    echo "default ssh key found, using it"
    KEY_FILE="/mnt/global_ssh_keys/id_rsa"
  else
    echo "default ssh key not found, using private key"
    KEY_FILE="/mnt/local_ssh_keys/id_rsa"
  fi
  CMD="$CMD --key-file $KEY_FILE"
fi

echo $CMD
exec $CMD