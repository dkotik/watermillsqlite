package wmsqlitezombiezen

import "github.com/ThreeDotsLabs/watermill"

var defaultLogger = watermill.NopLogger{}

// TODO: replace with cmp.Or after Watermill
// upgrades Golang version to 1.22
func cmpOrTODO[T comparable](vals ...T) T {
	var zero T
	for _, val := range vals {
		if val != zero {
			return val
		}
	}
	return zero
}

// TableNameGenerator creates a table name for a given topic either for
// a topic table or for offsets table.
type TableNameGenerator func(topic string) string

// TableNameGenerators is a struct that holds two functions for generating topic and offsets table names.
// A [Publisher] and a [Subscriber] must use identical generators for topic and offsets tables in order
// to communicate with each other.
type TableNameGenerators struct {
	Topic   TableNameGenerator
	Offsets TableNameGenerator
}

// WithDefaultGeneratorsInsteadOfNils returns a TableNameGenerators with default generators for topic and offsets tables
// if they were left nil.
func (t TableNameGenerators) WithDefaultGeneratorsInsteadOfNils() TableNameGenerators {
	if t.Topic == nil {
		t.Topic = func(topic string) string {
			return "watermill_" + topic
		}
	}
	if t.Offsets == nil {
		t.Offsets = func(topic string) string {
			return "watermill_offsets_" + topic
		}
	}
	return t
}
