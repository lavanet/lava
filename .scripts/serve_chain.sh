# echo ' ::: SERVING LAVA CHAIN (starport 0.19.2) :::'
(cd $LAVA && starport chain serve -v -r) 2>&1 | grep -e lava_ -e ERR_ -e STARPORT] -e !
# (cd /go/lava && starport chain serve -v -r) 2>&1 | grep -e lava_ -e ERR_ -e STARPORT] -e !