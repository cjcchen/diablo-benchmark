package main

import (
	"log"
	"diablo-benchmark/blockchains/nresilientdb/resdb_client/client"
  "github.com/resilientdb/go-resilientdb-sdk/proto"
)


func main() {
	var client *resdb_client.Client
	var tx *resdb.Transaction
	var uid uint64

	tx = resdb_client.MakeTransaction(1, "from", "to", 1)

	client,_ = resdb_client.MakeClient("127.0.0.1",10005)
	client.SendTransaction(tx)

	uid, _ = client.WaitNextTxn()
	log.Printf("get uid %d\n",uid)
}
