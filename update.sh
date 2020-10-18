#!/bin/sh
#
# update ddns
#

# MyDNS
wget -O - --http-user=$MYDNSID --http-password=$MYDNSPW http://www.mydns.jp/login.html