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
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"octopus/shared/unidb"
)

type SMysqlBackendConfig struct {
	Host   string `json:"mysql-host"`
	Port   uint   `json:"port"`
	DbName string `json:"mysql-db-name"`

	// for each batch of items to be saved
	// system will try to write data to `TableParallelism` tables
	// choosing tables by calling sharing function for each key (id)
	TableParallelism    uint `json:"table-parallelism"`
	PointersParallelism uint `json:"pointers-parallelism"`
	MaxRPSPerThread     uint `json:"max-rps"`
}

type threadID QueueElementIndex
type SMysqlBackend struct {
	logger         zerolog.Logger
	config         SMysqlBackendConfig
	tableCache     map[string]struct{}
	tableCacheLock sync.RWMutex
	Db             *unidb.UniDB
	trace          bool
	rpsLock        sync.RWMutex
	rpsTimers      map[threadID]time.Time
	Batches        uint64
	Packets        uint64
	rpsTimerRead   time.Time
}

func GetMySQLBackendForQueue(queueName QueueConfig, host string) (SynapseBackend, error) {
	backendConfig, exists := queueName.Hosts[host]
	if !exists {
		return nil, fmt.Errorf("no host %s defined for queue %s", host, queueName.Name)
	}

	return NewSMysqlBackend(SMysqlBackendConfig{
		Host:                host,
		Port:                backendConfig.Port,
		DbName:              backendConfig.DbName,
		TableParallelism:    backendConfig.TableParallelism,
		PointersParallelism: backendConfig.PointersParallelism,
		MaxRPSPerThread:     backendConfig.MaxRPSPerThread,
	})
}

func NewSMysqlBackend(config SMysqlBackendConfig) (*SMysqlBackend, error) {
	cfg := unidb.NewUniDB().
		WithHost(config.Host).
		WithDB(config.DbName).
		WithParseTime().
		WithMaxIdleConns(128).
		WithMaxConns(128).
		WithIgnoreEmptyQueriesFS(true)
	if config.Port != 0 {
		cfg.WithPort(config.Port)
	}

	db, err := cfg.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed connect to database %s on %s: %w", config.DbName, config.Host, err)
	}

	// let's create databases
	// WTF: unreachable code: getting `Unknown database` error on cfg.Connect()
	_, err = db.GetRawDB().Exec(fmt.Sprintf("create database if not exists `%s`", config.DbName))
	if err != nil {
		return nil, fmt.Errorf("failed to create database %s: %w", config.DbName, err)
	}

	// let's find out what tables to we have here...
	tables := make([]string, 0)
	rows, err := db.GetRawDB().Query("show tables")
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var tblName string
		err = rows.Scan(&tblName)
		if err != nil {
			return nil, err
		}

		tables = append(tables, tblName)
	}

	logger := log.With().Str("host", config.Host).Logger()

	logger.Info().Int("n-tables", len(tables)).Send()

	return &SMysqlBackend{
		logger:         logger,
		config:         config,
		Db:             db,
		tableCache:     make(map[string]struct{}),
		tableCacheLock: sync.RWMutex{},
		trace:          true,
		rpsLock:        sync.RWMutex{},
		rpsTimers:      make(map[threadID]time.Time),
	}, nil
}

func (s *SMysqlBackend) GetHostName() string {
	return s.config.Host
}

func (s *SMysqlBackend) SetTrace(trace bool) {
	s.trace = trace
}

func (s *SMysqlBackend) ReadBatch(queueName QueueName, data []*Packet) ([]*Packet, error) {
	tables := s.getTableNamesForQueue(queueName)
	splittedBatch := make(map[QueueElementIndex][]int)
	var shardsCnt = 0
	for i := range data {
		shardIdx := data[i].DbId % QueueElementIndex(s.config.TableParallelism)
		if splittedBatch[shardIdx] == nil {
			shardsCnt += 1
		}
		splittedBatch[shardIdx] = append(splittedBatch[shardIdx], i)
	}

	result := make([]*Packet, 0, len(data))
	resultLock := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(shardsCnt)

	errors := map[QueueElementIndex]error{}
	errorsLock := sync.Mutex{}

	for shardId, offsets := range splittedBatch {
		go func(shardId QueueElementIndex, offsets []int) {
			defer wg.Done()
			if len(offsets) == 0 {
				return
			}

			ids := make([]string, 0, len(offsets))
			for _, offset := range offsets {
				ids = append(ids, fmt.Sprintf("%d", data[offset].DbId))
			}

			query := fmt.Sprintf("select id, data from %s where id in (%s)",
				tables[shardId],
				strings.Join(ids, ","))

			rows, err := s.Db.GetRawDB().Query(query)
			defer rows.Close()

			if err != nil {
				errorsLock.Lock()
				errors[shardId] = fmt.Errorf("error querying database: %v", err)
				errorsLock.Unlock()
				return
			}

			for rows.Next() {
				var id QueueElementIndex
				var msg []byte

				err = rows.Scan(&id, &msg)
				if err != nil {
					errorsLock.Lock()
					errors[shardId] = fmt.Errorf("error querying database: %v", err)
					errorsLock.Unlock()
					return
				}

				resultLock.Lock()
				result = append(result, &Packet{
					Data: msg,
					DbId: id,
				})
				resultLock.Unlock()
			}
		}(shardId, offsets)
	}

	wg.Wait()
	for _, err := range errors {
		if err != nil {
			return nil, err
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].DbId < result[j].DbId
	})

	return result, nil
}

var warningsCounter uint64

func (s *SMysqlBackend) WriteBatch(queueName QueueName, data []*Packet) error {
	// if we are allowed to make at max s.config.MaxRPSPerThread
	// calls to underlying mysql_backend for this thread, let's
	// make sure it's at least
	//   minTimeForOneCall := ((1 * time.Second) / s.config.MaxRPSPerThread)
	// passed since last call...
	err := s.ensureTablesExists(queueName)
	if err != nil {
		return err
	}

	splittedBatch := make(map[QueueElementIndex][]int)
	for i := range data {
		shardIdx := data[i].DbId % QueueElementIndex(s.config.TableParallelism)
		splittedBatch[shardIdx] = append(splittedBatch[shardIdx], i)
	}

	tables := s.getTableNamesForQueue(queueName)

	if s.trace {
		s.logger.Info().Interface("sb", splittedBatch).Msg("sending to mysql")
	}
	if len(splittedBatch) > 1 {
		if atomic.AddUint64(&warningsCounter, 1)%1 == 0 {
			s.logger.Error().Msg(`too many rows in batch: the only reason it could be io-threads and logical spreading 
threads miss-match, it happened due to the problem in the whole architecture I don't think should be solved
when I'm writing this. There are two things you should know:
- performance is much lower then it should be;
- it's impossible to ensure maxRPSPerThread is respected - actual is much lower then given.
'`)
		}
	}
	for shardId, offsets := range splittedBatch {
		// ensure maxRPSPerThread
		s.rpsLock.RLock()
		timeSinceLastCall := time.Since(s.rpsTimers[threadID(shardId)])
		s.rpsLock.RUnlock()
		timeBetweenCalls := (1 * time.Second) / time.Duration(s.config.MaxRPSPerThread)
		if timeSinceLastCall < timeBetweenCalls {
			time.Sleep(timeBetweenCalls - timeSinceLastCall)
		}
		s.rpsLock.Lock()
		s.rpsTimers[threadID(shardId)] = time.Now()
		s.rpsLock.Unlock()
		// done

		tokens := make([]string, len(offsets))
		dataRecords := make([]interface{}, len(offsets))
		size := uint64(0)
		for i, offset := range offsets {
			tokens[i] = fmt.Sprintf("(%d, ?)", data[offset].DbId)
			dataRecords[i] = data[offset].Data
			size += uint64(len(data[offset].Data))
		}

		query := fmt.Sprintf(`insert into %s (id, data) values %s on duplicate key update data=values(data)`,
			tables[shardId],
			strings.Join(tokens, ","))

		ts := time.Now()
		_, err = s.Db.GetRawDB().Exec(query, dataRecords...)
		if time.Since(ts) > 1*time.Second {
			s.logger.Warn().
				Dur("query-time", time.Since(ts)).
				Int("sid", int(shardId)).
				Int("entries", len(tokens)).
				Uint64("bytes", size).
				Float64("avg-entry", float64(size)/float64(len(dataRecords))).
				Msg("slow nerve mysql insert")
		}
		if err != nil {
			return err
		}
	}
	if s.trace {
		s.logger.Debug().Interface("db-batch", data).Msg("saved")
	}

	atomic.AddUint64(&s.Batches, 1)
	atomic.AddUint64(&s.Packets, uint64(len(data)))
	return nil
}

func getPtrKeyName(name QueueName, consumer ConsumerId) string {
	return fmt.Sprintf("%s:%s", name, consumer)
}

func (s *SMysqlBackend) WritePtr(name QueueName, consumer ConsumerId, ptr QueueElementIndex) error {
	if consumer == "" {
		// ensure maxRPSPerThread
		s.rpsLock.RLock()
		timeSinceLastCall := time.Since(s.rpsTimerRead)
		s.rpsLock.RUnlock()
		timeBetweenCalls := (1 * time.Second) / time.Duration(s.config.MaxRPSPerThread)
		if timeSinceLastCall < timeBetweenCalls {
			time.Sleep(timeBetweenCalls - timeSinceLastCall)
		}
		s.rpsLock.Lock()
		s.rpsTimerRead = time.Now()
		s.rpsLock.Unlock()
		// done
	}
	pointersTables := s.getTableNamesForPointers(name)

	query := fmt.Sprintf("insert into %s (id, ptr) values (?, ?) on duplicate key update ptr = values(ptr)",
		pointersTables[0])
	_, err := s.Db.GetRawDB().Exec(query, getPtrKeyName(name, consumer), ptr)
	if err != nil {
		return err
	}
	return nil
}

func (s *SMysqlBackend) GetPtr(name QueueName, consumer ConsumerId) (QueueElementIndex, error) {
	err := s.ensureTablesExists(name)
	if err != nil {
		return 0, err
	}

	pointersTables := s.getTableNamesForPointers(name)
	query := fmt.Sprintf("select ptr from %s where id = ?", pointersTables[0])

	rows, err := s.Db.GetRawDB().Query(query, getPtrKeyName(name, consumer))
	if err != nil {
		return 0, err
	}

	defer rows.Close()

	for rows.Next() {
		var ptr QueueElementIndex
		if err = rows.Scan(&ptr); err != nil {
			return 0, err
		}

		return ptr, nil
	}

	return 0, nil
}

func (s *SMysqlBackend) ensureTablesExists(name QueueName) error {
	dataTables := s.getTableNamesForQueue(name)
	pointersTables := s.getTableNamesForPointers(name)

	missingDataTables := make([]int, 0)
	missingPointersTables := make([]int, 0)
	s.tableCacheLock.RLock()
	for idx, tblName := range dataTables {
		if _, exists := s.tableCache[tblName]; !exists {
			missingDataTables = append(missingDataTables, idx)
		}
	}
	for idx, tblName := range pointersTables {
		if _, exists := s.tableCache[tblName]; !exists {
			missingPointersTables = append(missingPointersTables, idx)
		}
	}
	s.tableCacheLock.RUnlock()

	var err error
	if len(missingDataTables) > 0 {
		logger := s.logger.With().
			Str("queue", string(name)).
			Logger()

		for _, idx := range missingDataTables {
			tblName := dataTables[idx]

			err = s.makeTable(logger, fmt.Sprintf(`
										create table if not exists %s (
											id bigint unsigned not null,
											data longblob not null,
											primary key(id)
										)`, tblName), tblName)
			if err != nil {
				break
			}
		}

		for _, idx := range missingPointersTables {
			tblName := pointersTables[idx]

			err = s.makeTable(logger, fmt.Sprintf(`
										create table if not exists %s (
											id varchar(255) not null,
											ptr bigint unsigned not null,
											primary key(id)
										)`, tblName), tblName)
			if err != nil {
				break
			}
		}
	}

	return err
}

func (s *SMysqlBackend) makeTable(logger zerolog.Logger, query, tblName string) error {
	_, err := s.Db.GetRawDB().Exec(query)
	if err != nil {
		logger.Error().Err(err).
			Str("table", tblName).
			Msg("error creating queue table")
		return err
	}

	s.tableCacheLock.Lock()
	s.tableCache[tblName] = struct{}{}
	s.tableCacheLock.Unlock()
	return nil
}

func (s *SMysqlBackend) getTableNamesForQueue(name QueueName) []string {
	results := make([]string, s.config.TableParallelism)
	for i := uint(0); i < s.config.TableParallelism; i++ {
		results[i] = fmt.Sprintf("queue_%s_%03d_%04d", name, s.config.TableParallelism, i)
	}

	return results
}

func (s *SMysqlBackend) getTableNamesForPointers(name QueueName) []string {
	results := make([]string, s.config.PointersParallelism)
	for i := uint(0); i < s.config.PointersParallelism; i++ {
		results[i] = fmt.Sprintf("queue_%s_%03d_%04d_pointers", name,
			s.config.PointersParallelism, i)
	}

	return results
}

func (s *SMysqlBackend) GetDefaultQueueParallelism(_ QueueName) uint {
	return s.config.TableParallelism
}
