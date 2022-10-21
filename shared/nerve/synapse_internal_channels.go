// Package nerve
// file was created on 25.05.2022 by ds
//          ,.,
//         MMMM_    ,..,
//           "_ "__"MMMMM          ,...,,
//    ,..., __." --"    ,.,     _-"MMMMMMM
//   MMMMMM"___ "_._   MMM"_."" _ """"""
//    """""    "" , \_.   "_. ."
//           ,., _"__ \__./ ."
//          MMMMM_"  "_    ./
//           ''''      (    )
//    ._______________.-'____"---._.
//     \                          /
//      \________________________/
//      (_)                    (_)
//
// ------------------------------------------------
//
package nerve

func (s *Synapse) getControlChannel() chan ControlCommand {
	ch := make(chan ControlCommand)
	s.controlChannelsLock.Lock()
	s.controlChannels = append(s.controlChannels, ch)
	s.controlChannelsLock.Unlock()

	return ch
}

func (s *Synapse) getQueueRunnerChannel(queueName QueueName, _ *Packet) chan *Packet {
	return getOrAddItem(&s.queueChannels, &s.queueChannelsLock, queueName, func() chan *Packet {
		ch := make(chan *Packet, s.getQueueRunnerChannelLen(queueName))
		go s.queueRunner(queueName, ch, s.getControlChannel())
		return ch
	})
}

func (s *Synapse) getQueueWriterChannel(queueName QueueName, dbId, lastSavedId QueueElementIndex) chan *Packet {
	// shardIdx := data[i].DbId % QueueElementIndex(s.config.TableParallelism)
	n := int(dbId % QueueElementIndex(s.getQueueWriterIOThreads(queueName)))

	return getOrAddNthItem(&s.queueWriterChannels, &s.queueWriterChannelsLock, queueName, n, func() []chan *Packet {
		channels := make([]chan *Packet, s.getQueueWriterIOThreads(queueName))
		for i := range channels {
			ch := make(chan *Packet, s.getQueueWriterIOThreadChannelLen(queueName))
			channels[i] = ch
			go s.queueWriter(i, queueName, lastSavedId, ch, s.getControlChannel())
		}

		return channels
	})
}

func (s *Synapse) getQueueAckManChannel(queueName QueueName, _ *Packet, lastSavedId QueueElementIndex) chan *Packet {
	return getOrAddItem(&s.queueAckManChannels, &s.queueAckManChannelsLock, queueName, func() chan *Packet {
		ch := make(chan *Packet, s.getQueueAckManChannelLen(queueName))
		go s.queueAckMan(queueName, lastSavedId, ch, s.getControlChannel())
		return ch
	})
}
