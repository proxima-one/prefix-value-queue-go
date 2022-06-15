package prefix_queue_model

type FlushCallback struct {
	LastState  string
	SavedCount int
	LastObject Transaction
}
