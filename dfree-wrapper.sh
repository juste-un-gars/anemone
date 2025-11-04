#!/bin/bash
echo "[$(date)] Called with: $@" >> /tmp/dfree.log
/home/franck/anemone/anemone-dfree "$@" 2>> /tmp/dfree.log
