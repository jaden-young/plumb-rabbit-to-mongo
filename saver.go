package main

import (
	"encoding/json"
	"log"

	"github.com/globalsign/mgo"
	"github.com/streadway/amqp"
)

// Saver represents a connection to a single Mongo collection
// and allows single-threaded saving to that collection
type Saver struct {
	uri        string
	session    *mgo.Session
	collection string
}

// NewSaver instantiates a new Saver. Still need to call Connect
// before performing any operations.
func NewSaver(uri, collection string) *Saver {
	return &Saver{
		uri:        uri,
		collection: collection,
	}
}

// Connect establishes a connection to the underlying MongoDB of the saver
func (s *Saver) Connect() error {
	log.Printf("Dialing %s", s.uri)
	sess, err := mgo.Dial(s.uri)
	if err != nil {
		return err
	}
	sess.SetMode(mgo.Monotonic, true)
	s.session = sess
	return nil
}

// Close cleans up resources used by the Saver
func (s *Saver) Close() {
	if s.session != nil {
		s.session.Close()
	}
	s.session = nil
}

// SaveAllDeliveries persists the body of any recieved delivery if it is
// valid json/bson. Assumes that the consuming channel is not set to
// auto ack, as this method DOES ack on successful save, and nack on errors.
func (s *Saver) SaveAllDeliveries(deliveries <-chan amqp.Delivery) {
	sess := s.session.Copy()
	defer sess.Close()

	// We assume DB was set in the uri
	c := sess.DB("").C(s.collection)

	for d := range deliveries {
		var val map[string]interface{}
		err := json.Unmarshal(d.Body, &val)
		if err != nil {
			log.Printf("Couldn't parse message as JSON: %s", err.Error())
			d.Nack(false, true)
		} else {
			err := c.Insert(val)
			if err != nil {
				log.Printf("Error saving message: %v", err)
				d.Nack(false, true)
			} else {
				d.Ack(false)
			}
		}
	}
	return
}
