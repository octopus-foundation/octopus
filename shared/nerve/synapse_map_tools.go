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

import "sync"

func getOrAddItem(m *map[QueueName]chan *Packet, l *sync.RWMutex, q QueueName, g func() chan *Packet) chan *Packet {
	l.RLock()
	if el, exists := (*m)[q]; exists {
		l.RUnlock()
		return el
	}
	l.RUnlock()
	l.Lock()
	if el, exists := (*m)[q]; exists {
		l.Unlock()
		return el
	}
	el := g()
	(*m)[q] = el
	l.Unlock()
	return el
}

// getOrAddNthItem
// n - number of element
func getOrAddNthItem(m *map[QueueName][]chan *Packet, l *sync.RWMutex, q QueueName, n int, g func() []chan *Packet) chan *Packet {
	l.RLock()
	if el, exists := (*m)[q]; exists {
		if n < len(el) {
			l.RUnlock()
			return el[n]
		}
	}
	l.RUnlock()
	l.Lock()
	if el, exists := (*m)[q]; exists {
		if n < len(el) {
			l.Unlock()
			return el[n]
		}
	}
	el := g()
	(*m)[q] = el
	l.Unlock()
	return el[n]
}
