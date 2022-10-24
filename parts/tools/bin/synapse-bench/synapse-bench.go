package main

import (
	"flag"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"octopus/shared/nerve"
	"os"
	"sync/atomic"
	"time"
)

var nIOThreads = flag.Uint("io-threads", 4, "number of IO threads")

func main() {
	flag.Parse()
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zlog.Logger = zlog.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	backend, err := nerve.NewSMysqlBackend(nerve.SMysqlBackendConfig{
		Host:                "127.0.0.1",
		DbName:              "nerve",
		TableParallelism:    *nIOThreads,
		PointersParallelism: *nIOThreads,
		MaxRPSPerThread:     50,
	})
	if err != nil {
		zlog.Fatal().Err(err).Send()
	}

	s := nerve.NewSynapse(backend)
	s.SetTrace(false)

	ts := time.Now()
	msgSent := uint64(0)
	for x := 0; x < 1_000; x++ {
		go func() {
			for {
				var pack = make([]*nerve.Packet, 200)
				for i := 0; i < 200; i++ {
					pack[i] = &nerve.Packet{
						Data: []byte("test"),
					}
				}
				_ = s.SendPack(nerve.NQLocalTest, pack)
				atomic.AddUint64(&msgSent, uint64(200))
			}
		}()
	}

	msgGot := uint64(0)
	msgBytesGot := uint64(0)
	go func() {
		receiver := s.GetReceiver(nerve.NQLocalTest, nerve.NCTest)
		for msg := range receiver.DataChan {
			atomic.AddUint64(&msgGot, 1)
			atomic.AddUint64(&msgBytesGot, uint64(len(msg.Data)))
			receiver.Ack(msg)
		}
	}()

	for {
		zlog.Info().Uint64("sent", msgSent).
			Uint64("rcvd", msgGot).
			Msgf("writes at %f, batch size of %f; reading at %f.",
				float64(msgSent)*float64(time.Second)/float64(time.Since(ts)),
				float64(backend.Packets)/float64(backend.Batches),
				float64(msgGot)*float64(time.Second)/float64(time.Since(ts)),
			)
		time.Sleep(1 * time.Second)
	}
}
