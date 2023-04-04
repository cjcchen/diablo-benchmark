package nresilientdb


import (
	"bytes"
	"context"
	"diablo-benchmark/core"
	"sync"
	"log"
	"diablo-benchmark/blockchains/nresilientdb/resdb_client/client"
)


type BlockchainClient struct {
	logger     core.Logger
	client    *resdb.Client
	preparer   transactionPreparer
	confirmer  transactionConfirmer
}

func newClient(logger core.Logger, client *resdb.Client, preparer transactionPreparer, confirmer transactionConfirmer) *BlockchainClient {
	return &BlockchainClient{
		logger: logger,
		client: client,
		preparer: preparer,
		confirmer: confirmer,
	}
}

func (this *BlockchainClient) DecodePayload(encoded []byte) (interface{}, error) {
	var buffer *bytes.Buffer = bytes.NewBuffer(encoded)
	var tx transaction
	var err error

	tx, err = decodeTransaction(buffer)
	if err != nil {
		log.Printf("decode pay load fail")
		return nil, err
	}

	this.logger.Tracef("decode transaction %d", tx.getUid())

	err = this.preparer.prepare(tx)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func (this *BlockchainClient) TriggerInteraction(iact core.Interaction) error {
	var tx transaction
	var txid uint64
	var stx *resdb.Transaction
	var err error

	tx = iact.Payload().(transaction)

	this.logger.Tracef("schedule transaction %d", tx.getUid())

	stx, err = tx.getTx()
	if err != nil {
		return err
	}

	this.logger.Tracef("submit transaction %d", tx.getUid())

	//log.Printf("report txn submit")
	iact.ReportSubmit()

	this.client.SendTransaction(stx)
	if err != nil {
		iact.ReportAbort()
		return err
	}
	txid = tx.getUid()

	return this.confirmer.confirm(iact, txid)
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
	logger  core.Logger
}

func newSignatureTransactionPreparer(logger core.Logger) transactionPreparer {
	return &signatureTransactionPreparer{
		logger: logger,
	}
}

func (this *signatureTransactionPreparer) prepare(tx transaction) error {
	var err error

	this.logger.Tracef("sign transaction %d", tx.getUid())
	_, err = tx.getTx()
	if err != nil {
		return err
	}

	return nil
}


type transactionConfirmer interface {
	confirm(core.Interaction, uint64) error
}

type pollblkTransactionConfirmer struct {
	logger    core.Logger
	client    *resdb.Client
	ctx       context.Context
	err       error
	lock      sync.Mutex
	pendings  map[uint64]*pollblkTransactionConfirmerPending
}

type pollblkTransactionConfirmerPending struct {
	channel  chan<- error
	iact     core.Interaction
}

func newPollblkTransactionConfirmer(logger core.Logger, client *resdb.Client, ctx context.Context) *pollblkTransactionConfirmer {
	var this pollblkTransactionConfirmer

	this.logger = logger
	this.client = client
	this.ctx = ctx
	this.err = nil
	this.pendings = make(map[uint64]*pollblkTransactionConfirmerPending)

	go this.run()

	return &this
}

func (this *pollblkTransactionConfirmer) confirm(iact core.Interaction, txid uint64) error {
	var tx transaction = iact.Payload().(transaction)
	var pending *pollblkTransactionConfirmerPending
	var uid uint64 = tx.getUid()
	var channel chan error
	var done bool

	channel = make(chan error)

	pending = &pollblkTransactionConfirmerPending{
		channel: channel,
		iact: iact,
	}

	this.lock.Lock()

	if this.pendings == nil {
		done = true
	} else {
		this.pendings[uid] = pending
		done = false
	}

	this.lock.Unlock()

	if done {
		close(channel)
		return this.err
	} else {
		return <- channel
	}
}

func (this *pollblkTransactionConfirmer) reportTransactions(uid uint64) {
	var pending *pollblkTransactionConfirmerPending
	var ok bool

	this.lock.Lock()

	pending, ok = this.pendings[uid]
	if !ok {
		return
	}

	delete(this.pendings, uid)

	this.lock.Unlock()

	//log.Printf("report txn commit")
	pending.iact.ReportCommit()

	this.logger.Tracef("transaction %d committed",
	pending.iact.Payload().(transaction).getUid())

	pending.channel <- nil

	close(pending.channel)
}

func (this *pollblkTransactionConfirmer) flushPendings(err error) {
	var pendings []*pollblkTransactionConfirmerPending
	var pending *pollblkTransactionConfirmerPending

	pendings = make([]*pollblkTransactionConfirmerPending, 0)

	this.lock.Lock()

	for _, pending = range this.pendings {
		pendings = append(pendings, pending)
	}

	this.pendings = nil
	this.err = err

	this.lock.Unlock()

	for _, pending = range pendings {
		pending.iact.ReportAbort()

		this.logger.Tracef("transaction %d aborted",
			pending.iact.Payload().(transaction).getUid())

		pending.channel <- err

		close(pending.channel)
	}
}

func (this *pollblkTransactionConfirmer) run() {
	var client *resdb.Client = this.client
	var uid uint64
	var err error

	loop: for {
    uid, err = client.WaitNextTxn()
      if err != nil {
        log.Printf("get uid %d err %s\n",uid, err)
          if this.ctx.Err() != nil {
            break loop
          }
          continue
      }
    this.reportTransactions(uid)
	}

  this.flushPendings(err)

}

