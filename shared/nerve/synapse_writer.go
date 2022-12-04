// Package nerve
// file was created on 25.05.2022 by ds
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
	"sync"
	"time"
)

func (s *Synapse) queueWriter(id int, queueName QueueName, lastSavedId QueueElementIndex, ch chan *Packet, controlChan chan ControlCommand) {
	buffer := &[]*Packet{}
	writerTrigger := make(chan struct{}, 2)

	writeLock := sync.RWMutex{}

	for {
		if len(*buffer) < s.getDefaultIOBatchSize(queueName) {
			select {
			case <-writerTrigger:
				if len(*buffer) > 0 && writeLock.TryLock() {
					writeBuffer := buffer
					buffer = &[]*Packet{}
					go func(writeBuffer *[]*Packet) {
						s.writeBuffer(id, queueName, lastSavedId, writeBuffer)

						writeLock.Unlock()
						if len(writerTrigger) == 0 {
							writerTrigger <- struct{}{}
						}
					}(writeBuffer)
				}
			case cmd := <-controlChan:
				if cmd.terminate {
					return
				}
			case p := <-ch:
				*buffer = append(*buffer, p)
				if len(writerTrigger) == 0 && (len(ch) == 0 || len(*buffer) > 100) {
					writerTrigger <- struct{}{}
				}
			}
		} else {
			timeOut := time.NewTimer(1 * time.Second)
			select {
			case <-writerTrigger:
				if len(*buffer) > 0 && writeLock.TryLock() {
					writeBuffer := buffer
					buffer = &[]*Packet{}
					go func(writeBuffer *[]*Packet) {
						s.writeBuffer(id, queueName, lastSavedId, writeBuffer)

						writeLock.Unlock()
						if len(writerTrigger) == 0 {
							writerTrigger <- struct{}{}
						}
					}(writeBuffer)
				}
			case <-timeOut.C:
				if len(writerTrigger) == 0 && (len(ch) == 0 || len(*buffer) > 100) {
					writerTrigger <- struct{}{}
				}
			case cmd := <-controlChan:
				if cmd.terminate {
					return
				}
			}
		}
	}
}

func (s *Synapse) writeBuffer(id int, queueName QueueName, lastSavedId QueueElementIndex, writeBuffer *[]*Packet) {
	if s.trace {
		s.logger.Info().Int("id", id).Interface("saving batch", *writeBuffer).Send()
	}
	for {
		err := s.Backend.WriteBatch(queueName, *writeBuffer)
		if err != nil {
			s.logger.Error().Int("id", id).
				Interface("p", *writeBuffer).
				Err(err).Msg("Failed to write batch")
		} else {
			break
		}
	}
	if s.trace {
		s.logger.Info().Int("id", id).Interface("done saving batch", *writeBuffer).Send()
	}
	for _, packet := range *writeBuffer {
		ackCh := s.getQueueAckManChannel(queueName, packet, lastSavedId)
		ackCh <- packet
	}
}

func (s *Synapse) saveQueueWriterPtr(queueName QueueName, ptr QueueElementIndex) error {
	if s.trace {
		s.logger.Debug().Str("q-name", string(queueName)).Interface("ptr", ptr).Msg("write-ptr")
	}

	for {
		err := s.Backend.WritePtr(queueName, "", ptr)
		if err == nil {
			return nil
		}

		s.logger.Error().Err(err).
			Str("queue", string(queueName)).
			Int64("ptr", int64(ptr)).
			Msg("error saving queue pointer")
	}
}
