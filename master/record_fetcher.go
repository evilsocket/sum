package master

import (
	"context"
	. "github.com/evilsocket/sum/proto"
	"sync"
)

// fetches records in a parallel way
type recordFetcher struct {
	ctx           context.Context
	cf            context.CancelFunc
	wg, readersWg *sync.WaitGroup
	errCh         chan string
	resCh         chan []*Record

	Errs    []string
	Records []*Record
}

func NewRecordFetcher() *recordFetcher {
	f := &recordFetcher{}
	f.ctx, f.cf = newCommContext()
	f.wg = &sync.WaitGroup{}
	f.readersWg = &sync.WaitGroup{}
	f.errCh, f.resCh = make(chan string), make(chan []*Record)

	f.readersWg.Add(2)

	go func() {
		defer f.readersWg.Done()
		for err := range f.errCh {
			f.Errs = append(f.Errs, err)
		}
	}()
	go func() {
		defer f.readersWg.Done()
		for r := range f.resCh {
			f.Records = append(f.Records, r...)
		}
	}()

	return f
}

func (f *recordFetcher) _fetch(node *NodeInfo, page, perPage uint64) {
	defer f.wg.Done()

	arg := &ListRequest{PerPage: perPage, Page: page}
	if resp, err := node.Client.ListRecords(f.ctx, arg); err != nil {
		f.cf()
		f.errCh <- err.Error()
	} else {
		f.resCh <- resp.Records
	}
}

func (f *recordFetcher) Fetch(node *NodeInfo, page, perPage uint64) {
	f.wg.Add(1)
	go f._fetch(node, page, perPage)
}

func (f *recordFetcher) Cancel() {
	f.cf()
}

func (f *recordFetcher) Wait() {
	f.wg.Wait()
	close(f.errCh)
	close(f.resCh)
	f.readersWg.Wait()
}
