package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-playground/webhooks/v6/github"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	utils "workflows.cloudbees.com/checks-sensor-github/utils"
)

const (
	path             = "/webhooks"
	pushEventChannel = "pushEvents"
)

var (
	logger *zap.SugaredLogger
	ctx    context.Context
)

func main() {
	logger = utils.NewLogger()
	logger.Info("Starting Github sensor...")
	ctx := context.Background()
	redis := utils.NewRedisClient()

	hook, _ := github.New(github.Options.Secret("TestChecks"))

	http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		payload, err := hook.Parse(r, github.PushEvent, github.RepositoryEvent)
		if err != nil {
			if err == github.ErrEventNotFound {
				logger.Error("Error:" + err.Error())
				// ok event wasn;t one of the ones asked to be parsed
			}
		}

		switch payload.(type) {
		case github.RepositoryPayload:
			repositoryPayload := payload.(github.RepositoryPayload)
			// Do whatever you want from here...
			logger.Info("%+v", repositoryPayload)

		case github.PushPayload:
			pushPayload := payload.(github.PushPayload)
			PublishMessage(ctx, redis, pushEventChannel, pushPayload)
		}
	})

	go subscribeToRedis(ctx, redis, pushEventChannel)

	logger.Info("Sensor listening on port 3000")
	http.ListenAndServe(":3000", nil)
}

// Subscribe to Redis. Mainly used for debugging Publising
func subscribeToRedis(ctx context.Context, redis *redis.Client, channel string) {
	pubsub := redis.Subscribe(ctx, channel)
	ch := pubsub.Channel()
	logger.Info("Subscribed to channel ", channel)

	for msg := range ch {
		logger.Info(msg.Channel, " content: ", msg.Payload)
	}
}

// Publish to Redis
func PublishMessage(ctx context.Context, redis *redis.Client, channel string, payload github.PushPayload) {
	event := PushEvent{Payload: payload}
	logger.Debug("Publishing event ", event)
	err := redis.Publish(ctx, channel, event).Err()
	if err != nil {
		panic(err)
	}
}

type PushEvent struct {
	Payload github.PushPayload `json:"payload"`
}

func (p PushEvent) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}
