#!/bin/bash

HOSTNAME=`hostname -s | awk '{print tolower($0)}'`
case $HOSTNAME in
    *"buf"*)
        echo "1"
        ;;
    *)
        echo "0"
        ;;
esac