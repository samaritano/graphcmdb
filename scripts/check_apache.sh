#!/bin/bash

sudo service httpd status 2> /dev/null | grep "active (running)" | wc -l