package prefix_queue_model

// QueueOptions
//  QueueMaxSize is max size of saving queue. If your consumer works unstable it's better to increase.
//  MaxRollbackLen is max size of transaction cache. There mustn't be a sequence of transactions with {Undos - Saves > MaxRollbackLen}.
//  BatchLen is max size of batch being sent to repo.
type QueueOptions struct {
	QueueMaxSize   int
	MaxRollbackLen int
	BatchLen       int
	FlushTimeoutMs int
}
