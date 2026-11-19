package database

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func ConnectMongo(ctx context.Context, uri string, cleanupTimeout time.Duration) (*mongo.Client, error) {
	if strings.TrimSpace(uri) == "" {
		return nil, errors.New("MongoDB URI is required")
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("create MongoDB client: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		pingErr := fmt.Errorf("ping MongoDB: %w", err)
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
		defer cancel()
		if cleanupErr := client.Disconnect(cleanupCtx); cleanupErr != nil {
			return nil, errors.Join(pingErr, fmt.Errorf("disconnect MongoDB client: %w", cleanupErr))
		}
		return nil, pingErr
	}

	return client, nil
}
