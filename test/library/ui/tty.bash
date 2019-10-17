#!/usr/bin/env bash

if [ -t 1 ]; then 
  IS_TTY=true
else
  IS_TTY=false
fi
export IS_TTY