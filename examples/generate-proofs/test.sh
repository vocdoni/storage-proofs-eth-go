#!/bin/bash
go run . --contract=0xdac17f958d2ee523a2206206994597c13d831ec7 --holderFile=tether_holders.txt
echo "proofs saved on proofs.json file"
