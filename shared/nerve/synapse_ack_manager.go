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
	"sort"
)

type syncSectionConfig struct {
	blockChan     chan *[]*Packet
	doneWriteChan chan struct{}
	controlChan   chan ControlCommand
	queueName     QueueName
}

func (s *Synapse) queueAckMan(queueName QueueName, lastSavedId QueueElementIndex, ch chan *Packet, controlChan chan ControlCommand) {
	mkConfBuffer := func() *[]*Packet {
		res := make([]*Packet, 0, s.getQueueConfirmationBufferDefaultSize(queueName))
		return &res
	}
	var confirmationsBuffer *[]*Packet
	confirmationsBuffer = mkConfBuffer()
	blockChan := make(chan *[]*Packet, 1)
	doneWriteChan := make(chan struct{}, 1)
	controlNext := make(chan ControlCommand)

	go s.queueAckManSyncSection(&syncSectionConfig{
		blockChan:     blockChan,
		doneWriteChan: doneWriteChan,
		controlChan:   controlNext,
		queueName:     queueName,
	}, lastSavedId)

	nowWriting := false
	for {
		select {
		case <-doneWriteChan:
			if len(*confirmationsBuffer) > 0 {
				blockChan <- confirmationsBuffer
				confirmationsBuffer = mkConfBuffer()
				nowWriting = true
			} else {
				nowWriting = false
			}
		case q := <-controlChan:
			controlNext <- q
			return
		case p := <-ch:
			*confirmationsBuffer = append(*confirmationsBuffer, p)
			if !nowWriting {
				blockChan <- confirmationsBuffer
				confirmationsBuffer = mkConfBuffer()
				nowWriting = true
			} else {
				if s.trace {
					s.logger.Info().Interface("cb", confirmationsBuffer).Msg("write already running")
				}
			}
		}
	}
}

func (s *Synapse) queueAckManSyncSection(config *syncSectionConfig, lastSavedId QueueElementIndex) {
	blockChan := config.blockChan
	doneWriteChan := config.doneWriteChan
	controlChan := config.controlChan

	blocks := make([]*Packet, 0)

	sortingHelper := func(i, j int) bool {
		return blocks[i].DbId < blocks[j].DbId
	}

	for {
		select {
		case cmd := <-controlChan:
			if cmd.terminate {
				return
			}
		case newBlocks := <-blockChan:
			if newBlocks == nil || len(*newBlocks) == 0 {
				continue
			}
			blocks = append(blocks, *newBlocks...)

			if !sort.SliceIsSorted(blocks, sortingHelper) {
				sort.Slice(blocks, sortingHelper)
			}
			min := blocks[0].DbId
			if min != lastSavedId+1 {
				if s.trace {
					s.logger.Info().Msgf("not saving pointer: minId=%d, lastSavedId=%d", min, lastSavedId)
				}
				doneWriteChan <- struct{}{}
				continue
			}

			newLastSavedId := lastSavedId
			for _, packet := range blocks {
				if packet.DbId == newLastSavedId+1 {
					newLastSavedId++
				} else {
					break
				}
			}

			// xTODO: update queue written counters
			// settings those to `newLastSavedId`
			if newLastSavedId > 0 && newLastSavedId != lastSavedId {
				blocksToConfirm := blocks[0:int(newLastSavedId-lastSavedId)]
				blocks = blocks[int(newLastSavedId-lastSavedId):]

				if s.trace {
					s.logger.Info().Interface("b2c", blocksToConfirm).
						Interface("blc", blocks).
						Send()
				}

				err := s.saveQueueWriterPtr(config.queueName, newLastSavedId)
				if err == nil {
					lastSavedId = newLastSavedId

					if s.trace {
						s.logger.Info().Interface("confirmations", blocksToConfirm).Send()
					}
					for _, packet := range blocksToConfirm {
						packet.confirmationChannel <- packet
					}
					if s.trace {
						s.logger.Info().Interface("confirmations", blocksToConfirm).Msg("done")
					}
				} else {
					s.infoChan <- ControlChanInfo{
						QueueName:                             config.queueName,
						ErrorSavingPointerInAckManSyncSection: err,
					}
				}
			} else {
				if len(blocks) > 0 {
					s.logger.Info().Interface("blocks", blocks).Send()
				}
			}

			doneWriteChan <- struct{}{}
		}
	}
}
