package gohome

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"cloud.google.com/go/pubsub"
	"github.com/pkg/errors"
	"google.golang.org/api/option"
)

const PROJECT = "gohome-dev"
const TOPIC = "calling_home"
const SUBSCRIPTION = "home_listening"
const defaultCredFile = ".gohome/gohome-cred.json"

var psc *pubsub.Client

type PubSub struct {
	client  *pubsub.Client
	inTopic *pubsub.Topic
	inSub   *pubsub.Subscription
	ctx     context.Context
}

func NewPubSub() (*PubSub, error) {
	var err error
	ctx := context.Background()
	homePath := os.Getenv("HOME")
	credentialPath := filepath.Join(homePath, defaultCredFile)
	psc, err := pubsub.NewClient(ctx, PROJECT, option.WithCredentialsFile(credentialPath))
	if err != nil {
		return nil, errors.Wrapf(err, "cannot create Pub/Sub client for project %s", PROJECT)
	}
	topic := psc.Topic(TOPIC)
	fmt.Printf("topic is:%v\n", topic)
	toExists, err := topic.Exists(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot check topic %s existance", TOPIC)
	}
	if !toExists {
		topic, err = psc.CreateTopic(context.Background(), TOPIC)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot create topic %s", TOPIC)
		}
	}
	sub := psc.Subscription(SUBSCRIPTION)
	ok, err := sub.Exists(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot check if subscription %s exists", SUBSCRIPTION)
	}
	if !ok {
		sub, err = psc.CreateSubscription(ctx, SUBSCRIPTION, pubsub.SubscriptionConfig{Topic: topic})
		fmt.Printf("Subscription created: %s\n", SUBSCRIPTION)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot create subscription %s", SUBSCRIPTION)
		}
	}
	pubsub := PubSub{client: psc, inTopic: topic, inSub: sub, ctx: ctx}
	return &pubsub, nil
}

func (p *PubSub) Listen(home *Home) (<-chan Message, <-chan error) {
	incoming := make(chan Message, 1)
	errors := make(chan error)
	go p.listen(home, incoming, errors)
	return incoming, errors
}

func (p *PubSub) listen(home *Home, incoming chan Message, errors chan error) {
	err := p.inSub.Receive(p.ctx, func(ctx context.Context, m *pubsub.Message) {
		msgJSON := string(m.Data)
		msg := home.Plant.ParseFromJSON(msgJSON)
		log.Printf("Received p/s message: %s, %v", msgJSON, msg)
		incoming <- msg
		m.Ack()
	})
	if err != nil {
		fmt.Printf("PUBSUB error: %v\n", err)
		errors <- err
	}
}
