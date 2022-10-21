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

const perChannelMemory = 1024 * 16 // * 8 bytes...

func (s *Synapse) getQueueConfirmationBufferDefaultSize(_ QueueName) int {
	return perChannelMemory
}

func (s *Synapse) getQueueRunnerChannelLen(_ QueueName) int {
	return perChannelMemory
}

func (s *Synapse) getQueueAckManChannelLen(_ QueueName) int {
	return perChannelMemory
}

func (s *Synapse) getQueueWriterIOThreads(name QueueName) uint {
	return s.Backend.GetDefaultQueueParallelism(name)
}

func (s *Synapse) getQueueWriterIOThreadChannelLen(_ QueueName) int {
	return perChannelMemory
}

func (s *Synapse) getDefaultReaderLimit(_ QueueName, _ ConsumerId) QueueElementIndex {
	return 50000
}

func (s *Synapse) getDefaultReceiverChanLen() int {
	return 10000
}
