#!/bin/bash
set -e

cat 50-top-contracts-with-holder.txt | while read l; do 
  c=$(echo $l| cut -d " " -f1) 
  h=$(echo $l|cut -d " " -f2) 
  go run . -contract=$c -holder=$h
done

