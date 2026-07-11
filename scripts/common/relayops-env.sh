#!/usr/bin/env bash

relayops_card_from_control() {

  basename "$(readlink -f "$1")" | sed 's/controlC//'

}

relayops_ts890_card() {

  relayops_card_from_control /dev/snd/by-radio/ts890-control

}

relayops_ic9700_card() {

  relayops_card_from_control /dev/snd/by-radio/ic9700-control

}

TS890_CARD="$(relayops_ts890_card)"
IC9700_CARD="$(relayops_ic9700_card)"

export DISPLAY="${DISPLAY:-:0}"
export XAUTHORITY="${XAUTHORITY:-$HOME/.Xauthority}"

# Common CrossOver settings
export BOTTLE="Winlink"
