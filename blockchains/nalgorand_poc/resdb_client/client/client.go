package algorand_poc_client

import (
	//"fmt"
	//"log"
  //"time"

  "github.com/resilientdb/go-resilientdb-sdk/client"
)

type Client struct {
  poc_client *resdb_client.PoCTransactionClient
}

// MakeClient is the factory for constructing a Client for a given endpoint.
func MakeClient(poc_ip string, poc_port int) (c *Client, err error) {
  c = &Client{}
  c.poc_client = resdb_client.MakePoCTransactionClient(poc_ip,poc_port)
	return
}

func (c *Client) WaitUids(req_uids []uint64) (map[uint64]int32, error){
  return c.poc_client.Query(req_uids)
}

