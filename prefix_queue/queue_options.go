package prefix_queue

// QueueOptions
//  QueueMaxSize is max size of saving queue. If your consumer works unstable it's better to increase.
//  BatchLen is max size of batch being sent to repo.
//  FlushTimeoutMs if no transactions, flusher will update db even if there is less than batchLen transactions after this timeout
type QueueOptions struct {
	QueueMaxSize   int
	BatchLen       int
	FlushTimeoutMs int
}
