package prefix_queue

import "github.com/proxima-one/prefix-value-queue-go/prefix_queue_model"

type queueEntry struct {
	Transaction prefix_queue_model.Transaction
	StateId     string
}
