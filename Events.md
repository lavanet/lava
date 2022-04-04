# Events
* this document describes all of the events on the lavad consensus
* events are divided between events types such as: 'NewBlock' 'Tx' etc..
* a client can subscribe to event types. see more at: https://docs.tendermint.com/master/tendermint-core/subscription.html#subscribing-to-events-via-websocket, API docs link within the page has further information

# Events Description:

Module | Event name | type | description | attr name | attribute | name | attribute | name | attribute | name | attribute |
------ | ---------- | ---- | ----------- | --------- | --------- | --------- | --------- | --------- | --------- | --------- | --------- |
User | lava_user_stake_new | Tx | sent upon successful stake of a new user | user | staked user address | deadline | the block height in which the user can start getting pairings | stake | the stake deposited by the user | requestedDeadline | the deadline the user requested
User | lava_user_stake_update | Tx | sent upon successful update of an existing user stake | user | staked user address | deadline | the block height in which the user can start getting pairings | stake | the stake deposited by the user | requestedDeadline | the deadline the user requested
User | lava_user_unstake_schedule | Tx | sent upon successful registration for unstaking | user | unstaked user address requested | deadline | the block height in which the user will be fully unstaked | stake | the stake that will be claimed by the user | requestedDeadline | the deadline the user requested for unstaking
User | lava_user_unstake_commit | NewBlock | sent upon the commit of a registered unstake request | user | unstaked user address requested | stake | the stake that will be claimed by the user 
Servicer | lava_servicer_stake_new | Tx | sent upon successful stake of a new servicer | servicer | staked servicer address | deadline | the block height in which the servicer can start getting pairings | stake | the stake deposited by the servicer | requestedDeadline | the deadline the servicer requested
Servicer | lava_servicer_stake_update | Tx | sent upon successful update of an existing servicer stake | servicer | staked servicer address | deadline | the block height in which the servicer can start getting pairings | stake | the stake deposited by the servicer | requestedDeadline | the deadline the servicer requested
Servicer | lava_servicer_unstake_schedule | Tx | sent upon successful registration for unstaking | servicer | unstaked servicer address requested | deadline | the block height in which the servicer will be fully unstaked | stake | the stake that will be claimed by the servicer | requestedDeadline | the deadline the servicer requested for unstaking
Servicer | lava_servicer_unstake_commit | NewBlock | sent upon the commit of a registered unstake request | servicer | unstaked servicer address requested | stake | the stake that will be claimed by the servicer 
Servicer | lava_relay_payment | Tx | sent upon the successful payment for a relay batch | client | the client that requested the relay | servicer |  the servicer that got paid for his work | CU | the compute units delivered | Mint | the coins minted for the servicer
Gov | lava_param_change | Tx | sent upon the successful change of a param by GOV | param | the param name to be changed | value | the new value

# Logging
## events
* all added events should start with "lava_" for differentiation between cosmos and tendermint events and custom lava events
* events should have an indicative name, stating what is the action that is logged
* error events should be created too, starting with ERR_, for failed state transitions.
* a wrapper for all events exist in utils/lavalog.go
