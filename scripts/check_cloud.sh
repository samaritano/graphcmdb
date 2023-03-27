#!/bin/bash

HOSTNAME=`hostname -s | awk '{print tolower($0)}'`
case $HOSTNAME in
    *"server1"*)
        echo "1"
        ;;
    *"server2"*)
        echo "1"
        ;;
    *"server3"*)
        echo "1"
        ;;
    *"server4"*)
        echo "1"
        ;;
    *"server5"*)
        echo "1"
        ;;
    *"server6"*)
        if [[ $HOSTNAME == *"app1"* ]]; then
            echo "0"
        else
            if [[ $HOSTNAME == *"app2"* ]]; then
                echo "0"
            else
                echo "1"
            fi
        fi
        ;;
    *)
        echo "0"
        ;;
esac