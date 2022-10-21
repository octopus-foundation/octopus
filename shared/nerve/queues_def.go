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

Created by ab, 21.10.2022
*/

package nerve

const (
	NCTest ConsumerId = "NerveConsumerId_Test"
)

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
