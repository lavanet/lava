# Events
* this document describes all of the events on the lavad consensus
* events are divided between events types such as: 'NewBlock' 'Tx' etc..
* a client can subscribe to event types. see more at: https://docs.tendermint.com/master/tendermint-core/subscription.html#subscribing-to-events-via-websocket, API docs link within the page has further information

# Events Description:

Module | Event name | type | description | attr name | attribute | name | attribute | name | attribute | name | attribute | name | attribute | name | attribute |
------ | ---------- | ---- | ----------- | --------- | --------- | --------- | --------- | --------- | --------- | --------- | --------- | --------- | --------- | --------- | --------- |
User | lava_user_stake_new | Tx | sent upon successful stake of a new user | spec | spec name | user | staked user address | deadline | the block height in which the user can start getting pairings | stake | the stake deposited by the user | requestedDeadline | the deadline the user requested
User | lava_user_stake_update | Tx | sent upon successful update of an existing user stake | spec | spec name | user | staked user address | deadline | the block height in which the user can start getting pairings | stake | the stake deposited by the user | requestedDeadline | the deadline the user requested
User | lava_user_unstake_schedule | Tx | sent upon successful registration for unstaking | spec | spec name | user | unstaked user address requested | deadline | the block height in which the user will be fully unstaked | stake | the stake that will be claimed by the user | requestedDeadline | the deadline the user requested for unstaking
User | lava_user_unstake_commit | NewBlock | sent upon the commit of a registered unstake request | spec | the spec name | user | unstaked user address requested | stake | the stake that will be claimed by the user 
Servicer | lava_servicer_stake_new | Tx | sent upon successful stake of a new servicer | spec | the spec name | servicer | staked servicer address | deadline | the block height in which the servicer can start getting pairings | stake | the stake deposited by the servicer | requestedDeadline | the deadline the servicer requested
Servicer | lava_servicer_stake_update | Tx | sent upon successful update of an existing servicer stake | spec | the spec name | servicer | staked servicer address | deadline | the block height in which the servicer can start getting pairings | stake | the stake deposited by the servicer | requestedDeadline | the deadline the servicer requested
Servicer | lava_servicer_unstake_schedule | Tx | sent upon successful registration for unstaking | spec | the spec name | servicer | unstaked servicer address requested | deadline | the block height in which the servicer will be fully unstaked | stake | the stake that will be claimed by the servicer | requestedDeadline | the deadline the servicer requested for unstaking
Servicer | lava_servicer_unstake_commit | NewBlock | sent upon the commit of a registered unstake request | spec | the spec name | servicer | unstaked servicer address requested | stake | the stake that will be claimed by the servicer 
Servicer | lava_relay_payment | Tx | sent upon the successful payment for a relay batch | chainID | the ID of the chain | client | the client that requested the relay | servicer |  the servicer that got paid for his work | CU | the compute units delivered | Mint | the coins minted for the servicer | totalCUInSession | the total CU used by the client in all of the session | isOverlap | true/false for overlap between sessions
Gov | lava_param_change | Tx | sent upon the successful change of a param by GOV | param | the param name to be changed | value | the new value
Spec | lava_spec_add | Tx | sent upon adding a spec proposal passed and performed | spec | the spec name | status | the spec status | chainID | the spec chain ID
Spec | lava_spec_modify | Tx | sent upon modifying an existing spec proposal passed and performed | spec | the spec name | status | the spec status | chainID | the spec chain ID


# Logging
## events
* all added events should start with "lava_" for differentiation between cosmos and tendermint events and custom lava events
* events should have an indicative name, stating what is the action that is logged
* error events should be created too, starting with ERR_, for failed state transitions.
* a wrapper for all events exist in utils/lavalog.go
```
utils.LogLavaEvent()
utils.LavaError()
```
# Errors description

Module | Event name | type | description | attr name | attribute | name | attribute | name | attribute | name | attribute | name | attribute | name | attribute | name | attribute | name | attribute |
------ | ---------- | ---- | ----------- | --------- | --------- | --------- | --------- | --------- | --------- | --------- | --------- | --------- | --------- | --------- | --------- | --------- | --------- | --------- | --------- |
Servicer | ERR_new_session | NewBlock | sent upon an error on session parameters update | error | the error that occured
Servicer | ERR_servicer_unstaking_credit | NewBlock | sent upon an error in crediting a servicer after unstake deadline was reached | spec | the spec name | error | the error that occured | servicer | unstaking servicer address | stake | the stake that needs to be credited
Servicer | ERR_servicer_unstaking_deadline | NewBlock | sent upon an error in setting a new deadline for unstaking callback  | minDeadline | the callback we are trying to set | height | the block we are currently at | unstakingCount | the number of unstaking servicers | deletedIndexes | the amount of unstaking servicers we credited already
Servicer | ERR_relay_proof_sig | Tx | sent upon an error in recovery of the public key of a client in relay proof  | sig | the data that was parsing the key
Servicer | ERR_relay_proof_user_addr | Tx | sent upon an error in parsing the address of a user in a relay | user | the data that was parsed for address
Servicer | ERR_relay_proof_addr | Tx | sent upon an error in parsing the address of a servicer in a relay or mismatch with the signed client address | servicer | the data that was parsed for address | creator | the creator address on the transaction
Servicer | ERR_relay_proof_spec | Tx | sent upon an error in the specified spec of a relay | chainID | the index of the spec that was specified
Servicer | ERR_relay_proof_pairing | Tx | sent upon an error in the pairing between servicer and client | client | the client address | servicer | the address of the servicer | error | the error in the pairing
Servicer | ERR_relay_proof_session | Tx | sent upon an error in reading the session parameters | height | the block height | error | the error that was received
Servicer | ERR_relay_proof_claim | Tx | sent upon identification of a double spend attempt | session | the session start | client | the client address | servicer | the address of the servicer | error | the error in the identifier check | unique_ID | the relay unique identifier
Servicer | ERR_relay_proof_user_limit | Tx | sent upon an error in the user CU usage | session | the session start | client | the client address | servicer | the address of the servicer error | the error in enforcing the CU usage | CU | the CU requested to add | totalCUInSession | the total CU used by the client in this session
Servicer | ERR_relay_proof_burn | Tx | sent upon an error in burning a client stake after usage | client | the client that requested the relay | servicer |  the servicer that got paid for his work | CU | the compute units delivered | Mint | the coins minted for the servicer | totalCUInSession | the total CU used by the client in all of the session | isOverlap | true/false for overlap between sessions | amountToBurn | the amount trying to burn | error | the error in burning
Servicer | ERR_relay_payment | Tx | sent upon an error in paying to servicer | chainID | the ID of the chain | client | the client that requested the relay | servicer |  the servicer that got paid for his work | CU | the compute units delivered | Mint | the coins minted for the servicer | totalCUInSession | the total CU used by the client in all of the session | isOverlap | true/false for overlap between sessions | error | the error in minting
Servicer | ERR_stake_servicer_spec | Tx | sent if the requested spec isn't found or isn't active | spec | the name of the spec that was attempted to retrieve
Servicer | ERR_stake_servicer_amount | Tx | sent if on staking request the amount is insufficient | spec | the name of the spec | servicer | the staking servicer address | stake | the stake requested | minStake | the minimum stake needed
Servicer | ERR_stake_servicer_addr | Tx | sent if the staking servicer address is invalid | servicer | the staking servicer address | error | the error that was triggered
Servicer | ERR_stake_servicer_operator | Tx | sent if the operator address provided is invalid | error | the error that was triggered | operator | the operator address that triggered it
Servicer | ERR_stake_servicer_update_amount | Tx | sent if on update of existing servicer stake amount, there isn't enough funds | spec | the spec name | servicer | staked servicer address | deadline | the block height in which the servicer can start getting pairings | stake | the stake deposited by the servicer | requestedDeadline | the deadline the servicer requested | error | the error that was triggered | neededStake | the stake that is needed to be paid
Servicer | ERR_stake_servicer_deadline | Tx | sent if a stake update request tries to increase the deadline | spec | the spec name | servicer | staked servicer address | deadline | the block height in which the servicer can start getting pairings | stake | the stake deposited by the servicer | requestedDeadline | the deadline the servicer requested
Servicer | ERR_stake_servicer_stake | Tx | sent if a stake update tries to lower stake | spec | the spec name | servicer | staked servicer address | deadline | the block height in which the servicer can start getting pairings | stake | the stake deposited by the servicer | requestedDeadline | the deadline the servicer requested | existingStake | the existing stake
Servicer | ERR_stake_servicer_new_amount | Tx | sent when insufficient amount to pay for requested stake | spec | the spec name | servicer | staked servicer address | deadline | the block height in which the servicer can start getting pairings | stake | the stake deposited by the servicer | requestedDeadline | the deadline the servicer requested | error | the error that was sent
Servicer | ERR_unstake_servicer_spec | Tx | sent when the spec specified  on unstake tx is invalid or not active | spec | the spec name
Servicer | ERR_unstake_servicer_storage | Tx | sent when there is a problem with the storage fdor handling the request | error | the storage that had the problem
Servicer | ERR_unstake_servicer_entry | Tx | sent when the entry for unstaking wasn't found meaning servicer wasn't staked | servicer | the address of the servicer | spec | the spec of the servicer
Gov | ERR_param_change | Tx | sent when a param change proposal passed but had an error | param | the param to change | value | the new value | error | the error that was triggered
Spec | ERR_spec_add_dup | Tx | sent when a proposal tried to add an existing spec | spec | the name of the spec | status | the status of the spec | chainID | the index of the spec
Spec | ERR_spec_modify_missing | Tx | sent when a proposal tried to modify a spec but wasn't found | spec | the name of the spec | status | the status of the spec | chainID | the index of the spec
User | ERR_user_unstaking_credit | NewBlock | sent upon an error in crediting a user after unstake deadline was reached | error | the error that occured | spec | the spec name | user | unstaking user address | stake | the stake that needs to be credited
User | ERR_user_unstaking_deadline | NewBlock | sent upon an invalid callback schedule for next unstaking commit | minDeadline | the next callback block | height | the current block | unstakingCount | the amount of unstaking users | deletedIndexes | the amount of users credited this callback
User | ERR_user_unstake_spec | Tx | sent when trying top unstake a user but the spec name is wrong | spec | spec name
User | ERR_user_unstake_entry | Tx | sent when trying to unstake a user, but the user isn't staked | spec | spec name | user | user address
User | ERR_stake_user_spec | Tx | sent when trying to stake user but the spec name is wrong | spec | spec name
User | ERR_stake_user_amount | Tx | sent when trying to stake user but the stake amount is too low | spec | spec name | user | user address | stake | the stake amount | minStake | the needed stake amount
User | ERR_stake_user_addr | Tx | sent when trying to stake user but the address is bad | spec | spec name | user | the user address
User | ERR_stake_user_modify_amount | Tx | sent when trying to modify a user stake but the balance is insufficient | spec | spec name | user | the user address | deadline | the block in which it takes effect | stake | the stake amount | requestedDeadline | the requested deadline | error | the error that occured | neededStake | the stake that is needed
User | ERR_stake_user_deadline | Tx | sent when trying to modify a user stake but the deadline requested is invalid | spec | spec name | user | the user address | deadline | the block in which it takes effect | stake | the stake amount | requestedDeadline | the requested deadline
User | ERR_stake_user_stake | Tx | sent when trying to modify a user stake but the new stake is lower | spec | spec name | user | the user address | deadline | the block in which it takes effect | stake | the stake amount | requestedDeadline | the requested deadline | existingStake | the existing stake
User | ERR_stake_user_new_amount | Tx | sent when trying to add a user stake but the user balance is too low | spec | spec name | user | the user address | deadline | the block in which it takes effect | stake | the stake amount | requestedDeadline | the requested deadline | error | the error that occured
