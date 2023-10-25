/*
 * Jacobin VM - A Java virtual machine
 * Copyright (c) 2022-3 by the Jacobin Authors. All rights reserved.
 * Licensed under Mozilla Public License 2.0 (MPL 2.0)
 */

package thread

import (
	"jacobin/globals"
	"sync"
	"testing"
)

func TestCreateThread(t *testing.T) {
	et := CreateThread()
	if et.ID < 1 ||
		et.Trace != false {
		t.Error("Invalid thread generated by CreateThread()")
	}
}

func TestAddThreadsToTable(t *testing.T) {
	globals.InitGlobals("test")
	gl := globals.GetGlobalRef()
	tbl := gl.Threads

	for i := 0; i < 10; i++ {
		th := CreateThread()
		th.AddThreadToTable(gl)
	}

	tblLen := len(tbl)
	if tblLen != 10 {
		t.Errorf("Expected thread table to have 10 elements; got %d",
			tblLen)
	}

	if gl.ThreadNumber != 10 {
		t.Errorf("Expected last inserted thread to be 10; got %d", gl.ThreadNumber)
	}
}

// Following test validates that the use of the mutex on addition of
// threads to the thread table works correctly. It starts four
// goroutines that each add 100 threads to the same table. It uses
// a wait group to wait for the four routines to finish, then gets
// the size of the table and validates that it = 400.

// Following test mostly generated by ChatGPT. Per ChatGPT, it is
// provided under the Creative Commons CC0 1.0 Universal (CC0 1.0)
// license, which allows all forms or reuse and does not require
// attribution. Nonetheless, we feel it right that we attribute it
// properly.
func TestAddingMultipleSimultaneousThreads(t *testing.T) {
	// t.Parallel()

	// Define test parameters
	numThreads := 4
	threadsToAdd := 100
	expectedSize := numThreads * threadsToAdd

	// initialize globals
	globals.InitGlobals("test")

	// Initialize WaitGroup and a channel for signaling completion
	wg := sync.WaitGroup{}
	channel := make(chan struct{})

	for i := 0; i < numThreads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			addThreadsToTable(threadsToAdd)
		}()
	}

	go func() {
		wg.Wait()
		close(channel)
	}()

	// Wait for completion using a channel
	select {
	case <-channel:
		size := len(globals.GetGlobalRef().Threads)
		if size != expectedSize {
			t.Errorf("Expecting thread table size of %d, got %d", expectedSize, size)
		}
	}
}

// part of previous test. Adds 100 threads to the global thread table.
func addThreadsToTable(numThreads int) {
	glob := globals.GetGlobalRef()

	for i := 0; i < numThreads; i++ {
		th := CreateThread()
		th.AddThreadToTable(glob)
	}
}
