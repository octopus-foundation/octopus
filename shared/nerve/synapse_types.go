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
	"github.com/rs/zerolog"
	"sync"
)

type BackendConfig struct {
	DbName              string `json:"dbname"`
	Port                uint   `json:"port"`
	TableParallelism    uint   `json:"table_parallelism"`
	PointersParallelism uint   `json:"pointers_parallelism"`
	MaxRPSPerThread     uint   `json:"max_rps_per_thread"`
}

type QueueConfig struct {
	Hosts map[string]BackendConfig `json:"hosts"`
	Name  QueueName                `json:"name"`
}

type Synapse struct {
	infoChan                chan ControlChanInfo
	queueChannels           map[QueueName]chan *Packet
	queueChannelsLock       sync.RWMutex
	controlChan             chan ControlCommand
	queueAckManChannels     map[QueueName]chan *Packet
	queueAckManChannelsLock sync.RWMutex
	controlChannels         []chan ControlCommand
	controlChannelsLock     sync.RWMutex
	queueWriterChannels     map[QueueName][]chan *Packet
	queueWriterChannelsLock sync.RWMutex
	Backend                 SynapseBackend
	trace                   bool
	logger                  *zerolog.Logger
}

type QueueElementIndex int64
type QueueName string
type ConsumerId string

type Packet struct {
	dataHash            [64]byte
	queueId             uint64
	confirmationChannel chan *Packet
	Data                []byte
	DbId                QueueElementIndex
}

type ControlChanInfo struct {
	QueueName                             QueueName
	ErrorSavingPointerInAckManSyncSection error
}

type ControlCommand struct {
	terminate bool
}

type Receiver struct {
	DataChan              chan *Packet
	TerminateReceiverChan chan struct{}
	TerminateReaderChan   chan struct{}
	ConsumerId            ConsumerId
	QueueName             QueueName
	Synapse               *Synapse
	ackBuffer             []QueueElementIndex
	lastAckedId           QueueElementIndex
	AckChannel            chan QueueElementIndex
	ackBufferLock         sync.RWMutex
	lastReadId            QueueElementIndex
	logger                *zerolog.Logger
}

type SynapseBackend interface {
	WriteBatch(name QueueName, data []*Packet) error
	WritePtr(name QueueName, consumer ConsumerId, ptr QueueElementIndex) error
	GetDefaultQueueParallelism(name QueueName) uint
	GetPtr(name QueueName, consumer ConsumerId) (QueueElementIndex, error)
	ReadBatch(name QueueName, data []*Packet) ([]*Packet, error)
	SetTrace(trace bool)
	GetHostName() string
}
