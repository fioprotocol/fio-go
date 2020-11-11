package main

/*
Example of how to derive the public key used to sign a transaction, process is similar for any signed item,
the important part here is using the sig digest to derive the public key.
*/

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/fioprotocol/fio-go"
	"github.com/fioprotocol/fio-go/eos"
	"log"
)

// NOTE: this example relies upon a v1 history node, otherwise will result in the error:
// 'Not Found: unspecified: Unknown Endpoint' when calling 'get_key_accounts'

func main() {

	const nodeos = "https://testnet.fio.dev"

	// just a helper to keep noise down in the example, bails on an error
	e := func(err error) {
		if err != nil {
			log.Fatal(err)
		}
	}

	// chain id is used to derive the signer's key
	chainId, err := hex.DecodeString(`b20901380af44ef59c5918439a1f9a41d83669020319a80574b804a5f95cbd7e`)
	e(err)

	trx := eos.TransactionReceipt{}
	err = json.Unmarshal(transaction, &trx)
	e(err)

	// to derive the pubkey we provide the sig digest (*not* the transaction id, which although *is* a hash--it's not
	// the hash needed to get the signature.) The digest is based on the chain id and transaction's raw bytes
	pk, err := trx.Transaction.Packed.Signatures[0].PublicKey(eos.SigDigest(chainId, trx.Transaction.Packed.PackedTransaction, nil))
	e(err)

	actor, err := fio.ActorFromPub(pk.String())
	e(err)

	// signed by FIO6KGNdc5DyAYkkuNGWbu2Pab91eU3FAfYgZNwSivfXfDcWjnxqk voterxbrq3vw
	fmt.Println("signed by", pk, actor)

	// this transaction is a special situation, chosen specifically because the account name for the signature does not match
	// the @active permission, let's verify, unpack first:
	unpacked, err := trx.Transaction.Packed.Unpack()
	e(err)

	if unpacked.Actions[0].Authorization[0].Actor != actor {
		// Oh no, actor rssuh33ljdbm and signer voterxbrq3vw differ!
		fmt.Println("Oh no, actor", unpacked.Actions[0].Authorization[0].Actor, "and signer", actor, "differ!")

		// find what accounts the signer has been granted access to:
		api, _, err := fio.NewConnection(nil, nodeos)
		e(err)
		resp, err := api.GetKeyAccounts(pk.String())
		e(err)

		for _, controlled := range resp.AccountNames {
			if string(unpacked.Actions[0].Authorization[0].Actor) == controlled {
				// however, all is good: voterxbrq3vw is a controlling account for rssuh33ljdbm
				fmt.Println("however, all is good:", actor, "is a controlling account for", controlled)
				break
			}
		}
	}
}

// The transaction to deserialize and verify. This example uses a delegated permission, so the signing key's FIO account
// hash won't match the actor, and requires additional verification. Although it's normally *assumed* that the actor is
// a static value in FIO based on a hash of the public key, it isn't necessarily the case, so this is a dual example
// in that it shows both how to get the pubkey and that it doesn't necessarily always correlate 1:1 with the account hash.
//
// The EOS authentication system is still in effect in FIO, so don't get fooled by the 1:1 relationship to actor and key!
var transaction = []byte(`
{
  "status": "executed",
  "cpu_usage_us": 7509,
  "net_usage_words": 66,
  "trx": {
    "id": "d432b3ebbc94879404210c6ca9c38187c5da892908091ccb1372695c2272615f",
    "signatures": [
      "SIG_K1_K7XJRUB3nbX6zmR4fmYtmVxnu4yfCvXPUEAva2sfvP5rtz8x7kA33mygf3Nyq8EPSECxaa7hRiqtLw7JzTXFLK9GSXoykw"
    ],
    "compression": "none",
    "packed_context_free_data": "",
    "context_free_data": [],
    "packed_trx": "b3b3175fd36bed0820a400000000010000000000ea30557015d289deaa32dd01204f7a718ca631be0000000080ab32ddae03131661636865726f6e2d6270334066696f746573746e657412616c70686162704066696f746573746e657417617267656e74696e6166696f4066696f746573746e65741761786d6e3567676b313469664066696f746573746e657414626c6f636b70616e654066696f746573746e65740c62704066696f73776564656e1663727970746f6c696f6e734066696f746573746e65741663757272656e63796875624066696f746573746e657417656f73616d7374657264616d4066696f746573746e657417656f7362617263656c6f6e614066696f746573746e657411656f736461634066696f746573746e657413656f7370686572654066696f746573746e657411656f737573614066696f746573746e657414657665727374616b654066696f746573746e65740e6774674066696f746573746e6574156d616c7461626c6f636b4066696f746573746e6574126e6f64656f6e654066696f746573746e6574177465616d677265796d6173734066696f746573746e6574147a656e626c6f636b734066696f746573746e657410766f7465324066696f746573746e6574204f7a718ca631be0092fe1e0000000000",
    "transaction": {
      "expiration": "2020-07-22T03:34:11",
      "ref_block_num": 27603,
      "ref_block_prefix": 2753562861,
      "max_net_usage_words": 0,
      "max_cpu_usage_ms": 0,
      "delay_sec": 0,
      "context_free_actions": [],
      "actions": [
        {
          "account": "eosio",
          "name": "voteproducer",
          "authorization": [
            {
              "actor": "rssuh33ljdbm",
              "permission": "voter"
            }
          ],
          "data": {
            "producers": [
              "acheron-bp3@fiotestnet",
              "alphabp@fiotestnet",
              "argentinafio@fiotestnet",
              "axmn5ggk14if@fiotestnet",
              "blockpane@fiotestnet",
              "bp@fiosweden",
              "cryptolions@fiotestnet",
              "currencyhub@fiotestnet",
              "eosamsterdam@fiotestnet",
              "eosbarcelona@fiotestnet",
              "eosdac@fiotestnet",
              "eosphere@fiotestnet",
              "eosusa@fiotestnet",
              "everstake@fiotestnet",
              "gtg@fiotestnet",
              "maltablock@fiotestnet",
              "nodeone@fiotestnet",
              "teamgreymass@fiotestnet",
              "zenblocks@fiotestnet"
            ],
            "fio_address": "vote2@fiotestnet",
            "actor": "rssuh33ljdbm",
            "max_fee": 520000000
          },
          "hex_data": "131661636865726f6e2d6270334066696f746573746e657412616c70686162704066696f746573746e657417617267656e74696e6166696f4066696f746573746e65741761786d6e3567676b313469664066696f746573746e657414626c6f636b70616e654066696f746573746e65740c62704066696f73776564656e1663727970746f6c696f6e734066696f746573746e65741663757272656e63796875624066696f746573746e657417656f73616d7374657264616d4066696f746573746e657417656f7362617263656c6f6e614066696f746573746e657411656f736461634066696f746573746e657413656f7370686572654066696f746573746e657411656f737573614066696f746573746e657414657665727374616b654066696f746573746e65740e6774674066696f746573746e6574156d616c7461626c6f636b4066696f746573746e6574126e6f64656f6e654066696f746573746e6574177465616d677265796d6173734066696f746573746e6574147a656e626c6f636b734066696f746573746e657410766f7465324066696f746573746e6574204f7a718ca631be0092fe1e00000000"
        }
      ],
      "transaction_extensions": []
    }
  }
}
`)
