// Package nerve
// file was created on 03.06.2022 by ds
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
	"github.com/rs/zerolog"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

func (s *Synapse) newReceiver(logger *zerolog.Logger, queueName QueueName, consumer ConsumerId, bufSuze int) *Receiver {
	l := logger.With().Str("queue", string(queueName)).Logger()
	r := &Receiver{
		ConsumerId:            consumer,
		QueueName:             queueName,
		DataChan:              make(chan *Packet, bufSuze),
		TerminateReceiverChan: make(chan struct{}),
		TerminateReaderChan:   make(chan struct{}),
		Synapse:               s,
		ackBuffer:             make([]QueueElementIndex, 0),
		lastAckedId:           0,
		AckChannel:            make(chan QueueElementIndex, s.getDefaultReceiverChanLen()),
		ackBufferLock:         sync.RWMutex{},
		logger:                &l,
	}

	go r.readerAckManager()
	go r.receiverBody()

	return r
}

func (r *Receiver) Close() {
	r.TerminateReceiverChan <- struct{}{}
	r.TerminateReaderChan <- struct{}{}
}

func (r *Receiver) GetLastPointerFromBackend() QueueElementIndex {
	readerPtr, errR := r.Synapse.Backend.GetPtr(r.QueueName, r.ConsumerId)
	for errR != nil {
		time.Sleep(500 * time.Millisecond)
		readerPtr, errR = r.Synapse.Backend.GetPtr(r.QueueName, r.ConsumerId)
	}

	return readerPtr
}

func (r *Receiver) receiverBody() {
	var errCounter uint64 = 0
	for {
		var readerPtr QueueElementIndex
		var errR error
		if r.lastReadId == 0 {
			readerPtr, errR = r.Synapse.Backend.GetPtr(r.QueueName, r.ConsumerId)
			if r.Synapse.trace {
				r.logger.Info().Msgf("got reader ptr %v", readerPtr)
			}
		} else {
			readerPtr = r.lastReadId
		}
		writerPtr, errW := r.Synapse.Backend.GetPtr(r.QueueName, "")

		if errR != nil || errW != nil {
			r.logger.Error().
				Err(errR).
				Err(errW).
				Uint64("err-counter", errCounter).
				Msg("error reading pointers")
			errCounter++
			continue
		}

		if writerPtr > readerPtr {
			rp, err := r.readDataFromBackend(readerPtr, writerPtr,
				r.Synapse.getDefaultReaderLimit(r.QueueName, r.ConsumerId))
			if err == nil {
				if r.Synapse.trace {
					r.logger.Info().Msgf("set lastReadId to %v", rp)
				}
				r.lastReadId = rp
			}
		} else {
			time.Sleep(500 * time.Millisecond)
		}

		if len(r.TerminateReceiverChan) != 0 {
			select {
			case <-r.TerminateReceiverChan:
				return
			}
		}
	}
}

func minIndex(a, b QueueElementIndex) QueueElementIndex {
	if a < b {
		return a
	}
	return b
}

func (r *Receiver) readDataFromBackend(readerPtr, writerPtr, limit QueueElementIndex) (QueueElementIndex, error) {
	maxId := minIndex(writerPtr, readerPtr+limit)
	resultLen := maxId - readerPtr

	result := make([]*Packet, resultLen)
	for p := readerPtr + 1; p <= maxId; p++ {
		result[p-readerPtr-1] = &Packet{DbId: p}
	}

	result, err := r.Synapse.Backend.ReadBatch(r.QueueName, result)
	if err != nil {
		return 0, err
	}

	// log.Info().Interface("last-acked-id", readerPtr-1).Send()
	if readerPtr != 0 {
		atomic.CompareAndSwapInt64((*int64)(&r.lastAckedId), 0, int64(readerPtr))
	}

	if len(result) == 0 {
		r.logger.Warn().Msgf("got empty result from backend")
	}

	receivedMaxId := result[0].DbId
	receivedMinId := result[0].DbId
	for _, packet := range result {
		if packet.DbId > receivedMaxId {
			receivedMaxId = packet.DbId
		}
		if r.Synapse.trace {
			if packet.DbId < receivedMinId {
				receivedMinId = packet.DbId
			}
		}
		r.DataChan <- packet
	}
	if r.Synapse.trace {
		r.logger.Info().Msgf("Min pack %v max %v", receivedMinId, receivedMaxId)
	}

	return receivedMaxId, nil
}

func (r *Receiver) readerAckManager() {
	writing := false
	doneWriting := make(chan struct{}, 1)
	for {
		select {
		case <-r.TerminateReaderChan:
			return
		case <-doneWriting:
			writing = false
		case ackedId := <-r.AckChannel:
			if ackedId <= r.lastAckedId {
				if r.Synapse.trace {
					r.logger.Warn().
						Interface("p", r.lastAckedId).
						Msg("ackedId <= lastAckedId, skip ack")
				}
				continue
			}

			r.ackBufferLock.Lock()
			// log.Info().Interface("buffer-before-append", r.ackBuffer).Send()
			r.ackBuffer = append(r.ackBuffer, ackedId)
			if !writing {
				minId := r.ackBuffer[0]
				for _, v := range r.ackBuffer {
					if v < minId {
						minId = v
					}
				}
				r.ackBufferLock.Unlock()

				if r.Synapse.trace {
					r.logger.Info().
						Interface("p", r.lastAckedId).
						Interface("minId", minId).
						Interface("buf", r.ackBuffer).
						Msg("trying to start write")
				}
				if minId <= r.lastAckedId {
					r.logger.Error().
						Interface("p", r.lastAckedId).
						Msg("already acked pointer")
				}

				gotAtLeastOnePacketToMovePtrTo := minId == r.lastAckedId+1

				if gotAtLeastOnePacketToMovePtrTo {
					writing = true
					go func(ackedId QueueElementIndex) {
						_, n := r.tryToFlushReceiver(ackedId)
						if r.Synapse.trace {
							r.logger.Info().
								Str("queue-name", string(r.QueueName)).
								Str("host", r.Synapse.Backend.GetHostName()).
								Int("flushed-block-size", n).
								Interface("acked-id", ackedId).
								Msg("reader flushed index")
						}
						doneWriting <- struct{}{}
					}(ackedId)
				}
			} else {
				r.ackBufferLock.Unlock()
			}
		}
	}
}

func (r *Receiver) tryToFlushReceiver(currentDbId QueueElementIndex) (bool, int) {
	newReaderPtr := r.lastAckedId

	cnt := 0
	r.ackBufferLock.Lock()
	sort.Slice(r.ackBuffer, func(i, j int) bool {
		return r.ackBuffer[i] < r.ackBuffer[j]
	})

	for _, dbId := range r.ackBuffer {
		if dbId == newReaderPtr+1 {
			newReaderPtr = newReaderPtr + 1
			cnt++
		} else {
			break
		}
	}
	flushedIndex := cnt

	if r.Synapse.trace {
		r.logger.Info().Interface("cnt", cnt).
			Interface("from", r.ackBuffer[0]).
			Interface("to", r.ackBuffer[flushedIndex-1]).
			Int("buffer-len", len(r.ackBuffer)).
			Msg("flushing indexes")
	}

	r.ackBuffer = r.ackBuffer[flushedIndex:]

	/* log.Info().
	Interface("buffer", r.ackBuffer).
	Int("buffer-len", len(r.ackBuffer)).
	Msg("flushing indexes") */

	r.ackBufferLock.Unlock()

	for {
		err := r.Synapse.Backend.WritePtr(r.QueueName, r.ConsumerId, newReaderPtr)

		if err == nil {
			break
		}

		r.logger.Error().Err(err).Interface("r", r).
			Msgf("error moving reader pointer to %d", newReaderPtr)
	}

	atomic.StoreInt64((*int64)(&r.lastAckedId), int64(newReaderPtr))
	return currentDbId <= newReaderPtr, cnt
}

// Ack marks packet in question
func (r *Receiver) Ack(p *Packet) {
	r.AckChannel <- p.DbId
}

// AckId marks packet in question
func (r *Receiver) AckId(id QueueElementIndex) {
	if r.Synapse.trace {
		r.logger.Info().Interface("id", id).Msg("ack id")
	}
	r.AckChannel <- id
}
