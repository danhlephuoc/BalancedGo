// Follows the algorithm k divide decomp from Samer "Exploiting Parallelism in Decomposition Methods for Constraint Satisfaction"
package algorithms

import (
	"bytes"
	"fmt"
	"log"
	"reflect"

	. "github.com/cem-okulmus/BalancedGo/lib"
	"github.com/spakin/disjoint"
)

type divideComp struct {
	up           []int
	edges        Edges
	low          []int
	upConnecting bool
}

func (c divideComp) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("Up: ")
	buffer.WriteString(Edge{Vertices: c.up}.String())

	buffer.WriteString(" Edges: ")
	buffer.WriteString(Graph{Edges: c.edges}.String())

	buffer.WriteString(" Low: ")
	buffer.WriteString(Edge{Vertices: c.low}.String())

	buffer.WriteString(" upConnecting: ")
	buffer.WriteString(fmt.Sprintln(c.upConnecting))

	return buffer.String()

}

//similar to getComponents on Graphs, with minor changes to account for divideKDecomp
func (comp divideComp) getComponents(sep Edges) ([]divideComp, map[int]*disjoint.Element, int) {
	//edgesUp := FilterVertices(comp.edges, comp.up)   // all edges in comp connected to Up
	edgesLow := FilterVertices(comp.edges, comp.low) // all edges in comp conneted to low

	var output []divideComp

	var vertices = make(map[int]*disjoint.Element)
	var comps = make(map[*disjoint.Element][]Edge)

	sepVert := sep.Vertices()
	sepCache := make(map[int]bool, len(sepVert))
	//fmt.Println("Current separator ", Edge{Vertices: sepVert})
	for _, v := range sepVert {
		sepCache[v] = true
	}

	//  Set up the disjoint sets for each node
	for _, i := range comp.edges.Vertices() {
		vertices[i] = disjoint.NewElement()
	}

	// Merge together the connected components
	for _, e := range comp.edges.Slice() {
		for i := 0; i < len(e.Vertices); i++ {
			if sepCache[e.Vertices[i]] {
				continue
			}
			for j := i + 1; j < len(e.Vertices); j++ {
				if sepCache[e.Vertices[j]] {
					continue
				}
				//			fmt.Println("Union of ", m[e.Vertices[i]], "and ", m[e.Vertices[j]])
				disjoint.Union(vertices[e.Vertices[i]], vertices[e.Vertices[j]])
				// j = i-1
				break
			}
		}
	}

	//sort each edge to a corresponding component
	for _, e := range comp.edges.Slice() {
		var vertexRep int
		found := false
		for _, v := range e.Vertices {
			if sepCache[v] {
				continue
			}
			vertexRep = v
			found = true
			break
		}
		if !found {
			continue
		}

		slice, ok := comps[vertices[vertexRep].Find()]
		if !ok {
			newslice := make([]Edge, 0, comp.edges.Len())
			comps[vertices[vertexRep].Find()] = newslice
			slice = newslice
		}

		comps[vertices[vertexRep].Find()] = append(slice, e)
	}

	upConn := 0

	// Perf: You _can_ check comp for up/low connecting without an intersect operation
	// Store the components
	for i, _ := range comps {
		c := divideComp{edges: NewEdges(comps[i])}
		if len(Inter(comp.up, c.edges.Vertices())) > 0 { // comp Up connecting
			//c.edges.Append(DiffEdges(edgesUp, c.edges.Slice()...).Slice()...) // make sure all upEdges stay together
			c.upConnecting = true
			upConn++
			if upConn > 1 {
				return output, vertices, upConn
			}
			c.up = Inter(comp.up, c.edges.Vertices())
			//c.up = comp.up
			c.low = Inter(sepVert, c.edges.Vertices())
		} else {
			if len(Inter(comp.low, c.edges.Vertices())) > 0 { // comp Low connecting
				//	log.Println("Low connecting comp.", c.edges, " lowEdges", edgesLow, "\nAdding Edges ", DiffEdges(edgesLow, c.edges.Slice()...))
				c.edges.Append(DiffEdges(edgesLow, c.edges.Slice()...).Slice()...) // make sure all lowEdges stay together
				c.low = comp.low
			}
			c.up = Inter(sepVert, c.edges.Vertices())
		}

		output = append(output, c)
	}

	if upConn == 0 && !Subset(comp.up, sepVert) {
		fmt.Println("H: ")
		for _, e := range comp.edges.Slice() {
			fmt.Println(e.FullString())
		}

		fmt.Println("Sep, ", sep, "Vertices: ", PrintVertices(sepVert))
		fmt.Println("compUp", PrintVertices(comp.up))
		i := 0
		for _, c := range comps {
			cEdges := NewEdges(c)
			fmt.Println("Component ", cEdges, " vertices: ", PrintVertices(cEdges.Vertices()))
			i++
		}

		log.Panicln("something is rotten in the state of this program")
	}

	return output, vertices, upConn
}

type DivideKDecomp struct {
	Graph     Graph
	K         int
	BalFactor int
}

func (d DivideKDecomp) CheckBalancedSep(comp divideComp, sep Edges) bool {

	comps, vertices, upConn := comp.getComponents(sep)

	if upConn > 1 {
		log.Println("More than one upconnecting, not in normal form")
		return false
	}

	//check if up and low separated
	// constant check enough as all vertices in up (resp. low) part of the same comp
	if len(comp.up) > 0 && len(comp.low) > 0 {
		if vertices[comp.up[0]] == vertices[comp.low[0]] {
			log.Println("Up and low not separated")
			log.Println("Current: ", comp, "\n\n")

			for i := range comps {
				log.Println(comps[i], "\n")
			}

			return false
		}
	}

	//ensure that connection to upper node is still intact
	for i := range comps {
		if comps[i].upConnecting && !reflect.DeepEqual(comp.up, comps[i].up) {
			return false
		}
	}

	// TODO, make this work only a single loop
	if len(comp.low) == 0 {
		// log.Printf("Components of sep %+v\n", comps)
		for i := range comps {

			if comps[i].edges.Len() > (((comp.edges.Len()) * (d.BalFactor - 1)) / d.BalFactor) {
				//log.Printf("Using %+v component %+v has weight %d instead of %d\n", sep, comps[i], comps[i].edges.Len(), (((comp.edges.Len())*(d.BalFactor-1))/d.BalFactor)+d.K)
				return false
			}
		}
	} else {
		for i := range comps {
			if len(comps[i].low) == 0 {
				if comps[i].edges.Len() == comp.edges.Len() { // must make some progres
					return false
				}

				continue
			}
			if comps[i].edges.Len() > (((comp.edges.Len()) * (d.BalFactor - 1)) / d.BalFactor) {
				//log.Printf("Using %+v component %+v has weight %d instead of %d\n", sep, comps[i], comps[i].edges.Len(), (((comp.edges.Len())*(d.BalFactor-1))/d.BalFactor)+d.K)
				return false
			}
		}
		if len(Inter(sep.Vertices(), comp.edges.Vertices())) == 0 { //make some progress
			return false
		}
	}

	return true
}

//TODO check if this kind of manipulation actually works outside of current scope
func reorderComps(parent Node, subtree Node, up []int) Node {
	log.Println("Two Nodes enter: ", parent, subtree)
	up = Inter(up, parent.Vertices())
	//finding connecting leaf in parent
	found, leaf := parent.CheckLeaves(up, subtree)
	if !found {
		fmt.Println("\n \n comp ", PrintVertices(up))
		fmt.Println("parent ", parent)

		log.Panicln("parent tree doesn't contain connecting node!")
	}

	// //attaching subtree to parent
	// leaf.Children = []Node{subtree}
	log.Println("Leaf ", leaf)
	log.Println("One Node leaves: ", parent)
	return parent
}

func (d DivideKDecomp) decomposable(comp divideComp) Decomp {
	log.Printf("\n\nCurrent SubGraph: %v\n", comp)

	//base case: size of comp <= K
	if comp.edges.Len() <= d.K {
		sep := NewEdges(comp.edges.Slice())
		return Decomp{Graph: d.Graph,
			Root: Node{Cover: sep, Bag: sep.Vertices()}}
	}
	edges := FilterVertices(d.Graph.Edges, comp.edges.Vertices())

	gen := GetCombin(edges.Len(), d.K)

OUTER:
	for gen.HasNext() {
		gen.Confirm()
		balsep := GetSubset(edges, gen.Combination)
		if !d.CheckBalancedSep(comp, balsep) {
			continue
		}
		log.Println("Chosen Sep ", balsep)

		comps, _, upConn := comp.getComponents(balsep)

		log.Printf("Comps of Sep: %v\n", comps)

		var parent Node
		var subtrees []Node
		for i, _ := range comps {
			child := d.decomposable(comps[i])
			if reflect.DeepEqual(child, Decomp{}) {
				log.Printf("REJECTING %v: couldn't decompose %v \n", Graph{Edges: balsep}, comps[i])
				log.Printf("\n\nCurrent SubGraph: %v\n", comp)
				continue OUTER
			}

			log.Printf("Produced Decomp: %v\n", child)

			if upConn == 1 && comps[i].upConnecting {
				parent = child.Root
			} else {
				subtrees = append(subtrees, child.Root)
			}
		}

		var output Node

		SubtreeRootedAtS := Node{Cover: balsep, Bag: balsep.Vertices(), Children: subtrees}

		if reflect.DeepEqual(parent, Node{}) && (!Subset(comp.up, balsep.Vertices())) {

			fmt.Println("Subtrees: ")
			for _, s := range subtrees {
				fmt.Println("\n\n", s)
			}

			log.Panicln("Parent missing")
		}

		if upConn == 1 {
			output = reorderComps(parent, SubtreeRootedAtS, balsep.Vertices())

			log.Printf("Reordered Decomp: %v\n", output)
		} else {
			output = SubtreeRootedAtS
		}

		return Decomp{Graph: d.Graph, Root: output}
	}

	return Decomp{} // using empty decomp as reject case

}

func (d DivideKDecomp) FindDecomp() Decomp {
	return d.decomposable(divideComp{edges: d.Graph.Edges})
}
