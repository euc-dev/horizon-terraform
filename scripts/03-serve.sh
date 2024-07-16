#!/bin/bash

# set common env vars / etc
source scripts/00-env.sh

# render markdown to html and serve it on port tcp/80
mkdocs serve -f $CONFIG --dev-addr=0.0.0.0:80

# vim: set syn=sh ft=unix ts=4 sw=4 et tw=78:
