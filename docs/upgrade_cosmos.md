
# Upgrade from Starport 0.19.2 to Ignite 0.22.1 with Cosmos 0.45.5

## upgrade your ignite to 0.22.1
```
# Remove local ignite (this is safe, as it's a different binary than starport)
rm $(which ignite)
# Get Ignite 0.22.1
curl https://get.ignite.com/cli! | bash
```

## Make sure you're on a branch that's updated to Ignite 0.22.1
## Start Lava with Ignite
ignite chain serve -v -r 2>&1 | grep -e lava_ -e ERR_ -e STARPORT] -e !