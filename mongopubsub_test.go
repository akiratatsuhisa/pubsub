package pubsub

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestMongoPubSub(t *testing.T) {
	opts := MongoPubSubOpts{
		ctx:      context.Background(),
		uri:      "",
		dbName:   "demo_pubsub",
		collName: "pubsub",
	}

	ps1 := NewMongoPubSub(&opts)
	ps2 := NewMongoPubSub(&opts)

	ch1 := make(chan []byte)
	ch2 := make(chan []byte)

	un1 := ps1.Subscribe("test_chan", ch1)
	defer un1()

	un2 := ps2.Subscribe("test_chan", ch2)
	defer un2()

	ps1.Publish("test_chan", &TestData{
		Name:    "Test",
		Content: "Content",
	})

	dataB1 := <-ch1
	dataB2 := <-ch2

	var data1, data2 TestData
	_ = bson.Unmarshal(dataB1, &data1)
	_ = bson.Unmarshal(dataB2, &data2)

	if data1.Name != data2.Name {
		t.Fatalf("Error")
	}
}
