package ndiem_poc

import (
	"bytes"
	"diablo-benchmark/core"
	"encoding/hex"
	"fmt"
	"sync"
  "log"
	"time"

	"github.com/diem/client-sdk-go/diemclient"
	"github.com/diem/client-sdk-go/diemjsonrpctypes"
	"github.com/diem/client-sdk-go/diemtypes"
	"diablo-benchmark/blockchains/ndiem_poc/resdb_client/client"
)

type BlockchainClient struct {
	logger    core.Logger
	client    diemclient.Client
	preparer  transactionPreparer
	confirmer transactionConfirmer
}

func newClient(logger core.Logger, client diemclient.Client, preparer transactionPreparer, confirmer transactionConfirmer) *BlockchainClient {
	return &BlockchainClient{
		logger:    logger,
		client:    client,
		preparer:  preparer,
		confirmer: confirmer,
	}
}

func (this *BlockchainClient) DecodePayload(encoded []byte) (interface{}, error) {
	var buffer *bytes.Buffer = bytes.NewBuffer(encoded)
	var tx transaction
	var err error

	tx, err = decodeTransaction(buffer)
	if err != nil {
		return nil, err
	}

	this.logger.Tracef("decode transaction %s", tx.getName())

	err = this.preparer.prepare(tx)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func (this *BlockchainClient) TriggerInteraction(iact core.Interaction) error {
	var stx *diemtypes.SignedTransaction
	var tx transaction
	var err error

	tx = iact.Payload().(transaction)

	this.logger.Tracef("schedule transaction %s", tx.getName())

	stx, err = tx.getSigned()
	if err != nil {
		return err
	}

	this.confirmer.prepare(iact)

	this.logger.Tracef("submit transaction %s", tx.getName())

	iact.ReportSubmit()

	err = this.client.SubmitTransaction(stx)
	if err != nil {
		return fmt.Errorf("transaction %s failed (%s)", tx.getName(),
			err.Error())
	}

	return this.confirmer.confirm(iact)
}

type transactionPreparer interface {
	prepare(transaction) error
}

type nothingTransactionPreparer struct {
}

func newNothingTransactionPreparer() transactionPreparer {
	return &nothingTransactionPreparer{}
}

func (this *nothingTransactionPreparer) prepare(transaction) error {
	return nil
}

type signatureTransactionPreparer struct {
	logger core.Logger
}

func newSignatureTransactionPreparer(logger core.Logger) transactionPreparer {
	return &signatureTransactionPreparer{
		logger: logger,
	}
}

func (this *signatureTransactionPreparer) prepare(tx transaction) error {
	var err error

	this.logger.Tracef("sign transaction %s", tx.getName())

	_, err = tx.getSigned()
	if err != nil {
		return err
	}

	return nil
}

type transactionConfirmer interface {
	prepare(core.Interaction)
	confirm(core.Interaction) error
}

type polltxTransactionConfirmer struct {
	logger core.Logger
	client diemclient.Client
	mwait  time.Duration
}

func newPolltxTransactionConfirmer(logger core.Logger, client diemclient.Client) *polltxTransactionConfirmer {
	return &polltxTransactionConfirmer{
		logger: logger,
		client: client,
		mwait:  30 * time.Second,
	}
}

func (this *polltxTransactionConfirmer) prepare(core.Interaction) {
}

func (this *polltxTransactionConfirmer) confirm(iact core.Interaction) error {
	var state *diemjsonrpctypes.Transaction
	var stx *diemtypes.SignedTransaction
	var tx transaction
	var err error

	tx = iact.Payload().(transaction)

	stx, err = tx.getSigned()
	if err != nil {
		return err
	}

	state, err = this.client.WaitForTransaction2(stx, this.mwait)
	if err != nil {
		return err
	}

	if state.VmStatus.GetType() != "executed" {
		iact.ReportAbort()
		this.logger.Tracef("transaction %s failed (%s)", tx.getName(),
			state.VmStatus.GetType())
		return nil
	}

	iact.ReportCommit()
	this.logger.Tracef("transaction %s committed", tx.getName())
	return nil
}

type pollblkTransactionConfirmer struct {
	logger   core.Logger
	client   diemclient.Client
	err      error
	lock     sync.Mutex
	pendings map[pollblkTransactionConfirmerKey]*pollblkTransactionConfirmerPending
  poc_client *diem_poc_client.Client
}

type pollblkTransactionConfirmerKey struct {
	sender   diemtypes.AccountAddress
	sequence uint64
}

type pollblkTransactionConfirmerPending struct {
	channel chan error
	iact    core.Interaction
}

func newPollblkTransactionConfirmer(logger core.Logger, client diemclient.Client, poc_client *diem_poc_client.Client) *pollblkTransactionConfirmer {
	var this pollblkTransactionConfirmer

	this.logger = logger
	this.client = client
	this.err = nil
	this.pendings = make(map[pollblkTransactionConfirmerKey]*pollblkTransactionConfirmerPending)
  this.poc_client = poc_client

	go this.run()

	return &this
}

func (this *pollblkTransactionConfirmer) prepare(iact core.Interaction) {
	var tx transaction = iact.Payload().(transaction)
	var value *pollblkTransactionConfirmerPending
	var key pollblkTransactionConfirmerKey
	var stx *diemtypes.SignedTransaction
	var channel chan error

	stx, _ = tx.getSigned()

	channel = make(chan error)

	key = pollblkTransactionConfirmerKey{
		sender:   stx.RawTxn.Sender,
		sequence: stx.RawTxn.SequenceNumber,
	}

	value = &pollblkTransactionConfirmerPending{
		channel: channel,
		iact:    iact,
	}

	this.lock.Lock()

	if this.pendings != nil {
		this.pendings[key] = value
	}

	this.lock.Unlock()
}

func (this *pollblkTransactionConfirmer) confirm(iact core.Interaction) error {
	var tx transaction = iact.Payload().(transaction)
	var value *pollblkTransactionConfirmerPending
	var key pollblkTransactionConfirmerKey
	var stx *diemtypes.SignedTransaction

	stx, _ = tx.getSigned()

	key = pollblkTransactionConfirmerKey{
		sender:   stx.RawTxn.Sender,
		sequence: stx.RawTxn.SequenceNumber,
	}

	this.lock.Lock()

	if this.pendings == nil {
		value = nil
	} else {
		value = this.pendings[key]
	}

	this.lock.Unlock()

	if value == nil {
		return this.err
	} else {
		return <-value.channel
	}
}

func (this *pollblkTransactionConfirmer) parseTransaction(tx *diemjsonrpctypes.Transaction) {
	var pending *pollblkTransactionConfirmerPending
	var key pollblkTransactionConfirmerKey
	var account diemtypes.AccountAddress
	var sender []byte
	var err error
	var ok bool
	var i int

	sender, err = hex.DecodeString(tx.Transaction.Sender)
	if err != nil {
		return
	}

	for i = range sender {
		account[i] = sender[i]
	}

	key = pollblkTransactionConfirmerKey{
		sender:   account,
		sequence: tx.Transaction.SequenceNumber,
	}

	this.lock.Lock()

	pending, ok = this.pendings[key]
	if ok {
		delete(this.pendings, key)
	}

	this.lock.Unlock()

	if !ok {
    log.Print("txn not exist")
		return
	}

	pending.iact.ReportCommit()

	this.logger.Tracef("transaction %s committed",
		pending.iact.Payload().(transaction).getName())
  //log.Printf("get versioin", tx.Version)
  //log.Printf("get sender:",tx.Transaction.Sender)
  //log.Printf("get seq:",tx.Transaction.SequenceNumber)
  //log.Printf("get tx:",tx)

	pending.channel <- nil

	close(pending.channel)
}

func (this *pollblkTransactionConfirmer) run() {
	var txs []*diemjsonrpctypes.Transaction
	var tx *diemjsonrpctypes.Transaction
	var meta *diemjsonrpctypes.Metadata
	var v, version uint64
	var err error
  var uid_list []uint64
  var fail_list []*diemjsonrpctypes.Transaction
  var resp_list map[uint64]int32
  var ok bool

  fail_list = nil
  resp_list=make(map[uint64]int32)

	meta, err = this.client.GetMetadata()
	if err != nil {
		this.logger.Errorf("get meta: %s", err.Error())
		return
	}

	v = meta.Version
 // log.Printf("get meta v:",v)

	for {
		meta, err = this.client.GetMetadata()
		if err != nil {
			this.logger.Errorf("get meta: %s", err.Error())
			return
		}

		version = meta.Version

		for v < version {
			v += 1

			txs, err = this.client.GetTransactions(v, 100, true)
			if err != nil {
				continue
			}

      //log.Printf("fail list:",len(fail_list))
      txs = append(txs, fail_list...)
      log.Print("get txs:",len(txs))

      uid_list = make([]uint64, len(txs))
      idx := 0
			for _, tx = range txs {
				if tx.Transaction.Type != "user" {
					continue
				}
        uid_list[idx] = tx.Version
        idx=idx+1
      }

      //log.Printf("uid_lists:",uid_list)

      if (len(uid_list) > 0) {
        resp_list, err = this.poc_client.WaitUids(uid_list)
          if (err != nil) {
            log.Print("fail")
            time.Sleep(time.Second)
            v = v-1
            continue
          }
      }

      if (len(resp_list)==0) {
            time.Sleep(time.Second)
            v = v-1
            continue
      }

      log.Printf("get uid_lists:",len(uid_list), len(resp_list))
      fail_list = nil

			for _, tx = range txs {
				if tx.Version > v {
					v = tx.Version
				}

				if tx.Transaction.Type != "user" {
					continue
				}
        _, ok = resp_list[tx.Version];
        if(!ok) {
          fail_list = append(fail_list, tx)
          continue
        }
        //log.Printf("find version:",tx.Version)
        //log.Printf("ok:",ok)
				this.parseTransaction(tx)
			}
		}
	}
}
