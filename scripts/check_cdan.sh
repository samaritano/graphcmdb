#!/bin/bash

HOSTNAME=`hostname -s | awk '{print tolower($0)}'`
case $HOSTNAME in
    *"cdan"*)
        echo "1"
        ;;
    *"elk01"*)
        echo "1"
        ;;
    *"tomcat"*)
        echo "1"
        ;;
    *)
        echo "0"
        ;;
esac