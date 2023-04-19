package resdb_client

import (
	//"fmt"
	//"log"
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
	fail_task []uint64
	done_tasks chan uint64
	pending_tasks chan*resdb.Transaction
  poc_client *resdb_client.PoCTransactionClient
}

// MakeClient is the factory for constructing a Client for a given endpoint.
func MakeClient(ip string, port int, poc_ip string, poc_port int) (c *Client, err error) {
  c = &Client{}

	c.pending_tasks = make(chan*resdb.Transaction, 1000000)
	c.fail_task = nil
	c.done_tasks = make(chan uint64, 1000000)
  c.poc_client = resdb_client.MakePoCTransactionClient(poc_ip,poc_port)

  for i:=0; i < 100; i++ {
    go func(c * Client, ip string, port int){
         var client *resdb_client.PoCTransactionClient

         client = resdb_client.MakePoCTransactionClient(ip,port)

         loop :  for {
          var tx *resdb.Transaction
          var err error
          var tx_list []*resdb.Transaction
          //var uids map[uint64]int32
          var ok bool
          var j int

          tx_list = make([]*resdb.Transaction, 100)
          j = 0
          innerloop : for j =0; j < 100; j++ {
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
            time.Sleep(time.Duration(time.Microsecond) * 50)
            continue
          }
          //log.Printf("send uids: %d, j = %d\n",len(tx_list), j)
          err = client.SendBatchTransaction(tx_list)
          if err != nil {
            return
          }
          for _,txn := range tx_list {
            c.done_tasks <- txn.Uid
          }
        }
    }(c, ip, port)
  }

	return
}

func (c *Client) SendTransaction(tx *resdb.Transaction) {
  c.pending_tasks <- tx
  //log.Printf("send uids: txn\n")
  return
}

func (c *Client) WaitUids() (uids map[uint64]int32, err error){
  var req_uids []uint64
  var resp_uids map[uint64]int32
  var j int
  var ok bool
  var uid uint64
  //log.Printf("has fail task %d\n",c.fail_task == nil || len(c.fail_task) == 0);

  if(c.fail_task == nil || len(c.fail_task) < 100){
    req_uids = make([]uint64, 100)
      innerloop : for j =0; j < 10; j++ {
        select {
          case  uid, ok = <-c.done_tasks:
            if ok {
              //tx_list = append(tx_list, tx)
              //log.Printf("get uid??:",uid)
              req_uids[j] = uid
                continue
            } else {
              break innerloop
            }
          case <-time.After(time.Microsecond*500):
            break innerloop
          default:
              break innerloop
        }
      }
      req_uids= req_uids[0:j]
      if( len(req_uids) == 0){
        return nil, nil
      }
      if (c.fail_task != nil){
        req_uids = append(req_uids, c.fail_task...)
        c.fail_task = nil
      }
  } else {
    req_uids = c.fail_task;
    c.fail_task = nil
  }
  resp_uids, err = c.poc_client.Query(req_uids)
  if (resp_uids == nil || len(resp_uids) == 0 ){
    time.Sleep(time.Microsecond*500)
    c.fail_task = req_uids
  } else {
    //log.Printf("get resp:",resp_uids)
    c.fail_task = nil
  }
  return resp_uids, err
}

