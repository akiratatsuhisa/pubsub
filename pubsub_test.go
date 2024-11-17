package pubsub

import (
	"testing"
)

type TestData struct {
	Name    string `bson:"name"`
	Content string `bson:"content"`
}

func TestPubSub(t *testing.T) {
	ps := NewPubSub()

	ch1 := make(chan interface{})
	ch2 := make(chan interface{})

	un1 := ps.Subscribe("test_chan", ch1)
	defer un1()

	un2 := ps.Subscribe("test_chan", ch2)
	defer un2()

	ps.Publish("test_chan", &TestData{
		Name:    "Test",
		Content: "Content",
	})

	data1 := <-ch1
	data2 := <-ch2

	if data1 != data2 {
		t.Fatalf("Error")
	}
}
