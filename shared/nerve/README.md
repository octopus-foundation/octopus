# Nerve queues
https://github.com/octopus-foundation/octopus/tree/main/shared/nerve

Nerve is a fast, persistent, infinite, sequential data storage facility within the existing infrastructure (currently - MySQL + b2 with the capacity to add any other databases)

[toc]

# Speed

Test environment:
- M1 Max MacBook Pro 2021
- MySQL 8.0.31 with the following config https://github.com/octopus-foundation/octopus/blob/main/ansible/playbooks/roles/app-nerve-mysql/templates/mysql.cnf

| Action  | Messages/second |
|---------|-----------------|
| Writing | 490 000         |
| Reading | 650 000         |


Benchmarks:
- https://github.com/octopus-foundation/octopus/blob/main/parts/tools/bin/synapse-bench/synapse-bench.go
- https://github.com/octopus-foundation/octopus/blob/main/shared/nerve/synapse_bench_test.go

# Usage

## Queue definition

First, you need to define your queue and backend, within the code is the best place to define your queue:
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
- your queue id is `NQLocalTest`
- queue will be used only on the local MySQL server `127.0.0.1` (you can use one queue on multiple servers with different configs)
- queue will be stored in the database named `nerve`
- queue will use 4 shard tables to store queue entries
- there should be no more than 50 queries per thread per second

*Important*:
Make sure that you are using fine-tuned MySQL with a config like this:
https://github.com/octopus-foundation/octopus/blob/main/ansible/playbooks/roles/app-nerve-mysql/templates/mysql.cnf

## Queue publishing

```go
backend, err := nerve.GetMySQLBackendForQueue(nerve.NQLocalTest, "127.0.0.1")
if err != nil {
	// do whatever you want
}
synapse := nerve.NewSynapse(backend)

// now you can re-use synapse on any place in your code to publish

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

We have the following queue publishing methods in synapse:
- `SendPack` - for batch sending
- `SendProtoPack` - for protobuf-encoded packet sending
- `SendSourcedPack` - for `nerve.NerveSourcedPacket` sending - packet with extra metadata (source type, source id)
- `Send` - for single packet sending, not as fast as batch sending

*Important*:
- we don't use transactions for packet writing, so it's impossible to roll anything back
- you should only ever have ONE publisher per queue, to maintain the sequence
- you should never change queue parallelism on-the-fly

If you want to use `nerve` as eventbus with MySQL, the following architecture is recommended:
- use `proxy` table for publishing events in transaction from your application
- create an app that will read the events from `proxy` table and publish them to the nerve queue

In this case:
- you will have only one publisher for a queue
- you will have transactional event publishing

## Queue reading

To read the data you should define a unique consumer in your code near the queue definition:
```go
const (
	NCTest ConsumerId = "NerveConsumerId_Test"
)
```

After that you can read your data:

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
- `DbId`is a monotonic incremented uint64 counter, assigned on writing
- `Ack` is an async operation, so on restart you can lose previously ack-ed data (you need to store and check last processed DbId)
- Ack is thread-safe

# MySQL backend

Nerve MySQL backend efficiently uses the InnoDB engine with sharding for storing data. For the example above we have the following MySQL tables:
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

Queue pointers:
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

When nerve receives a packet (or a batch of packets) for writing, it will:
- find a queue running channel (based on queue id)
- add a packet to this channel
- wait for write confirmation

Queue worker will:
- get the last pointer from the backend on start
- read the new packet for writing from the channel
- find a shard channel based on `DbId % TableParallelism`
- add a packet to this shard channel

Shard worker, in its turn, will react to the packet and do the following:
- add it to the buffer
- if we can write (`MaxRPS`-restricted for best io performance):
  - write the buffer to the MySQL table shard
  - `ack` all messages in the buffer

Queue ack manager worker, after getting `ack` of the packet from the channel, will take the following actions:
- add ack-ed id to the buffer
- find max sequential number of the ack-ed buffer
- update database pointer
- send write confirmation for all sequential ack-ed packets

## Reading

On start, the synapse receiver reads the latest consumed pointer. After that, in a continuous loop it will:
- read the last "writer" pointer
- read data from the backend in the range between the reader pointer and writer pointer (with backend limitations)
- send this data to the reader channel one-by-one, ordered by `DbId`

What about ack? It's simple and works in exactly the same way as for writing:
- add ack-ed id to the buffer
- find max sequential number of the ack-ed buffer
- update database pointer

So, if you process messages in a random order – the ack manager will wait for ack ids to be sequential.