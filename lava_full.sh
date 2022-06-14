##############################
########   RUN LAVA    #######
##############################

echo ' ::: SERVING LAVA CHAIN (ignite 0.22.1) :::'
### Serve chain
sh ./.scripts/serve_chain.sh &
#TODO await event
echo ' ::: AWITING LAVA CHAIN ::: (40 secs) '
sleep 40

### Init Full 
###     Init chain commands
###     Run mock proxies
###     Run Providers (blocked uptil specs)
echo ' ::: RUNNING INITFULL - SPECS, PROXIES, MOCK PROVIDERS :::'
sh ./.scripts/init_full.sh 

### Run Clients
#TODO await event
echo ' ::: AWITING FOR PROVIDERS ::: (40 secs)'
echo ' ::: RUNNING INITFULL - SPECS, PROXIES, MOCK PROVIDERS :::'
sleep 40
sh ./.scripts/run_clients.sh

echo ' ::: DONE - FINISHED CLIENTS :::'
echo ''
echo ' :::::::::::::::::::::::::::::::'
echo ' :::::::::::::::::::::::::::::::'
echo ' ::: DONE - FINISHED LAVA ::::::'
echo ' :::::::::::::::::::::::::::::::'
echo ' :::::::::::::::::::::::::::::::'
echo ''

# Kill lava processes
# killall lavad starport main go 

# [+]