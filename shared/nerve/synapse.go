// Package nerve
// file was created on 21.05.2022 by ds
//
//	       ,.,
//	      MMMM_    ,..,
//	        "_ "__"MMMMM          ,...,,
//	 ,..., __." --"    ,.,     _-"MMMMMMM
//	MMMMMM"___ "_._   MMM"_."" _ """"""
//	 """""    "" , \_.   "_. ."
//	        ,., _"__ \__./ ."
//	       MMMMM_"  "_    ./
//	        ''''      (    )
//	 ._______________.-'____"---._.
//	  \                          /
//	   \________________________/
//	   (_)                    (_)
//
// ------------------------------------------------
package nerve

import (
	"github.com/rs/zerolog/log"
	"octopus/shared/gremlin"
	"octopus/target/generated-sources/protobuf/nerve"
	"sync"
)

func NewSynapse(backend SynapseBackend) *Synapse {
	logger := log.With().
		Str("host", backend.GetHostName()).
		Logger()

	s := &Synapse{
		infoChan:                make(chan ControlChanInfo),
		queueChannels:           make(map[QueueName]chan *Packet),
		queueChannelsLock:       sync.RWMutex{},
		controlChan:             make(chan ControlCommand),
		queueAckManChannels:     make(map[QueueName]chan *Packet),
		queueAckManChannelsLock: sync.RWMutex{},
		controlChannels:         make([]chan ControlCommand, 0),
		controlChannelsLock:     sync.RWMutex{},
		queueWriterChannels:     make(map[QueueName][]chan *Packet),
		queueWriterChannelsLock: sync.RWMutex{},
		Backend:                 backend,
		trace:                   false,
		logger:                  &logger,
	}

	backend.SetTrace(s.trace)

	go func() {
		for {
			info := <-s.infoChan
			if s.trace {
				s.logger.Info().Interface("info", info).Send()
			}
		}
	}()

	return s
}

func (s *Synapse) SetTrace(trace bool) {
	s.trace = trace
	if s.Backend != nil {
		s.Backend.SetTrace(trace)
	}
}

// Send sends message to `queueName` with body of `data`
// if ordering is important in some respect, pass `orderKey` (e.g. user-id)
// otherwise - pass `nil` as orderKey
func (s *Synapse) Send(queue QueueConfig, packet *Packet) (QueueElementIndex, error) {
	// packet.dataHash = sha512.Sum512(packet.Data)
	packet.confirmationChannel = make(chan *Packet, 2)
	s.getQueueRunnerChannel(queue.Name, packet) <- packet

	res := <-packet.confirmationChannel

	return res.DbId, nil
}

func (s *Synapse) SendProtoPack(queue QueueConfig, msg []gremlin.ProtoWriter) error {
	var packets []*Packet
	for _, m := range msg {
		data := m.Marshal()
		packets = append(packets, &Packet{Data: data})
	}
	return s.SendPack(queue, packets)
}

// SendPack sends slice of packets to nerve to improve throughput
func (s *Synapse) SendPack(queue QueueConfig, packets []*Packet) error {
	var confirmChan = make(chan *Packet, len(packets))

	for _, packet := range packets {
		packet.confirmationChannel = confirmChan
		s.getQueueRunnerChannel(queue.Name, packet) <- packet
	}

	var cnt = 0
	for cnt < len(packets) {
		<-confirmChan
		cnt += 1
	}
	return nil
}

func (s *Synapse) SendSourcedPack(queue QueueConfig, pack []*nerve.NerveSourcedPacket) error {
	var packets = make([]*Packet, len(pack))
	for i, packet := range pack {
		serialized := packet.Marshal()
		packets[i] = &Packet{Data: serialized}
	}

	return s.SendPack(queue, packets)
}

func (s *Synapse) AsyncSendSourcedPacket(queue QueueConfig, msg *nerve.NerveSourcedPacket) (chan *Packet, error) {
	var confirmChan = make(chan *Packet)
	packet := &Packet{Data: msg.Marshal(), confirmationChannel: confirmChan}
	s.getQueueRunnerChannel(queue.Name, packet) <- packet

	return confirmChan, nil
}

func (s *Synapse) AsyncSendSourcedPack(queue QueueConfig, msg []*nerve.NerveSourcedPacket) (chan struct{}, error) {
	var confirmChan = make(chan *Packet, len(msg))
	var packets = make([]*Packet, len(msg))
	for i, data := range msg {
		packets[i] = &Packet{Data: data.Marshal(), confirmationChannel: confirmChan}
		s.getQueueRunnerChannel(queue.Name, packets[i]) <- packets[i]
	}

	var res = make(chan struct{})
	go func() {
		var cnt = 0
		for cnt < len(packets) {
			<-confirmChan
			cnt += 1
		}
		res <- struct{}{}
	}()
	return res, nil
}

// GetReceiver returns pointer to Receiver structure
// which in turn contains two channels - one to receive
// data packets from and one to send empty structure to
// terminate receiver thread
func (s *Synapse) GetReceiver(queue QueueConfig, consumer ConsumerId) *Receiver {
	return s.GetBufferedReceiver(queue, consumer, 0)
}

func (s *Synapse) GetBufferedReceiver(queue QueueConfig, consumer ConsumerId, bufferSize int) *Receiver {
	return s.newReceiver(s.logger, queue.Name, consumer, bufferSize)
}

func (s *Synapse) GetPointer(queue QueueConfig, consumer ConsumerId) (QueueElementIndex, error) {
	return s.Backend.GetPtr(queue.Name, consumer)
}
