#!/usr/bin/env bash

TS890_CARD=$(basename "$(readlink -f /dev/snd/by-radio/ts890-control)" | sed 's/controlC//')
IC9700_CARD=$(basename "$(readlink -f /dev/snd/by-radio/ic9700-control)" | sed 's/controlC//')