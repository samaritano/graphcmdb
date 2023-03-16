#!/bin/bash

netstat -ant -4 |
grep -e "^tcp" |
grep -v -i listen |
awk '{print $4","$5}' |
sed "s/:/,/g" |
sed "s/,3306,/,3306 (mariadb),/g" |
sed "s/,3306$/,3306 (mariadb)/g" |
sed "s/,9200,/,9200 (elk),/g" |
sed "s/,9200$/,9200 (elk)/g" |
sed "s/,22,/,22 (ssh),/g" |
sed "s/,22$/,22 (ssh)/g" |
sed "s/,80,/,80 (http),/g" |
sed "s/,80$/,80 (http)/g" |
sed "s/,443,/,443 (https),/g" |
sed "s/,443$/,443 (https)/g" |
sed "s/,9901,/,9901 (jexecso),/g" |
sed "s/,9901$/,9901 (jexecso)/g" |
sed "s/,3307,/,3307 (stexecso),/g" |
sed "s/,3307$/,3307 (stexecso)/g" |
sed "s/,2049,/,2049 (nfs),/g" |
sed "s/,2049$/,2049 (nfs)/g" |
sed "s/,27017,/,27017 (mongodb),/g" |
sed "s/,27017$/,27017 (mongodb)/g"