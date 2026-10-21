package utils

import "container/heap"

type Item[T any] struct {
	Value    T
	Priority int
	index    int
}

type priorityHeap[T any] []*Item[T]

func (h priorityHeap[T]) Len() int           { return len(h) }
func (h priorityHeap[T]) Less(i, j int) bool { return h[i].Priority < h[j].Priority }
func (h priorityHeap[T]) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *priorityHeap[T]) Push(x any) {
	n := len(*h)
	item := x.(*Item[T])
	item.index = n
	*h = append(*h, item)
}

func (h *priorityHeap[T]) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*h = old[:n-1]
	return item
}

type PriorityQueue[T any] struct {
	h priorityHeap[T]
}

func NewPriorityQueue[T any]() *PriorityQueue[T] {
	pq := &PriorityQueue[T]{h: make(priorityHeap[T], 0)}
	heap.Init(&pq.h)
	return pq
}

func (pq *PriorityQueue[T]) Push(value T, priority int) {
	item := &Item[T]{
		Value:    value,
		Priority: priority,
	}
	heap.Push(&pq.h, item)
}

func (pq *PriorityQueue[T]) Pop() (T, int) {
	item := heap.Pop(&pq.h).(*Item[T])
	return item.Value, item.Priority
}

func (pq *PriorityQueue[T]) Len() int {
	return pq.h.Len()
}

func (pq *PriorityQueue[T]) IsEmpty() bool {
	return pq.h.Len() == 0
}
