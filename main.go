package main

import (
	"log"
	"os"
	"runtime"

	"github.com/cenkalti/backoff"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "rabbit-uri",
			Value:  "amqp://guest:guest@localhost:5672/",
			EnvVar: "RABBIT_URI",
			Usage:  "RabbitMQ connection string",
		},
		cli.StringFlag{
			Name:   "rabbit-queue",
			Value:  "vici",
			EnvVar: "RABBIT_QUEUE",
			Usage:  "Name of RabbitMQ queue to declare and consume from",
		},
		cli.StringFlag{
			Name:   "rabbit-exchange",
			Value:  "eiffel",
			EnvVar: "RABBIT_EXCHANGE",
			Usage:  "Name of the exchange to build [QUEUE] to",
		},
		cli.StringFlag{
			Name:   "rabbit-exchange-type",
			Value:  "topic",
			EnvVar: "RABBIT_EXCHANGE_TYPE",
			Usage:  "Type of [EXCHANGE]",
		},
		cli.StringFlag{
			Name:   "rabbit-binding-key",
			Value:  "eiffel.#",
			EnvVar: "RABBIT_BINDING_KEY",
			Usage:  "Binding key to use to bind [QUEUE] to [EXCHANGE]",
		},
		cli.StringFlag{
			Name:   "rabbit-consumer-tag",
			Value:  "plumb-rabbit-to-mongo",
			EnvVar: "RABBIT_CONSUMER_TAG",
			Usage:  "String to identify this consumer with the RabbitMQ broker",
		},
		cli.StringFlag{
			Name:   "mongo-uri",
			Value:  "mongodb://localhost/dbname",
			EnvVar: "MONGO_URI",
			Usage:  "MongoDB connection string. MUST contian database",
		},
		cli.StringFlag{
			Name:   "mongo-collection",
			Value:  "",
			EnvVar: "MONGO_COLLECTION",
			Usage:  "Name of the mongo collection to save to",
		},
	}
	app.Action = func(c *cli.Context) error {
		consumer := NewConsumer(
			c.String("rabbit-consumer-tag"),
			c.String("rabbit-uri"),
			c.String("rabbit-exchange"),
			c.String("rabbit-exchange-type"),
			c.String("rabbit-binding-key"))

		err := backoff.Retry(consumer.Connect, backoff.NewExponentialBackOff())
		if err != nil {
			panic(err)
		}

		deliveries, err := consumer.AnnounceQueue(
			c.String("rabbit-queue"),
			c.String("rabbit-binding-key"))
		if err != nil {
			panic(err)
		}

		log.Print("Connecting to Mongo")
		saver := NewSaver(c.String("mongo-uri"), c.String("mongo-collection"))
		err = backoff.Retry(saver.Connect, backoff.NewExponentialBackOff())
		if err != nil {
			panic(err)
		}
		defer saver.Close()

		consumer.Handle(
			deliveries,
			saver.SaveAllDeliveries,
			runtime.GOMAXPROCS(0),
			c.String("rabbit-queue"),
			c.String("rabbit-routing-key"))

		log.Print("Finished handling.")
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Goodbye!")
}
