package wmsqlitemodernc

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

var (
	ErrClosed = errors.New("subscriber is closed")
)

type Subscriber interface {
	message.Subscriber
	Unsubscribe(topic string) error
}

type SubscriberConfiguration struct {
	ConsumerGroup             string
	BatchSize                 int
	GenerateMessagesTableName TableNameGenerator
	GenerateOffsetsTableName  TableNameGenerator
	Connector                 Connector
	PollInterval              time.Duration
	Logger                    watermill.LoggerAdapter
}

type subscriber struct {
	consumerGroup             string
	batchSize                 int
	connector                 Connector
	closed                    chan struct{}
	generateMessagesTableName TableNameGenerator
	generateOffsetsTableName  TableNameGenerator
	logger                    watermill.LoggerAdapter

	mu                   sync.Mutex
	subscriptionsByTopic map[string]*subscription
}

func NewSubscriber(cfg SubscriberConfiguration) (Subscriber, error) {
	// TODO: validate config
	// TODO: validate consumer group - INJECTION
	// TODO: validate batch size
	return &subscriber{
		consumerGroup: cfg.ConsumerGroup,
		batchSize:     cmp.Or(cfg.BatchSize, 10),
		connector:     cfg.Connector,
		closed:        make(chan struct{}),
		generateMessagesTableName: cmp.Or(
			cfg.GenerateMessagesTableName,
			DefaultMessagesTableNameGenerator,
		),
		generateOffsetsTableName: cmp.Or(
			cfg.GenerateOffsetsTableName,
			DefaultOffsetsTableNameGenerator,
		),
		logger: cmp.Or[watermill.LoggerAdapter](
			cfg.Logger,
			watermill.NewSlogLogger(nil),
		),
		mu:                   sync.Mutex{},
		subscriptionsByTopic: make(map[string]*subscription),
	}, nil
}

func (s *subscriber) Subscribe(ctx context.Context, topic string) (c <-chan *message.Message, err error) {
	select {
	case <-s.closed:
		return nil, ErrClosed
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	matched, ok := s.subscriptionsByTopic[topic]
	if ok {
		return matched.destination, nil
	}

	db, err := s.connector.Connect()
	if err != nil {
		return nil, err
	}
	messagesTableName := s.generateMessagesTableName.GenerateTableName(topic)
	offsetsTableName := s.generateOffsetsTableName.GenerateTableName(topic)
	if err = createTopicAndOffsetsTablesIfAbsent(
		ctx,
		db,
		messagesTableName,
		offsetsTableName,
	); err != nil {
		return nil, errors.Join(err, db.Close())
	}

	_, err = db.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO '%s' (consumer_group, offset_acked, offset_consumed)
		VALUES ("%s", 0, 0)
		ON CONFLICT(consumer_group) DO NOTHING;
	`, offsetsTableName, s.consumerGroup))
	if err != nil {
		return nil, errors.Join(err, db.Close())
	}

	// TODO: customize batch size
	matched = &subscription{
		db:     db,
		ticker: time.NewTicker(time.Millisecond * 120),
		sqlNextMessageBatch: fmt.Sprintf(`
			SELECT
				"offset", uuid, created_at, payload, metadata
			FROM '%s'
			WHERE "offset" > (
				SELECT offset_acked FROM '%s' WHERE consumer_group = "%s"
			)
			ORDER BY offset LIMIT %d;
		`, messagesTableName, offsetsTableName, s.consumerGroup, s.batchSize),
		sqlAcknowledgeMessage: fmt.Sprintf(`
			UPDATE '%s' SET offset_acked = ? WHERE consumer_group = "%s" AND offset_acked = ?;
		`, offsetsTableName, s.consumerGroup),
		destination: make(chan *message.Message),
		logger:      s.logger, // TODO: logger.With
	}
	s.subscriptionsByTopic[topic] = matched
	go matched.Loop(s.closed)
	return matched.destination, nil
}

func (s *subscriber) Unsubscribe(topic string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	subscription, ok := s.subscriptionsByTopic[topic]
	if !ok {
		return nil
	}
	if err := subscription.Close(); err != nil {
		return err
	}
	delete(s.subscriptionsByTopic, topic)
	return nil
}

func (s *subscriber) Close() (err error) {
	if s.closed == nil {
		return nil
	}
	close(s.closed)
	s.closed = nil
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sub := range s.subscriptionsByTopic {
		err = errors.Join(err, sub.Close())
	}
	return err
}

func (s *subscriber) String() string {
	return "sqlite3-modernc-subscriber"
}
