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

import (
	"time"
)

func (s *Synapse) queueRunner(queueName QueueName, ch chan *Packet, controlChan chan ControlCommand) {
	var queueCounter QueueElementIndex
	var err error
	for {
		queueCounter, err = s.Backend.GetPtr(queueName, "")
		if err == nil {
			break
		}
		s.logger.Error().Err(err).
			Str("queue-name", string(queueName)).
			Msg("error reading queue counter in writer")

		time.Sleep(1 * time.Second)
	}

	lastSavedId := queueCounter

	for {
		select {
		case cmd := <-controlChan:
			if cmd.terminate {
				return
			}
		case p := <-ch:
			queueCounter++
			p.DbId = queueCounter
			if s.trace {
				s.logger.Info().Interface("packet", p).Msg("sent")
			}
			s.getQueueWriterChannel(queueName, p.DbId, lastSavedId) <- p
		}
	}
}
