package lib

import (
	"bytes"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/spakin/disjoint"
)

// A Graph is a collection of (special) edges
type Graph struct {
	Edges    Edges
	Special  []Edges
	vertices []int
}

func (g Graph) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("{")
	for i, e := range g.Edges.Slice() {
		buffer.WriteString(e.String())
		if i != g.Edges.Len()-1 {
			buffer.WriteString(", ")
		}
	}
	buffer.WriteString("}")

	if len(g.Special) > 0 {
		buffer.WriteString(" & Special Edges [")
		for i := range g.Special {
			buffer.WriteString(g.Special[i].String())
			if i != len(g.Special)-1 {
				buffer.WriteString(", ")
			}
		}
		buffer.WriteString(" ]")
	}

	return buffer.String()
}

func (g Graph) equal(other Graph) bool {
	return cmp.Equal(g, other, cmpopts.IgnoreUnexported(g), cmp.Comparer(equalEdges))
}

// Vertices produces the union of all vertices from all edges of the graph
func (g *Graph) Vertices() []int {
	if len(g.vertices) > 0 {
		return g.vertices
	}
	var output []int
	for _, otherE := range g.Edges.Slice() {
		output = append(output, otherE.Vertices...)
	}
	for i := range g.Special {
		output = append(output, g.Special[i].Vertices()...)
	}
	g.vertices = RemoveDuplicates(output)
	return g.vertices
}

// Len returns the number of edges and special edges of the graph
func (g Graph) Len() int {
	return g.Edges.Len() + len(g.Special)
}

// GetSubset produces a selection of edges from slice of integers s
// used as indices. This is used to select new potential separators.
// Note that special edges are ignored here, since they should never be
// considered when choosing a separator
func GetSubset(edges Edges, s []int) Edges {
	var output []Edge
	for _, i := range s {
		output = append(output, edges.Slice()[i])
	}
	return NewEdges(output)
}

// GetSubset is as above, but the first parameter is omitted when used as the method call of a graph
func (g Graph) GetSubset(s []int) Edges {
	return GetSubset(g.Edges, s)
}

// GetComponents uses Disjoint Set data structure to compute connected components
func (g Graph) GetComponents(sep Edges) ([]Graph, map[int]int, []Edge) {
	var outputG []Graph

	var vertices = make(map[int]*disjoint.Element)
	var comps = make(map[*disjoint.Element][]Edge)
	var compsSp = make(map[*disjoint.Element][]Edges)

	balsepVert := sep.Vertices()
	balSepCache := make(map[int]bool, len(balsepVert))
	for _, v := range balsepVert {
		balSepCache[v] = true
	}

	//  Set up the disjoint sets for each node
	for _, i := range g.Vertices() {
		vertices[i] = disjoint.NewElement()
	}

	// Merge together the connected components
	for k := range g.Edges.Slice() {
		for i := 0; i < len(g.Edges.Slice()[k].Vertices); i++ {
			if balSepCache[g.Edges.Slice()[k].Vertices[i]] {

				continue
			}
			for j := i + 1; j < len(g.Edges.Slice()[k].Vertices); j++ {
				if balSepCache[g.Edges.Slice()[k].Vertices[j]] {
					continue
				}
				// fmt.Println("Union of ", m[g.Edges.Slice()[k].Vertices[i]], "and ",
				//      m[g.Edges.Slice()[k].Vertices[j]])
				disjoint.Union(vertices[g.Edges.Slice()[k].Vertices[i]], vertices[g.Edges.Slice()[k].Vertices[j]])
				// j = i-1
				break
			}
		}
	}

	for k := range g.Special {
		for i := 0; i < len(g.Special[k].Vertices())-1; i++ {
			if balSepCache[g.Special[k].Vertices()[i]] {
				continue
			}
			for j := i + 1; j < len(g.Special[k].Vertices()); j++ {
				if balSepCache[g.Special[k].Vertices()[j]] {
					continue
				}
				disjoint.Union(vertices[g.Special[k].Vertices()[i]], vertices[g.Special[k].Vertices()[j]])
				// j = i-1
				break
			}
		}

	}

	var isolatedEdges []Edge

	//sort each edge and special edge to a corresponding component
	for i := range g.Edges.Slice() {
		var vertexRep int
		found := false
		for _, v := range g.Edges.Slice()[i].Vertices {
			if balSepCache[v] {
				continue
			}
			vertexRep = v
			found = true
			break
		}
		if !found {
			isolatedEdges = append(isolatedEdges, g.Edges.Slice()[i])
			continue
		}

		slice, ok := comps[vertices[vertexRep].Find()]
		if !ok {
			newslice := make([]Edge, 0, g.Edges.Len())
			comps[vertices[vertexRep].Find()] = newslice
			slice = newslice
		}

		comps[vertices[vertexRep].Find()] = append(slice, g.Edges.Slice()[i])

	}

	var isolatedSp []Edges
	for i := range g.Special {
		var vertexRep int
		found := false
		for _, v := range g.Special[i].Vertices() {
			if balSepCache[v] {
				continue
			}
			vertexRep = v
			found = true
			break
		}
		if !found {
			isolatedSp = append(isolatedSp, g.Special[i])
			continue
		}

		slice, ok := compsSp[vertices[vertexRep].Find()]
		if !ok {
			newslice := make([]Edges, 0, len(g.Special))
			compsSp[vertices[vertexRep].Find()] = newslice
			slice = newslice
		}

		compsSp[vertices[vertexRep].Find()] = append(slice, g.Special[i])
	}

	edgeToComp := make(map[int]int)

	// Store the components as graphs
	for k := range comps {
		slice := comps[k]
		for i := range slice {
			edgeToComp[slice[i].Name] = len(outputG)
		}
		g := Graph{Edges: NewEdges(slice), Special: compsSp[k]}
		outputG = append(outputG, g)
	}

	for k := range compsSp {
		_, ok := comps[k]
		if ok {
			continue
		}
		g := Graph{Edges: NewEdges([]Edge{}), Special: compsSp[k]}
		outputG = append(outputG, g)
	}

	for i := range isolatedSp {
		g := Graph{Edges: NewEdges([]Edge{}), Special: []Edges{isolatedSp[i]}}
		outputG = append(outputG, g)
	}

	return outputG, edgeToComp, isolatedEdges
}

// FilterVertices filters an Edges slice for a given set of vertices.
// Edges are only removed, if they have an empty intersection with the vertex set.
func FilterVertices(edges Edges, vertices []int) Edges {
	var output []Edge

	for _, e := range edges.Slice() {
		if len(Inter(e.Vertices, vertices)) > 0 {
			output = append(output, e)
		}
	}

	return NewEdges(output)
}

// FilterVerticesStrict filters an Edges slice for a given set of vertices.
// Edges are removed if they are not full subsets of the vertex set
func FilterVerticesStrict(edges Edges, vertices []int) Edges {
	var output []Edge

	for _, e := range edges.Slice() {
		if Subset(e.Vertices, vertices) {
			output = append(output, e)
		}
	}

	return NewEdges(output)
}

// CutEdges filters an Edges slice for a given set of vertices.
// Edges are transformed to their intersection against the vertex set,
// producing the induced subgraph
func CutEdges(edges Edges, vertices []int) Edges {
	var output []Edge

	for i := range edges.Slice() {
		inter := Inter(edges.Slice()[i].Vertices, vertices)
		if len(inter) > 0 {
			name := edges.Slice()[i].Name
			// if len(inter) < len(edges.Slice()[i].Vertices) {

			//  var mux sync.Mutex
			//  mux.Lock() // ensure that hash is computed only on one goroutine at a time
			//  name = encode
			//  m[encode] = m[edges.Slice()[i].Name] + "'"
			//  encode++
			//  mux.Unlock()

			// }
			output = append(output, Edge{Name: name, Vertices: inter})
		}
	}

	return NewEdges(output)
}

// ComputeSubEdges computes all relevant subedges to produce a GHD of width K
func (g Graph) ComputeSubEdges(K int) Graph {
	var output = g.Edges.Slice()

	for _, e := range g.Edges.Slice() {
		edgesWihoutE := diffEdges(g.Edges, e)
		gen := getCombin(edgesWihoutE.Len(), K)
		for gen.hasNext() {
			subset := GetSubset(edgesWihoutE, gen.combination)
			var tuple = subset.Vertices()
			output = append(output, Edge{Vertices: Inter(e.Vertices, tuple)}.subedges()...)
			gen.confirm()
		}
	}

	return Graph{Edges: removeDuplicateEdges(output)}
}

// GetBIP computes the BIP number of the graph
func (g Graph) GetBIP() int {
	var output int

	edges := g.Edges.Slice()

	for i := range edges {
		for j := range edges {
			if j <= i {
				continue
			}
			tmp := len(Inter(edges[i].Vertices, edges[j].Vertices))
			if tmp > output {
				output = tmp
			}
		}
	}

	return output
}
