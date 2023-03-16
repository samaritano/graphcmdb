#!/bin/bash

df -T | grep nfs | grep -v .snapshot | grep -v tmpfs | awk '{print $1","$7","$3","$4","$5}'