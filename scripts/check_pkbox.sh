#!/bin/bash

RESULT=`ps -ef | grep -i pkbox | grep org.apache.catalina.startup.Bootstrap | grep -v grep | wc -l`
if [ $RESULT -eq "0" ]; then
    echo "0"
else
    echo "1"
fi