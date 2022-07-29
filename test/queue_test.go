package test

import (
	"context"
	"github.com/proxima-one/prefix-value-queue-go/prefix_queue"
	"github.com/proxima-one/prefix-value-queue-go/prefix_queue_model"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

type Data struct {
	Transfer Transfer
	Undo     bool
}

func TestQueueWithMemoryRepo(t *testing.T) {
	repo := NewMemoryRepo()
	queue := prefix_queue.NewPrefixQueue(
		repo,
		func(t1 prefix_queue_model.CacheEntry, t2 prefix_queue_model.Transaction) prefix_queue_model.Transaction {
			res := *t2.(*Transfer)
			res.prefixValue = t1.(*Transfer).prefixValue
			res.prefixValue += t2.(*Transfer).value
			return &res
		},
		func() prefix_queue_model.Transaction {
			return &Transfer{
				id:          "",
				tokenId:     "",
				value:       0,
				prefixValue: 0,
			}
		}, prefix_queue.QueueOptions{
			QueueMaxSize:   50,
			BatchLen:       500,
			FlushTimeoutMs: 100,
		})

	groupsNumber := 20

	data := make([]Data, 0)
	actData := make([]Data, 0)
	for i := 0; i < 10000; i++ {
		var dat Data
		if len(actData) > 0 && rand.Intn(2) == 1 {
			dat.Transfer = actData[len(actData)-1].Transfer
			dat.Undo = true
			actData = actData[:len(actData)-1]
		} else {
			dat.Transfer = Transfer{
				id:          strconv.Itoa(i),
				tokenId:     strconv.Itoa(rand.Intn(groupsNumber)),
				value:       rand.Intn(1000),
				prefixValue: 0,
			}
			dat.Undo = false
			actData = append(actData, dat)
		}
		data = append(data, dat)
		//println(dat.Transfer.id, dat.Undo)
	}

	//println("===")

	cache := make(map[string]int)
	for i := range actData {
		//println(actData[i].Transfer.id, actData[i].Undo)
		actData[i].Transfer.prefixValue = actData[i].Transfer.value + cache[actData[i].Transfer.tokenId]
		cache[actData[i].Transfer.tokenId] = actData[i].Transfer.prefixValue
	}

	//t.Logf("%+v\n", data)
	//t.Logf("%+v\n", actData)

	totalSaved := 0
	ex := make(chan struct{}, 1)
	ex2 := make(chan struct{}, 1)
	go func() {
		for {
			if len(ex) > 0 {
				break
			}
			ctx, _ := context.WithTimeout(context.Background(), 15*time.Second)
			err := queue.FlushQueue(ctx, func(c prefix_queue_model.FlushCallback) { totalSaved += c.SavedCount })
			if err != nil {
				t.Error(err.Error())
			}
		}
		ex2 <- struct{}{}
		println("exiting")
	}()

	for _, tr := range data {
		transfer := tr.Transfer
		if tr.Undo {
			queue.Undo(context.Background(), &transfer, "0")
		} else {
			queue.Save(context.Background(), &transfer, "0")
		}
	}

	for queue.GetQueueSize() > 0 {
		//_ = queue.FlushQueue(context.Background(), func(c prefix_queue_model.FlushCallback) {})
	}
	ex <- struct{}{}
	<-ex2
	println(totalSaved, len(repo.Transfers), len(actData))
	//if len(actData) != len(repo.Transfers) {
	//	t.FailNow()
	//}

	//for _, tr := range repo.Transfers {
	//	t.Logf("%+v\n", tr)
	//}

	for _, tr := range actData {
		_, ok := repo.Transfers[tr.Transfer.id]
		if !ok {
			t.Fatalf("%+v\n", tr.Transfer)
		}
		if repo.Transfers[tr.Transfer.id].prefixValue != tr.Transfer.prefixValue {
			t.Errorf("%+v %+v\n", repo.Transfers[tr.Transfer.id], tr.Transfer)
		}
	}

}
