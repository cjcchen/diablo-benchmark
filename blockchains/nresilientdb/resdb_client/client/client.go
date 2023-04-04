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
	channel chan *resdb_client.TransactionClient
	done_tasks chan *Transaction
  client *resdb_client.TransactionClient
}


// MakeClient is the factory for constructing a Client for a given endpoint.
func MakeClient(ip string, port int) (c *Client, err error) {
  c = &Client{}
	c.channel =  make(chan *resdb_client.TransactionClient, 1000)
	c.done_tasks = make(chan *Transaction, 100000)

  for i:=0; i < 1000; i++ {
    c.channel <- resdb_client.MakeTransactionClient(ip,port)
  }
	return
}

func (c *Client) SendTransaction(tx *Transaction) {
  var res_cli *resdb_client.TransactionClient

  res_cli =<- c.channel

  go func(this *Client, c *resdb_client.TransactionClient, tx *Transaction){
      var uid uint64
      var err error

      if tx.uid %10000 == 0 {
        log.Printf("tx done, uid %d", tx.uid)
      }
      uid,err = c.SendRawTransaction(tx.uid, tx.from, tx.to, tx.amount)
      if (uid != tx.uid || err != nil) {
        if err != nil {
          log.Println("get error %s", err)
        } else {
          log.Println("get uid %d tx uid %d", uid, tx.uid)
        }
        tx.committed = false
      } else {
          //log.Println("get uid %d done", uid)
        tx.committed = true
      }
      this.done_tasks <- tx
      this.channel <- c
  }(c, res_cli, tx)

  //log.Printf("send txn done")
  return
}

func (c *Client) WaitNextTxn() (uid uint64, err error){
	var tx *Transaction
	var ok bool
	tx, ok = <-c.done_tasks
	if ok {
      if tx.uid %10000 == 0 {
        log.Printf("get one txn %d, status %d\n",tx.GetUid(), tx.committed)
      }
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

