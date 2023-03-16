#!/bin/bash

for LINE in `df -T | grep -v nfs | grep -v .snapshot | grep -v "docker/overlay" | grep -v tmpfs | awk '{if(NR>1)print $7","$3","$4","$5}'`; do echo $HOSTNAME.$LINE; done| grep .