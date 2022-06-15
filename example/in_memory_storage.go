package main

import (
	"context"
	"fmt"
	"github.com/proxima-one/prefix-value-queue-go/prefix_queue_model"
	"sort"
	"sync"
)

type MemoryRepo struct {
	Transfers map[string]*Transfer
	mt        sync.RWMutex
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{Transfers: make(map[string]*Transfer)}
}

func (repo *MemoryRepo) SaveTransactions(ctx context.Context, transactions []any) error {
	repo.mt.Lock()
	defer repo.mt.Unlock()
	for _, tr := range transactions {
		transfer := tr.(*Transfer)
		repo.Transfers[transfer.id] = transfer
	}
	return nil
}

func (repo *MemoryRepo) DeleteTransaction(ctx context.Context, id string) error {
	repo.mt.Lock()
	defer repo.mt.Unlock()
	delete(repo.Transfers, id)
	return nil
}

func (repo *MemoryRepo) DoesGroupExist(ctx context.Context, groupId string) bool {
	repo.mt.RLock()
	defer repo.mt.RUnlock()
	for _, transfer := range repo.Transfers {
		if transfer.tokenId == groupId {
			return true
		}
	}
	return false
}

func (repo *MemoryRepo) GetLastTransactionOfGroup(ctx context.Context, groupId string, result prefix_queue_model.Transaction) error {
	repo.mt.RLock()
	defer repo.mt.RUnlock()
	keys := make([]string, 0)
	for k, _ := range repo.Transfers {
		keys = append(keys, k)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))
	for _, k := range keys {
		if repo.Transfers[k].tokenId == groupId {
			result = repo.Transfers[k]
			break
		}
	}
	return nil
}

func (repo *MemoryRepo) Print() {
	repo.mt.RLock()
	defer repo.mt.RUnlock()
	keys := make([]string, 0)
	for k, _ := range repo.Transfers {
		keys = append(keys, k)
	}
	sort.Sort(sort.StringSlice(keys))
	for _, k := range keys {
		fmt.Printf("%+v\n", repo.Transfers[k])
	}
}
