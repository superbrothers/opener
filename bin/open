#!/bin/bash

set -eu

# get data either form stdin or from file
if (( $# == 0 )) ; then
  # if no argument, read from standard input from pipe
  buf=$(cat "$@")
else
  # otherwise read from all arguments
  buf=$@
fi

echo "$buf" | nc -U "$HOME/.opener.sock"
