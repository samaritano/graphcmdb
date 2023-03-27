#!/bin/bash

HOSTNAME=`hostname -s | awk '{print tolower($0)}'`
case $HOSTNAME in
    *"pippo"*)
        echo "1"
        ;;
    *"pluto"*)
        echo "1"
        ;;
    *"topolino"*)
        echo "1"
        ;;
    *)
        echo "0"
        ;;
esac