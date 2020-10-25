#!/bin/bash
C="$1"
[ -z "$C" ] && exit
curl -s "https://api.bloxy.info/token/token_holders_list?token=$C&key=ACCTGSPvYX4Lr&format=list" 
