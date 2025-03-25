package wmsqlitemodernc_test

import (
	"fmt"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/dkotik/watermillsqlite/wmsqlitemodernc"
	"github.com/dkotik/watermillsqlite/wmsqlitemodernc/tests"
	_ "modernc.org/sqlite"
)

func NewPubSubFixture(t *testing.T) tests.PubSubFixture {
	// &_txlock=exclusive
	connector := wmsqlitemodernc.NewConnector(fmt.Sprintf(
		"file://%s/%s?mode=memory&journal_mode=WAL&busy_timeout=1000&secure_delete=true&foreign_keys=true&cache=shared",
		t.TempDir(),
		"db.sqlite3",
	))
	// connector := wmsqlitemodernc.NewEphemeralConnector()

	return func(t *testing.T, consumerGroup string) (message.Publisher, message.Subscriber) {
		pub, err := wmsqlitemodernc.NewPublisher(wmsqlitemodernc.PublisherConfiguration{
			Connector: connector,
		})
		if err != nil {
			t.Fatal("unable to initialize publisher:", err)
		}
		t.Cleanup(func() {
			if err := pub.Close(); err != nil {
				t.Fatal(err)
			}
		})

		sub, err := wmsqlitemodernc.NewSubscriber(wmsqlitemodernc.SubscriberConfiguration{
			ConsumerGroup: consumerGroup,
			Connector:     connector,
		})
		if err != nil {
			t.Fatal("unable to initialize publisher:", err)
		}
		t.Cleanup(func() {
			if err := sub.Close(); err != nil {
				t.Fatal(err)
			}
		})

		return pub, sub
	}
}

func TestPubSub(t *testing.T) {
	// if testing.Short() {
	// 	t.Skip("focusing to acceptance tests")
	// }
	setup := NewPubSubFixture(t)
	t.Run("basic functionality", tests.TestBasicSendRecieve(setup))
	t.Run("one publisher three subscribers", tests.TestOnePublisherThreeSubscribers(setup, 1000))
	t.Run("perpetual locks", tests.NewHung(setup))
}

func TestOfficialImplementationAcceptance(t *testing.T) {
	tests.OfficialImplementationAcceptance(NewPubSubFixture(t))(t)
}
