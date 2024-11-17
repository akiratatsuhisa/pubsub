package pubsub

import (
	"context"
	"log"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type PubSubDocument struct {
	TriggerName string      `bson:"trigger_name"`
	Data        interface{} `bson:"data"`
	Timestamp   time.Time   `bson:"timestamp"`
}

type MongoPubSub struct {
	mu         sync.Mutex
	events     map[string][]chan []byte
	client     *mongo.Client
	collection *mongo.Collection
}

type MongoPubSubOpts struct {
	ctx      context.Context
	uri      string
	dbName   string
	collName string
}

func NewMongoPubSub(opts *MongoPubSubOpts) *MongoPubSub {
	client, err := mongo.Connect(options.Client().ApplyURI(opts.uri))
	if err != nil {
		panic(err)
	}

	collection := client.Database(opts.dbName).Collection(opts.collName)

	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "timestamp", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(3600),
	}
	_, err = collection.Indexes().CreateOne(opts.ctx, indexModel)
	if err != nil {
		panic(err)
	}

	instance := MongoPubSub{events: make(map[string][]chan []byte), client: client, collection: collection}

	instance.watch(opts.ctx)

	return &instance
}

func (ps *MongoPubSub) sendToListener(change bson.M) {
	fullDocument, err := bson.Marshal(change["fullDocument"])
	if err != nil {
		log.Fatalf("can't get full document")
		return
	}

	var document PubSubDocument
	err = bson.Unmarshal(fullDocument, &document)
	if err != nil {
		log.Fatalf("parsing error")
		return
	}

	dataBytes, err := bson.Marshal(document.Data)
	if err != nil {
		log.Fatalf("can't get document's data")
		return
	}

	for _, listener := range ps.events[document.TriggerName] {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Println("Recovered from panic:", r)
				}
			}()

			listener <- dataBytes
		}()
	}
}

func (ps *MongoPubSub) watch(ctx context.Context) {
	changeStream, err := ps.collection.Watch(ctx, mongo.Pipeline{})

	if err != nil {
		panic(err)
	}

	go func() {
		for changeStream.Next(ctx) {
			var change bson.M
			if err := changeStream.Decode(&change); err != nil {
				log.Fatal(err)
			} else {
				go ps.sendToListener(change)
			}
		}
	}()
}

func (ps *MongoPubSub) Subscribe(triggerName string, ch chan []byte) func() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.events[triggerName] = append(ps.events[triggerName], ch)

	return func() {
		ps.Unsubscribe(triggerName, ch)
	}
}

func (ps *MongoPubSub) Unsubscribe(triggerName string, ch chan []byte) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	for index, listener := range ps.events[triggerName] {
		if listener == ch {
			ps.events[triggerName] = append(ps.events[triggerName][:index], ps.events[triggerName][index+1:]...)
			close(listener)
		}
	}
}

func (ps *MongoPubSub) Publish(triggerName string, data interface{}) error {
	document := &PubSubDocument{
		TriggerName: triggerName,
		Data:        data,
		Timestamp:   time.Now(),
	}

	_, err := ps.collection.InsertOne(context.TODO(), document)

	return err
}
