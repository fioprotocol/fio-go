# fioreq

Command line tool for FIO requests.

Note: requires keosd for wallet functionality.

![usage example](fioreq.gif)

Examples:

```
$ fioreq -example

Important Options:
------------------
         -u 'URL for FIO nodeos endpoint'
  -password 'password for keosd wallet'
         -c 'fioreq command'
         -p 'actor permission (account) for transaction'
        -id 'request ID for command'
     -payee 'FIO Address that *recieves* funds'
     -payer 'FIO Address that *sends* funds'


View available accounts from keosd:
-----------------------------------
  fioreq -u https://testnet.fioprotocol.io -password PW5xxxx.... -c list


View sent requests for an account:
----------------------------------
  fioreq -u https://testnet.fioprotocol.io -password PW5xxxx.... -p aaaaaaaaaaaa -c sent


View pending requests for an account:
-------------------------------------
  fioreq -u https://testnet.fioprotocol.io -password PW5xxxx.... -p aaaaaaaaaaaa -c pending


View details for a request (including response):
------------------------------------------------
  fioreq -u https://testnet.fioprotocol.io -password PW5xxxx.... -p aaaaaaaaaaaa -c view-req -id 123


Reject a pending request
------------------------
  fioreq -u https://testnet.fioprotocol.io -password PW5xxxx.... -p aaaaaaaaaaaa -c reject -id 321


Request Payment:
----------------
  fioreq -u https://testnet.fioprotocol.io -password PW5xxxx.... -p aaaaaaaaaaaa -c request -payer shopper@fiotestnet -payee merchant@store '
    {
      "payee_public_address": "0x42F6cA7898A0f29e17CB66190f9E9B9d26f7D635",
      "amount": "123.45",
      "chain_code": "ETH",
      "token_code": "USDT",
      "memo": "payment for order 123"
    }'


Record a transaction for a pending request
------------------------------------------
  fioreq -u https://testnet.fioprotocol.io -password PW5xxxx.... -p aaaaaaaaaaaa -c record -id 321 -payee merchant@store -payer shopper@fiotestnet '
    {
      "payer_public_address": "FIO6ZJ9p6ZSvboXqaFiowR8bKLtSk8ZGUTHdT8ZkaW6pNnbusPdwa",
      "payee_public_address": "FIO6QtJu52ho38zRP4aZCcgtciLAWQUB3CBgXnmwfFfXi6LvfVYyj",
      "amount": "1.000",
      "chain_code": "FIO",
      "token_code": "FIO",
      "hash": "797c59d1601f6bd99f0b56deb2c4fca944501a12b750829b66e4f792b0019fd4"
    }'
```
