package algorithms

import . "github.com/cem-okulmus/BalancedGo/lib"

type Algorithm interface {
	Name() string
	FindDecomp() Decomp
	FindDecompGraph(G Graph) Decomp
	SetWidth(K int)
}

type UpdateAlgorithm interface {
	Name() string
	FindDecompUpdate(currentGraph Graph, savedScenes map[uint32]SceneValue) Decomp
	SetWidth(K int)
}
