#!/bin/bash

# set common env vars / etc
source scripts/00-env.sh

# render markdown to html
mkdocs build -f $CONFIG

# vim: set syn=sh ft=unix ts=4 sw=4 et tw=78:
