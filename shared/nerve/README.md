# Nerve queues
https://github.com/octopus-foundation/octopus/tree/main/shared/nerve

Nerve is fast, persistent, infinite, sequential data storage over existing infrastructure (currently - mysql + b2 with possibility to add any other databases)

[toc]

# Speed

Test environment:
- M1 Max MacBook Pro 2021
- mysql 8.0.31 with following config https://github.com/octopus-foundation/octopus/blob/main/ansible/playbooks/roles/app-nerve-mysql/templates/mysql.cnf

Write speed: around ~490 000 messages per second
Read speed: around ~650 000 messages per second

Benchmarks:
- https://github.com/octopus-foundation/octopus/blob/main/parts/tools/bin/synapse-bench/synapse-bench.go
- https://github.com/octopus-foundation/octopus/blob/main/shared/nerve/synapse_bench_test.go

# Usage

## Queues definition

First, you need to define your queue and backend, the code is best place to define our queue with config:
```go
package nerve

var NQLocalTest = QueueConfig{
	Name: "NQLocalTest",
	Hosts: map[string]BackendConfig{
		"127.0.0.1": {
			DbName:              "nerve",
			Port:                3306,
			TableParallelism:    4,
			PointersParallelism: 1,
			MaxRPSPerThread:     50,
		},
	},
}
```

In this config we specified the following:
- our queue id is `NQLocalTest`
- queue used only on local mysql instance on server "127.0.0.1" (we can use one queue on multiple servers with different configs)
- queue will be stored in database named `nerve`
- we will use 4 shard tables for storing queue entries
- maximum queries per thread should be 50

*Important*:
Make sure you are using fine-tuned mysql with config like this:
https://github.com/octopus-foundation/octopus/blob/main/ansible/playbooks/roles/app-nerve-mysql/templates/mysql.cnf

## Queue publishing

```go
backend, err := nerve.GetMySQLBackendForQueue(nerve.NQLocalTest, "127.0.0.1")
if err != nil {
	// do whatever you want
}
synapse := nerve.NewSynapse(backend)

// now we can re-use synapse on any place on our code to publish

var pack = make([]*nerve.Packet, 200)
for i := 0; i < 200; i++ {
    pack[i] = &nerve.Packet{
      Data: []byte("test"),
  }
}
err = synapse.SendPack(nerve.NQLocalTest, pack)
if err != nil {
  // handle error
}
```

We have following queue publish methods in synapse:
- `SendPack` - for batch sending
- `SendProtoPack` - for protobuf-encoded packets sending
- `SendSourcedPack` - for `nerve.NerveSourcedPacket` sending - packet with extra metadata (source type, source id)
- `Send` - for single packet sending, not so fast as packs

*Important*:
- we doesn't use transactions for packet writing, so it's impossible to rollback anything after writing
- you should always have only ONE publisher per queue, for sequence maintaining
- you should nether change queue parallelism on-the-fly

If you want to use `nerve` as eventbus with mysql, the following architecture is recommended:
- use `proxy` table for publishing events in transaction from your application
- create app, which will read events from `proxy` table and publish them to nerve queue

In this case:
- you will have only one publisher for queue
- you will have transactional event publishing

## Queue reading

For reading data you should define unique consumer in code, near queue definition:
```go
const (
	NCTest ConsumerId = "NerveConsumerId_Test"
)
```

Just to have control over all known consumers. And after that you can read data:

```go
backend, err := nerve.GetMySQLBackendForQueue(nerve.NQLocalTest, "127.0.0.1")
if err != nil {
	// do whatever you want
}
synapse := nerve.NewSynapse(backend)
receiver := synapse.GetReceiver(nerve.NQLocalTest, nerve.NCTest)
for msg := range receiver.DataChan {
  // got msg
  log.Printf("Got msg %v", msg.DbId)
  receiver.Ack(msg)
}
```

*Important*:
- `DbId` - monotonical, incremented uint64 counter, assigned on writing
- `Ack` is async operation, so on restart you can lose previously acked data (you need to store and check last processed DbId)
- Ack is thread-safe

# MySQL backend

Nerve mysql backend efficiently uses InnoDB engine with sharing for storing data.
For our example above, we have following mysql tables:
```
mysql> show tables;
+-------------------------------------+
| Tables_in_nerve                     |
+-------------------------------------+
| queue_NQLocalTest_004_0000          |
| queue_NQLocalTest_004_0000_pointers |
| queue_NQLocalTest_004_0001          |
| queue_NQLocalTest_004_0001_pointers |
| queue_NQLocalTest_004_0002          |
| queue_NQLocalTest_004_0002_pointers |
| queue_NQLocalTest_004_0003          |
| queue_NQLocalTest_004_0003_pointers |
+-------------------------------------+
8 rows in set (0.00 sec)
```

Where:
- `queue_NQLocalTest_004_000*` - tables for storing queue entries
- `queue_NQLocalTest_004_000*_pointers` - tables for storing queue pointers

Queues pointers:
```
mysql> select * from queue_NQLocalTest_004_0000_pointers;
+----------------------------------+----------+
| id                               | ptr      |
+----------------------------------+----------+
| NQLocalTest:                     | 26652582 |
| NQLocalTest:NerveConsumerId_Test | 19989254 |
+----------------------------------+----------+
2 rows in set (0.01 sec)
```

## Writing

When nerve get packet (or batch of packets) for writing, it will:
- find queue running channel (based on queue id)
- add message to this channel
- wait for write confirmation

Queue worker will:
- get last pointer from backend on start
- if got new message for writing from channel:
    - finds shard number based on `DbId % TableParallelism`
    - add message to shard channel

Shard worker, in turn, react on message and do the following:
- add message to buffer
- if we can write (`MaxRPS`-restricted for best io performance):
    - write buffer to mysql table shard
    - `ack` all messages in buffer

Queue ack manager worker, after getting `ack` of message from channel, do the following:
- add acked id to buffer
- find max sequential number of acked buffer
- update database pointer
- send write confirmation for all sequential ack-ed messages

## Reading

Synapse receiver on start reads the latest pointer for consumer
After that in forever loop it will:
- read last "writer" pointer
- read data from backend in range between reader pointer and writer pointer (with backend limitations)
- send this data to reader channel one-by-one, ordered by DbId

What about ack? It's simple and works exactly the same as for writing:
- add acked id to buffer
- find max sequential number of acked buffer
- update database pointer

So if you process messages in random order - ack manager will wait for Ack ids to be sequential.