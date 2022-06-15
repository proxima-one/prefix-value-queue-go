package main

import "github.com/proxima-one/prefix-value-queue-go/prefix_queue_model"

type Transfer struct {
	id          string
	tokenId     string
	value       int
	prefixValue int
}

func (transfer *Transfer) GetId() string { return transfer.id }

func (transfer *Transfer) GetGroupId() string { return transfer.tokenId }

func (transfer *Transfer) ToCacheEntry() prefix_queue_model.CacheEntry { return transfer.prefixValue }
