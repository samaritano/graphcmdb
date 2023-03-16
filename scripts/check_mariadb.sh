#!/bin/bash

RESULT=`ps -ef | grep  mysqld | grep -v grep | wc -l`
if [ $RESULT -eq "0" ]; then
    RESULT=`ps -ef | grep  mariadbd | grep -v grep | wc -l`
    if [ $RESULT -eq "0" ]; then
        echo "0"
    else
        echo "1"
    fi
else
    echo "1"
fi