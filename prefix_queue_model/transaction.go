package prefix_queue_model

type Transaction interface {
	GetId() string
	GetGroupId() string
	ToCacheEntry() CacheEntry
}
