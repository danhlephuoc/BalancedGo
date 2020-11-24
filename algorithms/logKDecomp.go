// Parallel Algorithm for computing HD with log-depth recursion depth
package algorithms

import (
	"fmt"
	"log"
	"reflect"
	"runtime"

	. "github.com/cem-okulmus/BalancedGo/lib"
)

type LogKDecomp struct {
	Graph Graph
	K     int
	cache Cache
	// cache     map[uint32]*CompCache
	// cacheMux  sync.RWMutex
	BalFactor int
}

func (l *LogKDecomp) SetWidth(K int) {
	l.K = K
}

// func (l *LogKDecomp) addPositive(sep []int, comp Graph, Sp []Special) {
// 	l.cacheMux.Lock()
// 	l.cache[IntHash(sep)].Succ = append(l.cache[IntHash(sep)].Succ, comp.Edges.HashExtended(Sp))
// 	l.cacheMux.Unlock()
// }

// func (l *LogKDecomp) addNegative(sep []int, comp Graph, Sp []Special) {
// 	l.cacheMux.Lock()
// 	l.cache[IntHash(sep)].Fail = append(l.cache[IntHash(sep)].Fail, comp.Edges.HashExtended(Sp))
// 	l.cacheMux.Unlock()
// }

// func (l *LogKDecomp) checkNegative(sep []int, comp Graph, Sp []Special) bool {
// 	l.cacheMux.RLock()
// 	defer l.cacheMux.RUnlock()

// 	compCachePrev, _ := l.cache[IntHash(sep)]
// 	for i := range compCachePrev.Fail {
// 		if comp.Edges.HashExtended(Sp) == compCachePrev.Fail[i] {
// 			log.Println("Comp ", comp, "(hash ", comp.Edges.Hash(), ")  known as negative for sep ", sep)
// 			return true
// 		}
// 	}

// 	return false
// }

// func (l *LogKDecomp) checkPositive(sep []int, comp Graph, Sp []Special) bool {
// 	l.cacheMux.RLock()
// 	defer l.cacheMux.RUnlock()

// 	compCachePrev, _ := l.cache[IntHash(sep)]
// 	for i := range compCachePrev.Fail {
// 		if comp.Edges.HashExtended(Sp) == compCachePrev.Succ[i] {
// 			//  log.Println("Comp ", comp, " known as negative for sep ", sep)
// 			return true
// 		}

// 	}

// 	return false
// }

func (l LogKDecomp) Name() string {
	return "LogKDecomp"
}

// func (l LogKDecomp) FindDecompSeq(K int) Decomp {
// 	l.cache = make(map[uint32]*CompCache)
// 	return l.findHD(l.Graph, []Special{}, []int{}, l.Graph.Edges)
// }

func (l LogKDecomp) FindDecomp() Decomp {
	// l.cache = make(map[uint32]*CompCache)
	l.cache.Init()
	return l.findHDParallel(l.Graph, []int{}, l.Graph.Edges)
}

func (l LogKDecomp) FindDecompGraph(Graph Graph) Decomp {
	l.Graph = Graph
	return l.FindDecomp()
}

// determine whether we have reached a (positive or negative) base case
func (l LogKDecomp) baseCaseCheck(lenE int, lenSp int, lenAE int) bool {
	if lenE <= l.K && lenSp == 0 {
		return true
	}
	if lenE == 0 && lenSp == 1 {
		return true
	}
	if lenE == 0 && lenSp > 1 {
		return true
	}
	if lenAE == 0 && (lenE+lenSp) >= 0 {
		return true
	}

	return false
}

func (l LogKDecomp) baseCase(H Graph, lenAE int) Decomp {
	// log.Printf("Base case reached. Number of Special Edges %d\n", len(Sp))
	var output Decomp

	// cover faiure cases

	if H.Edges.Len() == 0 && len(H.Special) > 1 {
		return Decomp{}
	}
	if lenAE == 0 && (H.Len()) >= 0 {
		return Decomp{}
	}

	// construct a decomp in the remaining two

	if H.Edges.Len() <= l.K && len(H.Special) == 0 {
		output = Decomp{Graph: H, Root: Node{Bag: H.Vertices(), Cover: H.Edges}}
	}
	if H.Edges.Len() == 0 && len(H.Special) == 1 {
		sp1 := H.Special[0]
		output = Decomp{Graph: H,
			Root: Node{Bag: sp1.Vertices(), Cover: sp1}}

	}

	return output
}

//attach the two subtrees to form one
func attachingSubtrees(subtreeAbove Node, subtreeBelow Node, connecting Edges) Node {
	// log.Println("Two Nodes enter: ", subtreeAbove, subtreeBelow)
	// log.Println("Connecting: ", PrintVertices(connecting.Vertices))
	// up = Inter(up, parent.Vertices())
	//finding connecting leaf in parent
	leaf := subtreeAbove.CombineNodes(subtreeBelow, connecting)

	if leaf == nil {
		fmt.Println("\n \n Connection ", PrintVertices(connecting.Vertices()))
		fmt.Println("subtreeAbove ", subtreeAbove)

		log.Panicln("subtreeAbove doesn't contain connecting node!")
	}

	// //attaching subtree to parent
	// leaf.Children = []Node{subtree}
	// log.Println("Leaf ", leaf)
	// log.Println("One Node leaves: ", leaf)
	return *leaf
}

// sequential version, for developement purposes only
func (l LogKDecomp) findHD(H Graph, Conn []int, allowedFull Edges) Decomp {

	log.Printf("\n\nCurrent SubGraph: %v\n", H)
	log.Printf("Current Allowed Edges: %v\n", allowedFull)
	log.Println("Conn: ", PrintVertices(Conn), "\n\n")

	// Base Case
	if l.baseCaseCheck(H.Edges.Len(), len(H.Special), allowedFull.Len()) {
		return l.baseCase(H, allowedFull.Len())
	}
	//all vertices within (H ∪ Sp)
	Vertices_H := append(H.Vertices())

	allowed := FilterVertices(allowedFull, Vertices_H)

	// Set up iterator for child
	genChild := GetCombin(allowed.Len(), l.K)

	// checks all possibles nodes in H, together with PARENT loops, it covers all parent-child pairings
CHILD:
	for genChild.HasNext() {

		childλ := GetSubset(allowed, genChild.Combination)
		genChild.Confirm()

		comps_c, _, _ := H.GetComponents(childλ)

		// Check if child is balanced
		for i := range comps_c {
			if comps_c[i].Len() > (H.Len())/2 {
				continue CHILD
				// lowFound = true
				// comp_low_index = i //keep track of the index for composing comp_up later
				// comp_low = comps_p[i]
				// compSp_low = compsSp_p[i]
			}

		}

		log.Println("Balanced Child found, ", childλ)

		// Check if child is possible root
		if Subset(Conn, childλ.Vertices()) {
			// log.Println("parent breaks connectivity: ")
			// log.Println("Conn: ", PrintVertices(Conn))
			// log.Println("V(parentλ) = ", PrintVertices(parentλ.Vertices()))
			// continue PARENT

			log.Printf("Child-Root cover chosen: %v\n", Graph{Edges: childλ})
			log.Printf("Comps of Child-Root: %v\n", comps_c)

			childχ := Inter(childλ.Vertices(), Vertices_H)

			// check chache for previous encounters
			if l.cache.CheckNegative(childλ, comps_c) {
				log.Println("Skipping a child sep", childχ)
				continue CHILD
			}

			// // check chache for previous encounters
			// l.cacheMux.RLock()
			// _, ok := l.cache[IntHash(childχ)]
			// l.cacheMux.RUnlock()
			// if !ok {
			// 	var newCache CompCache
			// 	l.cacheMux.Lock()
			// 	l.cache[IntHash(childχ)] = &newCache
			// 	l.cacheMux.Unlock()

			// } else {
			// 	for j := range comps_c {
			// 		if l.checkNegative(childχ, comps_c[j], Sp) { //TODO: Add positive check and cutNodes
			// 			log.Println("Skipping a child sep", childχ)
			// 			continue CHILD
			// 		}

			// 	}
			// }

			var subtrees []Node
			for y := range comps_c {
				V_comp_c := comps_c[y].Vertices()
				Conn_y := Inter(V_comp_c, childχ)

				decomp := l.findHD(comps_c[y], Conn_y, allowedFull)
				if reflect.DeepEqual(decomp, Decomp{}) {
					log.Println("Rejecting child-root")
					log.Printf("\nCurrent SubGraph: %v\n", H)
					log.Printf("Current Allowed Edges: %v\n", allowed)
					log.Println("Conn: ", PrintVertices(Conn), "\n\n")
					l.cache.AddNegative(childλ, comps_c[y])
					continue CHILD
				}

				// log.Printf("Produced Decomp w Child-Root: %+v\n", decomp)
				subtrees = append(subtrees, decomp.Root)
			}

			root := Node{Bag: childχ, Cover: childλ, Children: subtrees}
			return Decomp{Graph: H, Root: root}
		}

		alowedParent := FilterVertices(allowed, append(Conn, childλ.Vertices()...))

		genParent := GetCombin(alowedParent.Len(), l.K)

	PARENT:
		for genParent.HasNext() {

			// parentλ := GetSubsetMap(allowed, genParent.Combination, parentMap)
			parentλ := GetSubset(alowedParent, genParent.Combination)
			// parentλ := GetSubset(allowedParent, genParent.Combination)
			genParent.Confirm()

			comps_p, _, isolatedEdges := H.GetComponents(parentλ)

			foundLow := false
			var comp_low_index int
			var comp_low Graph

			// Check if parent is un-balanced
			for i := range comps_p {
				if comps_p[i].Len() > H.Len()/2 {
					foundLow = true
					comp_low_index = i //keep track of the index for composing comp_up later
					comp_low = comps_p[i]
				}
			}
			if !foundLow {
				continue PARENT
			}

			vertCompLow := comp_low.Vertices()
			childχ := Inter(childλ.Vertices(), vertCompLow)

			// Connectivity checks

			if !Subset(Inter(vertCompLow, Conn), parentλ.Vertices()) {
				// log.Println("Conn not covered by parent")

				// log.Println("Conn: ", PrintVertices(Conn))
				// log.Println("V(parentλ) \\cap Conn", PrintVertices(Inter(parentλ.Vertices(), Conn)))
				// log.Println("V(Comp_low) \\cap Conn ", PrintVertices(Inter(vertCompLow, Conn)))
				continue PARENT
			}

			// Connectivity check
			if !Subset(Inter(vertCompLow, parentλ.Vertices()), childχ) {
				// log.Println("Child not connected to parent!")
				// log.Println("Parent lambda: ", PrintVertices(parentλ.Vertices()))
				// log.Println("Child lambda: ", PrintVertices(childλ.Vertices()))

				// log.Println("Child", childλ)

				continue PARENT
			}

			// determine which componenents of child are inside comp_low

			//CHECK IF THIS ACTUALLY MAKES A DIFFERENCE
			comps_c, _, _ := comp_low.GetComponents(childλ)

			//omitting the check for balancedness as it's guaranteed to still be conserved at this point

			// check chache for previous encounters
			if l.cache.CheckNegative(childλ, comps_c) {
				// log.Println("Skipping a child sep", childχ)
				continue PARENT
			}

			// l.cacheMux.RLock()
			// _, ok := l.cache[IntHash(childχ)]
			// l.cacheMux.RUnlock()
			// if !ok {
			// 	var newCache CompCache
			// 	l.cacheMux.Lock()
			// 	l.cache[IntHash(childχ)] = &newCache
			// 	l.cacheMux.Unlock()

			// } else {
			// 	for j := range comps_c {
			// 		if l.checkNegative(childχ, comps_c[j], Sp) { //TODO: Add positive check and cutNodes
			// 			// log.Println("Skipping a child sep", childχ)
			// 			continue PARENT
			// 		}
			// 	}
			// }

			log.Printf("Parent Found: %v (%s) \n", Graph{Edges: parentλ}, PrintVertices(parentλ.Vertices()))

			log.Printf("Child chosen: %v (%s) \n", Graph{Edges: childλ}, PrintVertices(childχ))
			log.Printf("Comps of Child: %v\n", comps_c)

			//Computing subcomponents of Child

			var subtrees []Node
			for x := range comps_c {
				Conn_x := Inter(comps_c[x].Vertices(), childχ)

				decomp := l.findHD(comps_c[x], Conn_x, allowedFull)
				if reflect.DeepEqual(decomp, Decomp{}) {
					l.cache.AddNegative(childλ, comps_c[x])
					log.Println("Rejecting child")
					continue PARENT
				}

				log.Printf("Produced Decomp: %+v\n", decomp)
				subtrees = append(subtrees, decomp.Root)

			}

			//Computing upper component

			var comp_up Graph
			var decompUp Decomp
			var specialChild Edges
			tempEdgeSlice := []Edge{}
			tempSpecialSlice := []Edges{}

			tempEdgeSlice = append(tempEdgeSlice, isolatedEdges...)
			for i := range comps_p {
				if i != comp_low_index {
					tempEdgeSlice = append(tempEdgeSlice, comps_p[i].Edges.Slice()...)
					tempSpecialSlice = append(tempSpecialSlice, comps_p[i].Special...)
				}
			}

			// if no comps_p, other than comp_low, just use parent as is
			if len(comps_p) == 1 {
				comp_up.Edges = parentλ

				// adding new Special Edge to connect Child to comp_up
				specialChild = NewEdges(childλ.Slice())

				decompUp = Decomp{Graph: comp_up, Root: Node{Bag: Inter(parentλ.Vertices(), Vertices_H),
					Cover: parentλ, Children: []Node{Node{Bag: childχ, Cover: childλ}}}}

				if !Subset(Conn, parentλ.Vertices()) {

					log.Println("Comps of p", comps_p)

					log.Panicln("Conn not covered in parent, Wait, what?")
				}

			} else if len(tempEdgeSlice) > 0 { // otherwise compute decomp for comp_up

				comp_up.Edges = NewEdges(tempEdgeSlice)
				comp_up.Special = tempSpecialSlice

				log.Println("Upper component:", comp_up)

				//Reducing the allowed edges
				allowedReduced := allowedFull.Diff(comp_low.Edges)

				// adding new Special Edge to connect Child to comp_up
				specialChild = NewEdges(childλ.Slice())
				comp_up.Special = append(comp_up.Special, specialChild)

				decompUp = l.findHD(comp_up, Conn, allowedReduced)

				if reflect.DeepEqual(decompUp, Decomp{}) {

					// l.addNegative(childχ, comp_up, compSp_up) TODO think about how to cache this
					log.Println("Rejecting comp_up")

					continue PARENT
				}

			}

			// rearrange subtrees to form one that covers total of H
			rootChild := Node{Bag: childχ, Cover: childλ, Children: subtrees}

			var finalRoot Node
			if len(tempEdgeSlice) > 0 {
				finalRoot = attachingSubtrees(decompUp.Root, rootChild, specialChild)
			} else {
				finalRoot = rootChild
			}

			log.Printf("Produced Decomp: %v\n", finalRoot)
			return Decomp{Graph: H, Root: finalRoot}

		}
		log.Println("ran out of parents,\n\n")
	}

	// exhausted search space
	return Decomp{}
}

// a parallel version of logK, using goroutines and channels to run various parts concurrently
func (l LogKDecomp) findHDParallel(H Graph, Conn []int, allowedFull Edges) Decomp {

	// log.Printf("\n\nCurrent SubGraph: %v\n", H)
	// log.Printf("Current Allowed Edges: %v\n", allowedFull)
	// log.Println("Conn: ", PrintVertices(Conn), "\n\n")

	// Base Case
	if l.baseCaseCheck(H.Edges.Len(), len(H.Special), allowedFull.Len()) {
		return l.baseCase(H, allowedFull.Len())
	}
	//all vertices within (H ∪ Sp)
	Vertices_H := append(H.Vertices())

	allowed := FilterVertices(allowedFull, Vertices_H)

	// Set up iterator for child

	genChild := SplitCombin(allowed.Len(), l.K, runtime.GOMAXPROCS(-1), false)

	parallelSearch := Search{H: &H, Edges: &allowed, BalFactor: l.BalFactor, Generators: genChild}

	pred := BalancedCheck{}

	parallelSearch.FindNext(pred) // initial Search

	// checks all possibles nodes in H, together with PARENT loops, it covers all parent-child pairings
CHILD:
	for ; !parallelSearch.ExhaustedSearch; parallelSearch.FindNext(pred) {

		childλ := GetSubset(allowed, parallelSearch.Result)

		comps_c, _, _ := H.GetComponents(childλ)

		// log.Println("Balanced Child found, ", childλ)

		// Check if child is possible root
		if Subset(Conn, childλ.Vertices()) {

			// log.Printf("Child-Root cover chosen: %v\n", Graph{Edges: childλ})
			// log.Printf("Comps of Child-Root: %v\n", comps_c)

			childχ := Inter(childλ.Vertices(), Vertices_H)

			// check chache for previous encounters
			if l.cache.CheckNegative(childλ, comps_c) {
				// log.Println("Skipping a child sep", childχ)
				continue CHILD
			}

			// l.cacheMux.RLock()
			// _, ok := l.cache[IntHash(childχ)]
			// l.cacheMux.RUnlock()
			// if !ok {
			// 	var newCache CompCache
			// 	l.cacheMux.Lock()
			// 	l.cache[IntHash(childχ)] = &newCache
			// 	l.cacheMux.Unlock()

			// } else {
			// 	for j := range comps_c {
			// 		if l.checkNegative(childχ, comps_c[j], Sp) { //TODO: Add positive check and cutNodes
			// 			// log.Println("Skipping a child sep", childχ)
			// 			continue CHILD
			// 		}

			// 	}
			// }

			var subtrees []Node
			for y := range comps_c {
				V_comp_c := comps_c[y].Vertices()
				Conn_y := Inter(V_comp_c, childχ)

				decomp := l.findHDParallel(comps_c[y], Conn_y, allowedFull)
				if reflect.DeepEqual(decomp, Decomp{}) {
					// log.Println("Rejecting child-root")
					// log.Printf("\nCurrent SubGraph: %v\n", H)
					// log.Printf("Current Allowed Edges: %v\n", allowed)
					// log.Println("Conn: ", PrintVertices(Conn), "\n\n")
					l.cache.AddNegative(childλ, comps_c[y])
					continue CHILD
				}

				// log.Printf("Produced Decomp w Child-Root: %+v\n", decomp)
				subtrees = append(subtrees, decomp.Root)
			}

			root := Node{Bag: childχ, Cover: childλ, Children: subtrees}
			return Decomp{Graph: H, Root: root}
		}

		allowedParent := FilterVertices(allowed, append(Conn, childλ.Vertices()...))
		genParent := SplitCombin(allowedParent.Len(), l.K, runtime.GOMAXPROCS(-1), false)

		parentalSearch := Search{H: &H, Edges: &allowedParent, BalFactor: l.BalFactor, Generators: genParent}

		predPar := ParentCheck{Conn: Conn, Child: childλ.Vertices()}

		parentalSearch.FindNext(predPar)

	PARENT:
		for ; !parentalSearch.ExhaustedSearch; parentalSearch.FindNext(predPar) {

			parentλ := GetSubset(allowedParent, parentalSearch.Result)

			// log.Println("Looking at parent ", parentλ)

			comps_p, _, isolatedEdges := H.GetComponents(parentλ)

			foundLow := false
			var comp_low_index int
			var comp_low Graph

			// Check if parent is un-balanced
			for i := range comps_p {
				if comps_p[i].Len() > H.Len()/2 {
					foundLow = true
					comp_low_index = i //keep track of the index for composing comp_up later
					comp_low = comps_p[i]
				}
			}
			if !foundLow {
				continue PARENT
			}

			vertCompLow := comp_low.Vertices()
			childχ := Inter(childλ.Vertices(), vertCompLow)

			// determine which componenents of child are inside comp_low

			//CHECK IF THIS ACTUALLY MAKES A DIFFERENCE
			comps_c, _, _ := comp_low.GetComponents(childλ)

			//omitting the check for balancedness as it's guaranteed to still be conserved at this point

			// check chache for previous encounters
			if l.cache.CheckNegative(childλ, comps_c) {
				// log.Println("Skipping a child sep", childχ)
				continue PARENT
			}

			// l.cacheMux.RLock()
			// _, ok := l.cache[IntHash(childχ)]
			// l.cacheMux.RUnlock()
			// if !ok {
			// 	var newCache CompCache
			// 	l.cacheMux.Lock()
			// 	l.cache[IntHash(childχ)] = &newCache
			// 	l.cacheMux.Unlock()

			// } else {
			// 	for j := range comps_c {
			// 		if l.checkNegative(childχ, comps_c[j], Sp) { //TODO: Add positive check and cutNodes
			// 			// log.Println("Skipping a child sep", childχ)
			// 			continue PARENT
			// 		}
			// 	}
			// }

			// log.Printf("Parent Found: %v (%s) \n", Graph{Edges: parentλ}, PrintVertices(parentλ.Vertices()))

			// log.Printf("Child chosen: %v (%s) \n", Graph{Edges: childλ}, PrintVertices(childχ))
			// log.Printf("Comps of Child: %v\n", comps_c)

			//Computing subcomponents of Child

			var subtrees []Node
			for x := range comps_c {
				Conn_x := Inter(comps_c[x].Vertices(), childχ)

				decomp := l.findHDParallel(comps_c[x], Conn_x, allowedFull)
				if reflect.DeepEqual(decomp, Decomp{}) {
					l.cache.AddNegative(childλ, comps_c[x])
					// log.Println("Rejecting child")
					continue PARENT
				}

				// log.Printf("Produced Decomp: %+v\n", decomp)
				subtrees = append(subtrees, decomp.Root)

			}

			//Computing upper component

			var comp_up Graph
			var decompUp Decomp
			var specialChild Edges
			tempEdgeSlice := []Edge{}
			tempSpecialSlice := []Edges{}

			tempEdgeSlice = append(tempEdgeSlice, isolatedEdges...)
			for i := range comps_p {
				if i != comp_low_index {
					tempEdgeSlice = append(tempEdgeSlice, comps_p[i].Edges.Slice()...)
					tempSpecialSlice = append(tempSpecialSlice, comps_p[i].Special...)
				}
			}

			// if no comps_p, other than comp_low, just use parent as is
			if len(comps_p) == 1 {
				comp_up.Edges = parentλ

				// adding new Special Edge to connect Child to comp_up
				specialChild = NewEdges(childλ.Slice())

				decompUp = Decomp{Graph: comp_up, Root: Node{Bag: Inter(parentλ.Vertices(), Vertices_H),
					Cover: parentλ, Children: []Node{Node{Bag: childχ, Cover: childλ}}}}

				if !Subset(Conn, parentλ.Vertices()) {

					fmt.Println("Current SubGraph, ", H)

					fmt.Println("Child ", childχ)

					fmt.Println("Comps of child ", comps_c)

					fmt.Println("parent ", parentλ)
					fmt.Println("Conn ", Conn)
					fmt.Println("Comps of p", comps_p)

					log.Panicln("Conn not covered in parent, Wait, what?")
				}

			} else if len(tempEdgeSlice) > 0 { // otherwise compute decomp for comp_up

				comp_up.Edges = NewEdges(tempEdgeSlice)
				comp_up.Special = tempSpecialSlice

				// log.Println("Upper component:", comp_up)

				//Reducing the allowed edges
				allowedReduced := allowedFull.Diff(comp_low.Edges)

				// adding new Special Edge to connect Child to comp_up
				specialChild = NewEdges(childλ.Slice())
				comp_up.Special = append(comp_up.Special, specialChild)

				decompUp = l.findHDParallel(comp_up, Conn, allowedReduced)

				if reflect.DeepEqual(decompUp, Decomp{}) {

					// l.addNegative(childχ, comp_up, Sp)
					// log.Println("Rejecting comp_up")

					continue PARENT
				}

			}

			// rearrange subtrees to form one that covers total of H
			rootChild := Node{Bag: childχ, Cover: childλ, Children: subtrees}

			var finalRoot Node
			if len(tempEdgeSlice) > 0 {
				finalRoot = attachingSubtrees(decompUp.Root, rootChild, specialChild)
			} else {
				finalRoot = rootChild
			}

			// log.Printf("Produced Decomp: %v\n", finalRoot)
			return Decomp{Graph: H, Root: finalRoot}

		}
		// log.Println("ran out of parents", parentalSearch.ExhaustedSearch, "\n\n")
	}

	// exhausted search space
	return Decomp{}

}
