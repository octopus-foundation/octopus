/*
               .'\   /`.
             .'.-.`-'.-.`.
        ..._:   .-. .-.   :_...
      .'    '-.(o ) (o ).-'    `.
     :  _    _ _`~(_)~`_ _    _  :
    :  /:   ' .-=_   _=-. `   ;\  :
    :   :|-.._  '     `  _..-|:   :
     :   `:| |`:-:-.-:-:'| |:'   :
      `.   `.| | | | | | |.'   .'
        `.   `-:_| | |_:-'   .'
          `-._   ````    _.-'
              ``-------''

Created by ab, 24.10.2022
*/

package nerve

import (
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func BenchmarkSynapseWrite(b *testing.B) {
	const nIOThreads uint = 4

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zlog.Logger = zlog.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	backend, err := NewSMysqlBackend(SMysqlBackendConfig{
		Host:                "127.0.0.1",
		DbName:              "nerve",
		TableParallelism:    nIOThreads,
		PointersParallelism: nIOThreads,
		MaxRPSPerThread:     50,
	})
	if err != nil {
		zlog.Fatal().Err(err).Send()
	}

	s := NewSynapse(backend)

	b.ResetTimer()

	b.N = 10_000_000

	t := time.Now()
	sendData(s, b.N)

	b.Logf("sent %d messages in %v, speed = %v msg/s", b.N, time.Since(t), float64(b.N)/time.Since(t).Seconds())
}

func sendData(s *Synapse, n int) {
	const writeWorkers = 2000
	const packSize = 128

	wg := sync.WaitGroup{}
	wg.Add(writeWorkers)

	msgSent := uint64(0)
	for x := 0; x < writeWorkers; x++ {
		go func() {
			for {
				var pack = make([]*Packet, packSize)
				for i := 0; i < packSize; i++ {
					pack[i] = &Packet{
						Data: []byte("test"),
					}
				}
				_ = s.SendPack(NQLocalTest, pack)
				atomic.AddUint64(&msgSent, uint64(packSize))
			}
		}()
	}

	for {
		sent := atomic.LoadUint64(&msgSent)
		if sent >= uint64(n) {
			break
		}
	}
}

func BenchmarkSynapseRead(b *testing.B) {
	const nIOThreads uint = 4

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zlog.Logger = zlog.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	backend, err := NewSMysqlBackend(SMysqlBackendConfig{
		Host:                "127.0.0.1",
		DbName:              "nerve",
		TableParallelism:    nIOThreads,
		PointersParallelism: nIOThreads,
		MaxRPSPerThread:     50,
	})
	if err != nil {
		zlog.Fatal().Err(err).Send()
	}

	s := NewSynapse(backend)

	b.N = 10_000_000
	sendData(s, b.N)
	b.ResetTimer()

	t := time.Now()

	var msgGot uint64 = 0

	receiver := s.GetReceiver(NQLocalTest, NCTest)
	for msg := range receiver.DataChan {
		msgGot += 1
		receiver.Ack(msg)
		if msgGot >= uint64(b.N) {
			break
		}
	}

	b.Logf("read %d messages in %v, speed = %v msg/s", b.N, time.Since(t), float64(b.N)/time.Since(t).Seconds())
}
