package wmsqlitemodernc

import (
	"cmp"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

type PublisherConfiguration struct {
	Connector                 Connector
	GenerateMessagesTableName TableNameGenerator
	GenerateOffsetsTableName  TableNameGenerator
	Logger                    watermill.LoggerAdapter
}

type publisher struct {
	db                        *sql.DB
	generateMessagesTableName TableNameGenerator
	generateOffsetsTableName  TableNameGenerator
	logger                    watermill.LoggerAdapter // TODO: Implement logging

	mu          sync.Mutex
	knownTopics map[string]struct{}
}

func NewPublisher(cfg PublisherConfiguration) (message.Publisher, error) {
	db, err := cfg.Connector.Connect()
	if err != nil {
		return nil, err
	}

	return &publisher{
		db: db,
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
		mu:          sync.Mutex{},
		knownTopics: make(map[string]struct{}),
	}, nil
}

func (p *publisher) Publish(topic string, messages ...*message.Message) (err error) {
	if len(messages) == 0 {
		return nil
	}
	messagesTableName := p.generateMessagesTableName.GenerateTableName(topic)

	p.mu.Lock()
	if _, ok := p.knownTopics[topic]; !ok {
		if err = createTopicAndOffsetsTablesIfAbsent(
			context.TODO(), // using the context of the first message was failing an official implementation test
			p.db,
			messagesTableName,
			p.generateOffsetsTableName.GenerateTableName(topic),
		); err != nil {
			p.mu.Unlock()
			return err
		}
		p.knownTopics[topic] = struct{}{}
	}
	p.mu.Unlock()

	query := strings.Builder{}
	_, _ = query.WriteString("INSERT INTO '")
	_, _ = query.WriteString(messagesTableName)
	_, _ = query.WriteString("' (uuid, created_at, payload, metadata) VALUES ")

	values := make([]any, 0, len(messages)*4)
	for _, msg := range messages {
		metadata, err := json.Marshal(msg.Metadata)
		if err != nil {
			return fmt.Errorf("unable to encode message %q metadata to JSON: %w", msg.UUID, err)
		}
		values = append(values, msg.UUID, time.Now().Format(time.RFC3339), msg.Payload, metadata)
		query.WriteString(`(?,?,?,?),`)
		// fmt.Println("queued up message for publication:", msg.UUID)
	}

	_, err = p.db.Exec(
		strings.TrimRight(query.String(), ","),
		values...,
	)
	return err
}

func (p *publisher) Close() error {
	return p.db.Close()
}

func (p *publisher) String() string {
	return "sqlite3-modernc-publisher"
}
