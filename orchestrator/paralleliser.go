package main

import (
	"fmt"
	"sync"
)

// run `f` in parallel on all the available nodes
func (ms *MuxService) doParallel(f func(node *NodeInfo, resultChannel chan<- interface{}, errorChannel chan<- string)) (results []interface{}, errs []string) {
	// assumes ms.nodesLock is held

	wg, readersWg := &sync.WaitGroup{}, &sync.WaitGroup{}
	resultChan := make(chan interface{})
	errorChan := make(chan string)

	wg.Add(len(ms.nodes))
	readersWg.Add(2)

	for _, n := range ms.nodes {
		go func(n *NodeInfo) {
			defer wg.Done()
			defer func() {
				if e := recover(); e != nil {
					errorChan <- fmt.Sprintf("Worker exception: %v", e)
				}
			}()
			f(n, resultChan, errorChan)
		}(n)
	}

	go func() {
		for val := range resultChan {
			results = append(results, val)
		}
		readersWg.Done()
	}()

	go func() {
		for err := range errorChan {
			errs = append(errs, err)
		}
		readersWg.Done()
	}()

	wg.Wait()

	close(resultChan)
	close(errorChan)

	readersWg.Wait()

	return
}
