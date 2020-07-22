// Copyright © 2019 Martin Tournoij – This file is part of GoatCounter and
// published under the terms of a slightly modified EUPL v1.2 license, which can
// be found in the LICENSE file or at https://license.goatcounter.com

// Package bgrun allows simple synchronisation of goroutines.
//
// This is mostly intended for "fire and forget" type of goroutines like sending
// an email. They typically don't really need any synchronisation as such but
// you do want to wait for them to finish before the program exits, or you want
// to wait for them in tests.
package bgrun

import (
	"context"
	"sync"
	"time"

	"zgo.at/errors"
	"zgo.at/zlog"
	"zgo.at/zstd/zsync"
)

var (
	wg      = new(sync.WaitGroup)
	maxWait = 10 * time.Second
)

// Wait for all goroutines to finish for a maximum of maxWait.
func Wait() error {
	ctx, c := context.WithTimeout(context.Background(), maxWait)
	defer c()
	return errors.Wrap(zsync.Wait(ctx, wg), "bgrun.Wait")
}

// WaitAndLog calls Wait() and logs any errors.
func WaitAndLog() {
	err := Wait()
	if err != nil {
		zlog.Error(err)
	}
}

// Run the function in a goroutine.
//
//   bgrun.Run(func() {
//       // Do work...
//	 })
func Run(f func()) {
	done := Add()
	go func() {
		defer zlog.Recover()
		defer done()
		f()
	}()
}

// Add a new function to the waitgroup and return the done.
//
//    done := bgrun.Add()
//    go func() {
//       defer done()
//       defer zlog.Recover()
//    }()
func Add() func() {
	wg.Add(1)
	return func() { wg.Done() }
}
