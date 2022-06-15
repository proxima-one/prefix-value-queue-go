# Prefix value queue helper for Golang
This library helps to continuously update prefix values over independent 
groups in some stream efficiently handling 
Save & Undo operations and sending transactions in batches.

For example, you can calculate prefix sums of all transactions of some token and get 
its trading volume in some time frame using only 2 simple DB queries. </br>
Another example is to count NFT's owned by some account. 
Or again count number of NFT's transfers of some collection.

Examples can be found in `./test` and `./example` folders 
and at https://github.com/proxima-one/token-volume-apis

## Concept
### Group
In this library we use concept called Group. It's how you split transactions in independent (in terms of
calculating prefix data) groups. 

If we calculate prefix trading volumes, we probably want to count trading volume of each token independently
so our Groups can be different tokens, and we can use `TokenId` as `GroupId` here. 

And if we calculate NFT's owned by some account our groups will be 
different accounts so `GroupId` can be `AccoundId`.

### Transaction
Transaction must implement the next interface:
```
type Transaction interface {
    GetId() string
    GetGroupId() string
    ToCacheEntry() CacheEntry
}
```
`GetId` is used for accessing transactions in database. This func should return exactly field that is tagged with `bson:_id`.</br>
`GetGroupId` should return GroupId of Transaction. <i>For trading volumes it's just `return this.TokenId`.</i></br>
`ToCacheEntry` returns `CacheEntry` (explained below). Cache entry should contain all Transaction fields that are used 
to calculate next prefix value from previous. 

### Cache entry
Declared as 'any': `type CacheEntry any`. </br> 
To support saving queue, undos and decrease number of read queries in database 
at the same time there is Cache implemented. </br>
It is used when you add new transaction
so you first need to get last saved transaction of same group to calculate new prefix data.</br>

`Cache entry` is to decrease memory usage of cache by caching only necessary data (not entire Transactions). </br>
You should put in `CacheEntry` only information that is used in `combine` function.

For example, if you count transfer volumes, you should put in cache entry only last prefix sum because 
you will need it to calculate next value as `lastPrefix + curValue`. You don't need to save transaction id
or its value etc. in cache entry because it won't be ever used.

### Repository
Should implement the next interface:
```
type Repository interface {
    SaveTransactions(ctx context.Context, transactions []any) error
    DeleteTransaction(ctx context.Context, id string) error
    DoesGroupExist(ctx context.Context, groupId string) bool
    GetLastTransactionOfGroup(ctx context.Context, groupId string, result Transaction) error
}
```
When using Mongo there is implementation of Repository called `PrefixMongoRepository` over:
```
type BasicMongoRepository interface {
    GetCollection(string) *mongo.Collection
}
```
So you can implement only `GetCollection(string)` method in your Mongo and create `PrefixMongoRepository`
that extends `BasicMongoRepository` to `Repository`.

<b>It is HIGHLY recommended to have `group_id_1_timestamp_-1` index in your MongoDB.</b>

### Combine function
Combine function is used to calculate new prefix value over last value and new transaction:</br>
`func(CacheEntry, Transaction) Transaction`

### Generic Transaction
`GenericTransaction()` function should return empty (zero prefix values etc.) 
`Transaction` that is generic for new group and is being passed to `GetLastTransactionOfGroup` method
of Repository.

## Usage
Let's implement trading volumes over time task.

Here is out simple `Transfer`:
```
type Transfer struct {
	id          string
	tokenId     string
	value       int
	prefixValue int
}
```
And its corresponding methods:
```
func (transfer *Transfer) GetId() string { return transfer.id }
func (transfer *Transfer) GetGroupId() string { return transfer.tokenId }
func (transfer *Transfer) ToCacheEntry() prefix_queue_model.CacheEntry { return transfer.prefixValue }
```
Our GroupId is `tokenId` because we want to get trading volumes of each token independently.

And to calculate next prefix value we only need the last prefix sum and current value so 
in this case `CacheEntry` is just an `int`. By the word let's implement `combine` function:
```
func combine(t1 prefix_queue_model.CacheEntry, t2 prefix_queue_model.Transaction) prefix_queue_model.Transaction {
	res := *t2.(*Transfer)
	res.prefixValue = t1.(int) + t2.(*Transfer).value
	return &res
}
```

Generic transfer will just return completely empty transfer as its value and prefixValue 
are zeroes by default.

Assuming that we have some simple in-memory database we can easily create queue:
```
opts := prefix_queue.QueueOptions{
    QueueMaxSize:   10,
    MaxRollbackLen: 10,
    BatchLen:       10,
    FlushTimeoutMs: 100,
}
queue := prefix_queue.NewPrefixQueue(repo, combine, genericTransfer, opts)
```

Now we can use two main endpoints of queue to save and undo transfers:
```
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
```

Full code with outputs can be found in `./example` folder.