package wmsqlitezombiezen

// return New("file:memory:?mode=memory&journal_mode=WAL&busy_timeout=3000&secure_delete=true&foreign_keys=true&cache=shared", poolSize)
// &cache=shared is critical, see: https://github.com/zombiezen/go-sqlite/issues/92#issuecomment-2052330643

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
