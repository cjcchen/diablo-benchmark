package resdb

import (
	"fmt"
	"log"

  "github.com/resilientdb/go-resilientdb-sdk/client"
)

type Transaction struct {
	uid  uint64
	from      string
	to        string
	amount    uint64
	sign  []byte
  committed bool
}

func (this *Transaction) GetUid() uint64 {
	return this.uid
}

func MakeTransaction(uid uint64, from string, to string, amount uint64) *Transaction {
  return &Transaction{
		uid:uid,
		from:from,
		to:to,
		amount:amount,
  }
}

type Client struct {
	serverIp string
	serverPort int
	channel chan int
	done_tasks chan *Transaction
  client *resdb_client.TransactionClient
}


// MakeClient is the factory for constructing a Client for a given endpoint.
func MakeClient(ip string, port int) (c *Client, err error) {
  c = &Client{}
  c.client = resdb_client.MakeTransactionClient(ip,port)
	c.channel =  make(chan int, 10)
	c.done_tasks = make(chan *Transaction, 100)
	return
}

func (c *Client) SendTransaction(tx *Transaction) {
  c.channel <- 1

  go func(c * Client, tx *Transaction){
      var uid uint64
      var err error

      //log.Printf("tx done, uid %d", tx.uid)
      uid,err = c.client.SendRawTransaction(tx.uid, tx.from, tx.to, tx.amount)
      if (uid != tx.uid || err != nil) {
        tx.committed = false
      } else {
        tx.committed = true
      }
      c.done_tasks <- tx
      <-c.channel
  }(c, tx)

  //log.Printf("send txn done")
  return
}

func (c *Client) WaitNextTxn() (uid uint64, err error){
	var tx *Transaction
	var ok bool
	tx, ok = <-c.done_tasks
	if ok {
		//log.Printf("get one txn %d, status %d\n",tx.GetUid(), tx.committed)
    if tx.committed {
      return tx.GetUid(), nil
    } else {
      log.Printf("get one txn fail\n")
      return tx.GetUid(), fmt.Errorf("submit txn fail")
    }
	} else {
		log.Printf("get one txn fail\n")
		return 0, fmt.Errorf("get txn fail")
	}
}

