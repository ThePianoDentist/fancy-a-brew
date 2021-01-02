package fcm_client

import (
	"context"
	"log"

	"go.uber.org/zap"

	"google.golang.org/api/option"

	firebase "firebase.google.com/go/v4"

	"firebase.google.com/go/v4/messaging"
)

type FCMController struct {
	Client *messaging.Client
	Lgr    *zap.Logger
}

func NewFCMController(lgr *zap.Logger) *FCMController {
	//config := firebase.Config{
	//	AuthOverride:     nil,
	//	DatabaseURL:      "",
	//	ProjectID:        "",
	//	ServiceAccountID: "",
	//	StorageBucket:    "",
	//}
	// TODO update this.
	opt := option.WithCredentialsFile("/home/johnnyk/Documents/serviceAccountKey.json")
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}
	ctx := context.Background()
	client, err := app.Messaging(ctx)
	if err != nil {
		log.Fatalf("error getting Messaging client: %v\n", err)
	}
	return &FCMController{Client: client, Lgr: lgr}
}

func (c *FCMController) SendFCM(toToken string, data map[string]string) error {
	c.Lgr.Info("Sending fcm message to ", zap.String("To", toToken))
	message := &messaging.Message{
		Data:  data,
		Token: toToken,
	}

	// Send a message to the device corresponding to the provided registration token.
	response, err := c.Client.Send(context.Background(), message)
	if err != nil {
		return err
	}
	// Response is a message ID string.
	c.Lgr.Info("Successfully sent fcm message:", zap.String("resp", response), zap.String("To", toToken))
	return nil
}
