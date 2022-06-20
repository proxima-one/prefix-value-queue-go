package prefix_repository

import (
	"context"
	"github.com/proxima-one/prefix-value-queue-go/prefix_queue_model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type BasicMongoRepository interface {
	GetCollection(string) *mongo.Collection
}

type PrefixMongoRepository struct {
	repo                  BasicMongoRepository
	transfersCollectionId string
	groupBsonTag          string
	timestampBsonTag      string
}

func NewPrefixMongoRepository(
	ctx context.Context, repo BasicMongoRepository,
	transfersCollectionId, groupBsonTag, timestampBsonTag string) (*PrefixMongoRepository, error) {

	prefixRepo := &PrefixMongoRepository{
		repo:                  repo,
		transfersCollectionId: transfersCollectionId,
		groupBsonTag:          groupBsonTag,
		timestampBsonTag:      timestampBsonTag,
	}
	return prefixRepo, prefixRepo.checkIndex(ctx)
}

func (repo *PrefixMongoRepository) checkIndex(ctx context.Context) error {
	cur, err := repo.getCollection().Indexes().List(ctx)
	if err != nil {
		return err
	}
	var v []struct {
		Name string
	}
	err = cur.All(ctx, &v)
	if err != nil {
		return err
	}
	ind := repo.groupBsonTag + "_1_" + repo.timestampBsonTag + "_-1"
	for _, existing := range v {
		if existing.Name == ind {
			return nil
		}
	}
	println("WARN: no index", ind, "in collection", repo.transfersCollectionId)
	return nil
}

func (repo *PrefixMongoRepository) getCollection() *mongo.Collection {
	return repo.repo.GetCollection(repo.transfersCollectionId)
}

func (repo *PrefixMongoRepository) SaveTransactions(ctx context.Context, transactions []any) error {
	opts := options.InsertMany().SetOrdered(false)
	collection := repo.getCollection()
	_, err := collection.InsertMany(ctx, transactions, opts)
	if err != nil {
		switch err.(type) {
		case mongo.BulkWriteException:
			for _, obj := range err.(mongo.BulkWriteException).WriteErrors {
				if obj.Code == 11000 { // skip duplicates errors
					continue
				}
				return err
			}
		default:
			return err
		}
	}
	return nil
}

func (repo *PrefixMongoRepository) DeleteTransaction(ctx context.Context, id string) error {
	err := repo.getCollection().FindOneAndDelete(ctx, bson.M{"_id": id}).Err()
	if err != nil {
		return err
	}
	return nil
}

func (repo *PrefixMongoRepository) DoesGroupExist(ctx context.Context, groupId string) bool {
	res := repo.getCollection().FindOne(ctx, bson.M{repo.groupBsonTag: groupId})
	if res.Err() != nil || res == nil {
		return false
	}
	return true
}

func (repo *PrefixMongoRepository) GetLastTransactionOfGroup(ctx context.Context, groupId string, result prefix_queue_model.Transaction) error {
	opts := options.Find().SetSort(bson.D{{repo.timestampBsonTag, -1}}).SetLimit(1)

	cursor, err := repo.getCollection().Find(ctx, bson.D{{repo.groupBsonTag, groupId}}, opts)
	if err != nil {
		return err
	}
	cursor.Next(ctx)
	err = cursor.Decode(result)

	if err != nil {
		return err
	}
	return nil
}

func (repo *PrefixMongoRepository) GetGroupTransactionsInBounds(
	ctx context.Context,
	groupId string,
	start time.Time,
	end time.Time,
	res1 prefix_queue_model.Transaction,
	res2 prefix_queue_model.Transaction) error {

	opts := options.Find().SetSort(bson.D{{repo.timestampBsonTag, -1}}).SetLimit(1)

	cursor, err := repo.getCollection().Find(
		ctx, bson.D{
			{"$and", []interface{}{
				bson.D{{repo.timestampBsonTag, bson.D{{"$lt", start}}}},
				bson.D{{repo.groupBsonTag, groupId}},
			}},
		}, opts)
	if err != nil {
		return err
	}

	cursor.Next(ctx)
	err = cursor.Decode(res1)

	cursor, err = repo.getCollection().Find(
		ctx, bson.D{
			{"$and", []interface{}{
				bson.D{{repo.timestampBsonTag, bson.D{{"$lt", end}}}},
				bson.D{{repo.groupBsonTag, groupId}},
			}},
		}, opts)
	if err != nil {
		return err
	}
	cursor.Next(ctx)
	err = cursor.Decode(res2)
	if err != nil {
		return err
	}

	return nil
}
