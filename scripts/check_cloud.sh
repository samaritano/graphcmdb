#!/bin/bash

HOSTNAME=`hostname -s | awk '{print tolower($0)}'`
case $HOSTNAME in
    *"apache"*)
        echo "1"
        ;;
    *"dbapm"*)
        echo "1"
        ;;
    *"dbaps"*)
        echo "1"
        ;;
    *"jexecso"*)
        echo "1"
        ;;
    *"stexecso"*)
        echo "1"
        ;;
    *"urbi"*)
        if [[ $HOSTNAME == *"buf"* ]]; then
            echo "0"
        else
            if [[ $HOSTNAME == *"wt"* ]]; then
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