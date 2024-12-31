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

// PubSubDocument represents the structure of the document stored in MongoDB.
type PubSubDocument struct {
	TriggerName string      `bson:"trigger_name"` // Name of the event trigger
	Data        interface{} `bson:"data"`         // Event data
	Timestamp   time.Time   `bson:"timestamp"`    // Timestamp for TTL indexing
}

// MongoPubSub handles Pub/Sub functionality using MongoDB as a backend.
type MongoPubSub struct {
	mu         sync.Mutex               // Mutex to handle concurrent access
	events     map[string][]chan []byte // Map of event names to subscribed channels
	client     *mongo.Client            // MongoDB client
	collection *mongo.Collection        // MongoDB collection used for Pub/Sub
}

// MongoPubSubOpts holds configuration options for MongoPubSub.
type MongoPubSubOpts struct {
	Ctx        context.Context // Context for MongoDB operations
	Uri        string          // MongoDB connection URI
	DbName     string          // Name of the database
	CollName   string          // Name of the collection
	TtlSeconds int32           // TTL (Time-To-Live) for events in seconds
}

// NewMongoPubSub initializes a MongoPubSub instance with the given options.
func NewMongoPubSub(opts *MongoPubSubOpts) *MongoPubSub {
	// Connect to MongoDB
	client, err := mongo.Connect(options.Client().ApplyURI(opts.Uri))
	if err != nil {
		panic(err)
	}

	collection := client.Database(opts.DbName).Collection(opts.CollName)

	// Default TTL to 3600 seconds (1 hour) if not provided
	if opts.TtlSeconds <= 0 {
		opts.TtlSeconds = 3600
	}

	// Create TTL index on the "timestamp" field
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "timestamp", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(opts.TtlSeconds),
	}
	_, err = collection.Indexes().CreateOne(opts.Ctx, indexModel)
	if err != nil {
		panic(err)
	}

	// Initialize MongoPubSub and start watching for changes
	instance := MongoPubSub{events: make(map[string][]chan []byte), client: client, collection: collection}
	instance.watch(opts.Ctx)

	return &instance
}

// sendToListener sends data to all subscribers of a specific trigger.
func (ps *MongoPubSub) sendToListener(change bson.M) {
	if change["operationType"].(string) != "insert" {
		return
	}

	// Extract and unmarshal the full document from the change stream
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

	// Marshal the event data to bytes for delivery
	dataBytes, err := bson.Marshal(document.Data)
	if err != nil {
		log.Fatalf("can't get document's data")
		return
	}

	// Notify all subscribers of the event
	for _, listener := range ps.events[document.TriggerName] {
		go func() {
			// Recover in case a channel is closed or encounters a panic
			defer func() {
				if r := recover(); r != nil {
					log.Println("Recovered from panic:", r)
				}
			}()

			// Send the data to the listener
			listener <- dataBytes
		}()
	}
}

// watch listens to MongoDB change streams for updates in the collection.
func (ps *MongoPubSub) watch(ctx context.Context) {
	changeStream, err := ps.collection.Watch(ctx, mongo.Pipeline{})

	if err != nil {
		panic(err)
	}

	// Process change stream events asynchronously
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

// Subscribe registers a channel to listen for events with a specific trigger name.
func (ps *MongoPubSub) Subscribe(triggerName string, ch chan []byte) func() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.events[triggerName] = append(ps.events[triggerName], ch)

	// Return a function to allow unsubscribing from the event
	return func() {
		ps.Unsubscribe(triggerName, ch)
	}
}

// Unsubscribe removes a channel from the list of listeners for a specific trigger.
func (ps *MongoPubSub) Unsubscribe(triggerName string, ch chan []byte) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Find and remove the channel from the list of listeners
	for index, listener := range ps.events[triggerName] {
		if listener == ch {
			ps.events[triggerName] = append(ps.events[triggerName][:index], ps.events[triggerName][index+1:]...)
			close(listener) // Close the channel to signal the end of the subscription
		}
	}
}

// Publish sends an event with the specified trigger name and data to all subscribers.
func (ps *MongoPubSub) Publish(triggerName string, data interface{}) error {
	document := &PubSubDocument{
		TriggerName: triggerName,
		Data:        data,
		Timestamp:   time.Now(),
	}

	// Insert the event into the MongoDB collection
	_, err := ps.collection.InsertOne(context.TODO(), document)

	return err
}
