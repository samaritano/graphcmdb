#!/bin/bash

HOSTNAME=`hostname -s | awk '{print tolower($0)}'`
case $HOSTNAME in
    *"wt"*)
        echo "1"
        ;;
    *)
        echo "0"
        ;;
esac