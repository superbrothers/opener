#!/usr/bin/env bash

if [ -p /dev/stdin ]; then
    cat - | nc -U "$HOME/.opener.sock"
else
    echo "${@}" | nc -U "$HOME/.opener.sock"
fi
