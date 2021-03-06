package prefix_queue

import (
	"context"
	"github.com/proxima-one/prefix-value-queue-go/prefix_queue_model"
	"github.com/proxima-one/prefix-value-queue-go/prefix_repository"
	"time"
)

type PrefixQueue struct {
	repo        prefix_repository.Repository
	queue       chan queueEntry
	prefixCache map[string]prefix_queue_model.Transaction

	combine            func(prefix_queue_model.CacheEntry, prefix_queue_model.Transaction) prefix_queue_model.Transaction
	genericTransaction func() prefix_queue_model.Transaction

	options QueueOptions
}

func NewPrefixQueue(
	repo prefix_repository.Repository,
	combine func(prefix_queue_model.CacheEntry, prefix_queue_model.Transaction) prefix_queue_model.Transaction,
	genericTransaction func() prefix_queue_model.Transaction,
	options QueueOptions) *PrefixQueue {

	return &PrefixQueue{
		repo:               repo,
		queue:              make(chan queueEntry, options.QueueMaxSize),
		prefixCache:        make(map[string]prefix_queue_model.Transaction),
		combine:            combine,
		genericTransaction: genericTransaction,
		options:            options,
	}
}

func (saver *PrefixQueue) Save(ctx context.Context, transaction prefix_queue_model.Transaction, stateId string) {
	cacheEntry := saver.getLastTransactionOfGroup(ctx, transaction.GetGroupId())
	transactionToSave := saver.combine(cacheEntry, transaction)
	saver.addToCache(transactionToSave)
	saver.addToQueue(transactionToSave, stateId)
}

func (saver *PrefixQueue) Undo(ctx context.Context, oldTransaction prefix_queue_model.Transaction, stateId string) {
	transaction := oldTransaction.GetNegative()
	cacheEntry := saver.getLastTransactionOfGroup(ctx, transaction.GetGroupId())
	transactionToSave := saver.combine(cacheEntry, transaction)
	saver.addToCache(transactionToSave)
	saver.addToQueue(transactionToSave, stateId)
}

// FlushQueue flushes queue into repo and calls onFlush. Meant to be called from other goroutine.
func (saver *PrefixQueue) FlushQueue(ctx context.Context, onFlush func(prefix_queue_model.FlushCallback)) error {
	transactions := make([]any, 0, saver.options.BatchLen)
	state := ""
	var lastTransaction prefix_queue_model.Transaction
	timeout := time.NewTicker(time.Duration(saver.options.FlushTimeoutMs) * time.Millisecond)
GetTransactionsLoop:
	for len(transactions) < saver.options.BatchLen {
		timeout.Reset(time.Duration(saver.options.FlushTimeoutMs) * time.Millisecond)
		select {
		case entry := <-saver.queue:
			transaction := entry.Transaction
			state = entry.StateId
			lastTransaction = transaction
			transactions = append(transactions, transaction)
		case <-ctx.Done():
			break GetTransactionsLoop
		case <-timeout.C:
			break GetTransactionsLoop
		}
	}
	timeout.Stop()
	if len(transactions) == 0 {
		return nil
	}

	err := saver.repo.SaveTransactions(ctx, transactions)
	if err != nil {
		return err
	}

	onFlush(prefix_queue_model.FlushCallback{
		LastState:  state,
		SavedCount: len(transactions),
		LastObject: lastTransaction,
	})

	return nil
}

func (saver *PrefixQueue) GetCacheSize() int { return len(saver.prefixCache) }

func (saver *PrefixQueue) GetQueueSize() int { return len(saver.queue) }

// Firstly tries to get from cache, then asks db.
func (saver *PrefixQueue) getLastTransactionOfGroup(ctx context.Context, groupId string) prefix_queue_model.CacheEntry {
	value, ok := saver.prefixCache[groupId]
	if ok { // found in cache
		return value.ToCacheEntry()
	}

	if !saver.repo.DoesGroupExist(ctx, groupId) { // this operation is much faster than GetLastTokenTransaction
		return saver.genericTransaction().ToCacheEntry()
	}

	// then try to find in db
	transaction := saver.genericTransaction()
	err := saver.repo.GetLastTransactionOfGroup(ctx, groupId, transaction)
	if err != nil || transaction == nil {
		return saver.genericTransaction().ToCacheEntry()
	}
	saver.addToCache(transaction) // cache of this tokenId is guaranteed to be empty
	return transaction.ToCacheEntry()
}

func (saver *PrefixQueue) addToQueue(transaction prefix_queue_model.Transaction, stateId string) {
	saver.queue <- queueEntry{transaction, stateId}
}

func (saver *PrefixQueue) addToCache(transaction prefix_queue_model.Transaction) {
	saver.prefixCache[transaction.GetGroupId()] = transaction
}
