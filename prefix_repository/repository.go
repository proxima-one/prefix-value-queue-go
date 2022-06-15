package prefix_repository

import (
	"context"
	"github.com/proxima-one/prefix-value-queue-go/prefix_queue_model"
)

type Repository interface {
	SaveTransactions(ctx context.Context, transactions []any) error
	DeleteTransaction(ctx context.Context, id string) error
	DoesGroupExist(ctx context.Context, groupId string) bool
	GetLastTransactionOfGroup(ctx context.Context, groupId string, result prefix_queue_model.Transaction) error
}
