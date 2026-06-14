package pulsar

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// ErrUnknownJobType is returned when the job name is not in the topic registry.
var ErrUnknownJobType = errors.New("unknown job type")

// TopicResolver maps job type names to Pulsar topic URIs.
type TopicResolver interface {
	Resolve(jobName string) (topic string, err error)
}

// FileTopicResolver loads job name → topic mappings from a YAML manifest.
type FileTopicResolver struct {
	byName map[string]string
}

type jobTopicsFile struct {
	Topics map[string]jobTopicEntry `yaml:"topics"`
}

type jobTopicEntry struct {
	Topic  string `yaml:"topic"`
	Domain string `yaml:"domain"`
}

// NewFileTopicResolver reads and parses the YAML file at path.
func NewFileTopicResolver(path string) (*FileTopicResolver, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read job topics config: %w", err)
	}
	var file jobTopicsFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse job topics config: %w", err)
	}
	if len(file.Topics) == 0 {
		return nil, errors.New("job topics config: topics map is empty")
	}
	byName := make(map[string]string, len(file.Topics))
	for name, entry := range file.Topics {
		name = strings.TrimSpace(name)
		topic := strings.TrimSpace(entry.Topic)
		if name == "" {
			return nil, errors.New("job topics config: empty job name key")
		}
		if topic == "" {
			return nil, fmt.Errorf("job topics config: topic for %q is empty", name)
		}
		domain := strings.TrimSpace(entry.Domain)
		if domain == "" {
			return nil, fmt.Errorf("job topics config: domain for %q is empty", name)
		}
		byName[name] = topic
	}
	return &FileTopicResolver{byName: byName}, nil
}

// Resolve returns the Pulsar topic for jobName.
func (r *FileTopicResolver) Resolve(jobName string) (string, error) {
	jobName = strings.TrimSpace(jobName)
	if jobName == "" {
		return "", ErrUnknownJobType
	}
	topic, ok := r.byName[jobName]
	if !ok {
		return "", fmt.Errorf("%w: %q", ErrUnknownJobType, jobName)
	}
	return topic, nil
}

// UniqueTopics returns all distinct Pulsar topics in the registry.
// Used for multi-consumer setup where executors subscribe to all topics.
func (r *FileTopicResolver) UniqueTopics() []string {
	seen := make(map[string]bool)
	var topics []string

	for _, topic := range r.byName {
		if !seen[topic] {
			seen[topic] = true
			topics = append(topics, topic)
		}
	}

	return topics
}
