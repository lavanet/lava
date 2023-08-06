
# Upgrade from Starport 0.19.2 to Ignite v0.27.1 with Cosmos 0.46.7

## upgrade your ignite to v0.27.1
```
# Remove local ignite (this is safe, as it's a different binary than starport)
rm $(which ignite)
# Get Ignite v0.27.1
curl https://get.ignite.com/cli! | bash
```

## Make sure you're on a branch that's updated to Ignite 0.27.1
## Start Lava with Ignite
ignite chain serve -v -r 2>&1 | grep -e lava_ -e ERR_ -e IGNITE] -e !