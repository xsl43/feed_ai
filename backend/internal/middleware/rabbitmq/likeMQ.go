package rabbitmq

import (
	"context"
	"errors"
	"time"
)

type LikeMQ struct {
	*RabbitMQ
}

const (
	likeExchange   = "like.events"
	likeQueue      = "like.events"
	likeBindingKey = "like.*"

	likeLikeRK   = "like.like"
	likeUnlikeRK = "like.unlike"
)

type LikeEvent struct {
	EventID    string    `json:"event_id"`
	Action     string    `json:"action"`
	UserID     uint      `json:"user_id"`
	VideoID    uint      `json:"video_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

func NewLikeMQ(base *RabbitMQ) (*LikeMQ, error) {
	if base == nil {
		return nil, errors.New("rabbitmq base is nil")
	}
	if err := base.DeclareTopic(likeExchange, likeQueue, likeBindingKey); err != nil {
		return nil, err
	}
	return &LikeMQ{RabbitMQ: base}, nil
}

func (l *LikeMQ) Like(ctx context.Context, userID, videoID uint) error {
	return l.publish(ctx, "like", likeLikeRK, userID, videoID)
}

func (l *LikeMQ) Unlike(ctx context.Context, userID, videoID uint) error {
	return l.publish(ctx, "unlike", likeUnlikeRK, userID, videoID)
}

func (l *LikeMQ) publish(ctx context.Context, action, routingKey string, userID, videoID uint) error {
	if l == nil || l.RabbitMQ == nil {
		return errors.New("like mq is not initialized")
	}
	if userID == 0 || videoID == 0 {
		return errors.New("userID and videoID are required")
	}
	id, err := newEventID(16)
	if err != nil {
		return err
	}
	event := LikeEvent{
		EventID:    id,
		Action:     action,
		UserID:     userID,
		VideoID:    videoID,
		OccurredAt: time.Now(),
	}
	return l.PublishJSON(ctx, likeExchange, routingKey, event)
}
