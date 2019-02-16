package shell

import (
	"encoding/binary"
	"encoding/json"

	"github.com/libp2p/go-libp2p-peer"
	pb "github.com/libp2p/go-libp2p-pubsub/pb"
)

// PubSubRecord is a record received via PubSub.
type PubSubRecord interface {
	// From returns the peer ID of the node that published this record
	From() peer.ID

	// Data returns the data field
	Data() []byte

	// SeqNo is the sequence number of this record
	SeqNo() int64

	//TopicIDs is the list of topics this record belongs to
	TopicIDs() []string
}

type message struct {
	*pb.Message
}

func (m *message) GetFrom() peer.ID {
	return peer.ID(m.Message.GetFrom())
}

type floodsubRecord struct {
	msg *message
}

func (r floodsubRecord) From() peer.ID {
	return r.msg.GetFrom()
}

func (r floodsubRecord) Data() []byte {
	return r.msg.GetData()
}

func (r floodsubRecord) SeqNo() int64 {
	return int64(binary.BigEndian.Uint64(r.msg.GetSeqno()))
}

func (r floodsubRecord) TopicIDs() []string {
	return r.msg.GetTopicIDs()
}

///

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
func (s *PubSubSubscription) Next() (PubSubRecord, error) {
	if s.resp.Error != nil {
		return nil, s.resp.Error
	}

	d := json.NewDecoder(s.resp.Output)

	r := &message{}
	err := d.Decode(r)

	return floodsubRecord{msg: r}, err
}

// Cancel cancels the given subscription.
func (s *PubSubSubscription) Cancel() error {
	if s.resp.Output == nil {
		return nil
	}

	return s.resp.Output.Close()
}
