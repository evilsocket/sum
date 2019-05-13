package master

import (
	"fmt"
	"sync"
)

// run `f` in parallel on the given nodes
func doParallel(nodes []*NodeInfo, f func(node *NodeInfo, resultChannel chan<- interface{}, errorChannel chan<- string)) (results []interface{}, errs []string) {
	wg, readersWg := &sync.WaitGroup{}, &sync.WaitGroup{}
	resultChan := make(chan interface{})
	errorChan := make(chan string)
	nNodes := len(nodes)

	if nNodes == 0 {
		return nil, nil
	}

	worker := func(n *NodeInfo) {
		defer wg.Done()
		defer func() {
			if e := recover(); e != nil {
				errorChan <- fmt.Sprintf("Worker exception: %v", e)
			}
		}()
		f(n, resultChan, errorChan)
	}

	resultReader := func() {
		for val := range resultChan {
			results = append(results, val)
		}
		readersWg.Done()
	}
	errorReader := func() {
		for err := range errorChan {
			errs = append(errs, err)
		}
		readersWg.Done()
	}

	wg.Add(nNodes)
	readersWg.Add(2)

	for _, n := range nodes {
		go worker(n)
	}

	go resultReader()
	go errorReader()

	wg.Wait()

	close(resultChan)
	close(errorChan)

	readersWg.Wait()

	return
}

// run `f` in parallel on all the available nodes
// NB: assumes an held lock on ms.nodesLock
func (ms *Service) doParallel(f func(node *NodeInfo, resultChannel chan<- interface{}, errorChannel chan<- string)) (results []interface{}, errs []string) {
	// assumes ms.nodesLock is held
	return doParallel(ms.nodes, f)
}
