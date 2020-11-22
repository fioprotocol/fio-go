# FIO Integration - Account balances and history

There are many options available for wallets, exchanges, and information providers for presenting a user's balance and history. Not all approaches are appropriate for all applications, and how an organization integrates other blockchains may affect the strategy. Here are a few of the approaches tried with FIO, hopefully this helps explain some of the choices available during integration.

This mostly addresses the issue from the viewpoint of wanting to self-host the nodes and not to use publicly available resources. However, many of the options mentioned here are available publicly such as v1 history nodes and Hyperion.

## But first, Don't forget about the fees!

FIO differs from EOS in many ways, but one major difference may present a challenge for an integrator: fees. Fees are attached to many transaction types, and in some cases the fees are waived based on "bundled" transactions provided when registering a new FIO address. When submitting a transaction, there is a `max_fee` field that a user submits and if the fee required is less than this amount the fee is deducted. FIO will not extract more than the actual fee required for the call (unlike ethereum or bitcoin fees.) It is not a safe assumption that the `max_fee` value will be what was charged. Fees vary over time. The block producer community is a warden of this process, by adjusting fees through a voting process that should prevent token price changes from making it too expensive to use the FIO protocol.

There is one **very important** thing to note about fee collection: it is an internal-action to the contracts. Without an action trace, fees assessed or rewards paid will not be evident in a transaction.

## Getting account information

There are two approaches: pre-processing the data, and on-demand access:

### On-Demand Queries

1. _Account action traces_ using the v1 history plugin with `get_actions` API call for each account. This does have a wealth of information, but does come with some caveats. History can be truncated depending on how the node is configured, and understanding action traces can be complex. Some of the internal actions are repeated only changing the receiver (for example for fee collection).
2. _Account balance_ using the `get_currency_balance` or `get_fio_balance` APIs. This should always return a correct balance.

### Pre-processing Options

There are two approaches often seen in pre-processing, pulling data via repeated requests, and streaming data via websocket. Because of the overhead of making many repeated HTTP requests, and that nodeos does not support pipelining, any of the streaming options are going to be significantly faster. (At some point a Unix domain socket option may become available making the pull option less inefficient but as of the time of writing, it is not yet enabled in the http_plugin.) After each solution below the complexity is ranked (in terms of required infrastructure to run the solution), difficulty (how difficult it is to handle the information from the approach,) and quality (is the data complete? Is it trustworthy?)

1. [Crawl the blocks](get-block.go) using `get_block` API. Old, tried and true method, with no additional plugins required. Also slow, and the most likely to result in missing information. I really caution against using this method if accuracy is important.  There are several issues with this approach: 1) Action traces are not included in the transactions so seeing fees being charged, and rewards payouts is not possible, 2) the transactions generated as a result of a multi-signature transaction have no details, only the transaction ID is present. **low complexity, low difficulty, low quality**
1. [Crawl the blocks and then fetch full traces](v1history.go) using the v1 history endpoints `get_block_txids` and `get_transaction` endpoints. The major downside to this approach is that it requires many calls to get all of the transactions, but it does result in having full action-traces available and ensures multi-sig transactions are not missed. **low complexity, low difficulty, high quality**
1. _Consume action traces via websocket_ using the state-history-plugin. The state-history plugin is very fast and efficient at providing data, but it is difficult to understand and use directly. Queries are specified using ABI-encoded binary requests, and the data returned is also ABI encoded. Generally this is how many of the more advanced tools ingest the data before normalizing it. **low complexity, high difficulty, high quality**
1. [Chronicle](https://github.com/EOSChronicleProject/eos-chronicle): Chronicle is a tool that consumes the state-history-plugins data and converts it to JSON. It sends this data over an outgoing websocket for processing. There are some challenges here too, as many of the numeric fields are changed to a string, which can be problematic for strongly-typed languages. Chronicle has a lot of options, making it a very good choice for when integrating into a custom data backend. [fio.etl is a tool that uses Chronicle](https://github.com/fioprotocol/fio.etl) **high complexity, low difficulty, high quality**
1. [Hyperion](https://hyperion.docs.eosrio.io/) adds a large number of capabilities including streaming APIs with filtering support, v1 history compatible APIs plus many additional useful endpoints. It is a somewhat complex app, involving message queues, key-value stores, ingest processes, and an elasticsearch backend. **high complexity, low difficulty, high quality**
1. _Consume blocks via P2P_ (only recommended for near-real-time monitoring) it's possible to have a node push blocks directly over a TCP connection using the EOS p2p protocol, and then to process each block using the ABI to decode the transactions. This has the same downsides as using `get_block` and the added complexity of handling the binary protocol (but is useful for handling data real-time.) [fiowatch is an example of a tool that does this.](https://github.com/blockpane/fiowatch). **low complexity, high difficulty, low quality**

