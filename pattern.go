package melrose

import (
	"fmt"
)

type GroupBy struct {
	sizes []int
}

func (g GroupBy) RequiredSequenceSize() int {
	sum := 0
	for _, size := range g.sizes {
		sum += size
	}
	return sum
}

func (g GroupBy) Transform(seq Sequence) Sequence {
	if len(g.sizes) == 0 {
		return seq
	}
	if seq.Size() != g.RequiredSequenceSize() {
		panic(fmt.Sprintf("mismatch in group size:%v and sequence size:%v",
			g.RequiredSequenceSize(), seq.Size()))
	}
	notes := [][]Note{}
	sizeIndex := 0
	group := []Note{}
	seq.NotesDo(func(n Note) {
		if len(group) < g.sizes[sizeIndex] {
			group = append(group, n)
		} else { // group full
			notes = append(notes, group)
			sizeIndex++
			group = []Note{n}
		}
	})
	notes = append(notes, group)
	return Sequence{notes}
}
