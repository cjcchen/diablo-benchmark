package main

import (
	"log"
	"diablo-benchmark/blockchains/nresilientdb/resdb_client/client"
)



func main() {
	var client *resdb.Client
	var tx *resdb.Transaction
	var uid uint64

	tx = resdb.MakeTransaction(1, "from", "to", 1)

	client,_ = resdb.MakeClient("127.0.0.1",30005)
	client.SendTransaction(tx)

	uid, _ = client.WaitNextTxn()
	log.Printf("get uid %d\n",uid)
}
