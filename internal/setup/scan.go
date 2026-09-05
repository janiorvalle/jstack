package setup

import (
	"context"
	"sync"
)

// scanWorkers is how many git or gh commands the tracker scan runs at
// once. A hundred checkouts are five hundred git commands, and eight at a
// time keeps that to a few seconds without forking them all at once on a
// two-core machine.
const scanWorkers = 8

// forEach runs work for every index below count, scanWorkers at a time,
// and hands out no more indexes once ctx is done. Each call writes its own
// slot, so a caller reads results by index and the input's order holds
// whatever order the calls finish in. A count that runs the scan is the
// place a progress callback would wrap work.
func forEach(ctx context.Context, count int, work func(index int)) {
	indexes := make(chan int)
	var workers sync.WaitGroup
	for range min(count, scanWorkers) {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for index := range indexes {
				work(index)
			}
		}()
	}
	for index := 0; index < count && ctx.Err() == nil; index++ {
		indexes <- index
	}
	close(indexes)
	workers.Wait()
}

// readRepoStates is readRepoState for every repo, in the repos' order.
func readRepoStates(ctx context.Context, opts Options, repos []trackerRepo) []repoState {
	states := make([]repoState, len(repos))
	forEach(ctx, len(repos), func(index int) {
		states[index] = readRepoState(ctx, opts, repos[index].dir)
	})
	return states
}

// readOrigins is readOrigin for every URL, in the URLs' order.
func readOrigins(ctx context.Context, opts Options, urls []string) []originFacts {
	facts := make([]originFacts, len(urls))
	forEach(ctx, len(urls), func(index int) {
		facts[index] = readOrigin(ctx, opts, urls[index])
	})
	return facts
}
