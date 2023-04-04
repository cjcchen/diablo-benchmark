package resdb_client

import (
	"fmt"
	"log"
  "time"

  "github.com/resilientdb/go-resilientdb-sdk/client"
  "github.com/resilientdb/go-resilientdb-sdk/proto"
)

func MakeTransaction(uid uint64, from string, to string, amount uint64) *resdb.Transaction {
  return &resdb.Transaction{
		Uid:uid,
		From:from,
		To:to,
		Amount:amount,
  }
}

type Client struct {
	serverIp string
	serverPort int
	channel chan int
	done_tasks chan uint64
	pending_tasks chan*resdb.Transaction
}


// MakeClient is the factory for constructing a Client for a given endpoint.
func MakeClient(ip string, port int) (c *Client, err error) {
  c = &Client{}

	c.pending_tasks = make(chan*resdb.Transaction, 1000000)
	c.done_tasks = make(chan uint64, 1000000)

  for i:=0; i < 50; i++ {
    go func(c * Client, ip string, port int){
         var client *resdb_client.TransactionClient

         client = resdb_client.MakeTransactionClient(ip,port)

         loop :  for {
          var tx *resdb.Transaction
          var err error
          var tx_list []*resdb.Transaction
          var uids map[uint64]int32
          var ok bool
          var j int

          tx_list = make([]*resdb.Transaction, 50)
          j = 0
          innerloop : for j =0; j < 50; j++ {
            select {
              case tx,ok =<- c.pending_tasks:
                if ok {
                  //tx_list = append(tx_list, tx)
                  tx_list[j] = tx
                  continue
                } else {
                  break loop
                }
              case <-time.After(time.Microsecond):
                break innerloop
              default:
                break innerloop
            }
          }

          tx_list = tx_list[0:j]
          if(len(tx_list) == 0) {
            //log.Printf("no data ")
            time.Sleep(time.Duration(time.Microsecond) * 100)
            continue
          }
          //log.Printf("send uids: %d, j = %d\n",len(tx_list), j)
          uids,err = client.SendBatchTransaction(tx_list)
          if err != nil {
            return
          }
          for uid, ret := range uids {
            //log.Printf("get uids: %d, %d\n",uid, ret)
            if ret >=0 {
              c.done_tasks <- uid
            }
          }
        }

    }(c, ip, port)
  }

	return
}




func (c *Client) SendTransaction(tx *resdb.Transaction) {
  c.pending_tasks <- tx
  return
}

func (c *Client) WaitNextTxn() (uid uint64, err error){
	var tx uint64
	var ok bool
	tx, ok = <-c.done_tasks
	if ok {
		//log.Printf("get one txn %d\n",tx)
    return tx, nil
	} else {
		log.Printf("get one txn fail\n")
		return 0, fmt.Errorf("get txn fail")
	}
}

