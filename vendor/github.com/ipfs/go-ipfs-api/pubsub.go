package shell

import (
	"encoding/json"

	"github.com/libp2p/go-libp2p-peer"
)

// Message is a pubsub message.
type Message struct {
	From     peer.ID
	Data     []byte
	Seqno    []byte
	TopicIDs []string
}

// PubSubSubscription allow you to receive pubsub records that where published on the network.
type PubSubSubscription struct {
	resp *Response
}

func newPubSubSubscription(resp *Response) *PubSubSubscription {
	sub := &PubSubSubscription{
		resp: resp,
	}

	return sub
}

// Next waits for the next record and returns that.
func (s *PubSubSubscription) Next() (*Message, error) {
	if s.resp.Error != nil {
		return nil, s.resp.Error
	}

	d := json.NewDecoder(s.resp.Output)

	var r struct {
		From     []byte   `json:"from,omitempty"`
		Data     []byte   `json:"data,omitempty"`
		Seqno    []byte   `json:"seqno,omitempty"`
		TopicIDs []string `json:"topicIDs,omitempty"`
	}

	err := d.Decode(&r)
	if err != nil {
		return nil, err
	}

	from, err := peer.IDFromBytes(r.From)
	if err != nil {
		return nil, err
	}
	return &Message{
		From:     from,
		Data:     r.Data,
		Seqno:    r.Seqno,
		TopicIDs: r.TopicIDs,
	}, nil
}

// Cancel cancels the given subscription.
func (s *PubSubSubscription) Cancel() error {
	if s.resp.Output == nil {
		return nil
	}

	return s.resp.Output.Close()
}
