package main

import (
	"context"
	"fmt"
	"github.com/proxima-one/prefix-value-queue-go/prefix_queue"
	"github.com/proxima-one/prefix-value-queue-go/prefix_queue_model"
)

func combine(t1 prefix_queue_model.CacheEntry, t2 prefix_queue_model.Transaction) prefix_queue_model.Transaction {
	res := *t2.(*Transfer)
	res.prefixValue = t1.(int) + t2.(*Transfer).value
	return &res
}

func genericTransfer() prefix_queue_model.Transaction {
	return &Transfer{
		id:          "",
		tokenId:     "",
		value:       0,
		prefixValue: 0,
	}
}

func main() {
	repo := NewMemoryRepo()

	opts := prefix_queue.QueueOptions{
		QueueMaxSize:   10,
		MaxRollbackLen: 10,
		BatchLen:       10,
		FlushTimeoutMs: 100,
	}
	queue := prefix_queue.NewPrefixQueue(repo, combine, genericTransfer, opts)

	queue.Save(context.Background(),
		&Transfer{
			id:      "0",
			tokenId: "0",
			value:   5,
		}, "0x0")

	queue.Save(context.Background(),
		&Transfer{
			id:      "1",
			tokenId: "0",
			value:   5,
		}, "0x1")

	queue.Save(context.Background(),
		&Transfer{
			id:      "2",
			tokenId: "1",
			value:   7,
		}, "0x2")

	queue.Save(context.Background(),
		&Transfer{
			id:      "3",
			tokenId: "0",
			value:   7,
		}, "0x3")

	queue.Undo(&Transfer{
		id:      "3",
		tokenId: "0",
		value:   7,
	}, "0x4")

	queue.Save(context.Background(),
		&Transfer{
			id:      "4",
			tokenId: "0",
			value:   8,
		}, "0x5")

	_ = queue.FlushQueue(context.Background(), func(callback prefix_queue_model.FlushCallback) {
		fmt.Printf("%+v\nLast object: %+v\n", callback, callback.LastObject)
	})

	println()
	repo.Print()
}
