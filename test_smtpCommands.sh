#!/bin/bash

proxyPort="10025"

cat smtp_commands.txt | nc localhost ${proxyPort} 
