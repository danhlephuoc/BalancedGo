package lib

import (
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/participle/lexer/ebnf"
)

var m map[int]string // stores the encoding of vertices for last file parsed (bit of a hack)
var mutex = sync.RWMutex{}
var encode int // stores the encoding of the highest int used

type ParseEdge struct {
	Name     string   ` @(Ident|Number)`
	Vertices []string `"(" ( @(Ident|Number)  ","? )* ")"`
}

type ParseGraph struct {
	Edges []ParseEdge `( @@ ","?)*`
	m     map[string]int
}

// Implement PACE 2019 format

type ParseEdgePACE struct {
	Name     int   ` @Number`
	Vertices []int ` ( @Number   )* "\n" `
}

type ParseGraphPACEInfo struct {
	Vertices int `"p htd":Begin @(Number) `
	Edges    int `@(Number) "\n"`
}

type ParseGraphPACE struct {
	Info  ParseGraphPACEInfo `@@`
	Edges []ParseEdgePACE    `(@@) *`
	m     map[int]int
}

// Updated PACE 2019 format, with initial Special Edges

type ParseEdgeUpdate struct {
	Name     string   ` @(Ident|Number)`
	Vertices []string `"(" ( @(Ident|Number)  ","? )* ")"`
}

type ParseSpecialEdgeUpdate struct {
	Name     string   ` @(Ident|Number)`
	Vertices []string `"(" ( @(Ident|Number)  ","? )* ")"`
}

type ParseGhostEdgeUpdate struct {
	Name     string   ` @(Ident|Number)`
	Vertices []string `"(" ( @(Ident|Number)  ","? )* ")"`
}

type ParseGraphUpdate struct {
	Edges   []ParseEdgeUpdate        `( @@ ","?)* "."`
	Ghost   []ParseSpecialEdgeUpdate `("👻" ( @@ ","?)*)?`
	Special []ParseGhostEdgeUpdate   `"✨" ( @@ ","?)*`
	m       map[string]int
}

// type ParseGraphUpdateInfo struct {
// 	Vertices int `"p htd":Begin @(Number) `
// 	Edges    int `@(Number) "\n"`
// }

// type ParseGraphUpdate struct {
// 	Info         ParseGraphUpdateInfo     `@@`
// 	Edges        []ParseEdgeUpdate        `(@@) *`
// 	SpecialEdges []ParseSpecialEdgeUpdate `(@@) *`
// 	GhostEdges   []ParseGhostEdgeUpdate   `(@@) *`
// 	m            map[int]int
// }

func GetGraph(s string) (Graph, ParseGraph) {

	graphLexer := lexer.Must(ebnf.New(`
    Comment = ("%" | "//") { "\u0000"…"\uffff"-"\n" } .
    Ident = (alpha | "_") { "_" | alpha | digit | stuff } .
    Number = ("." | digit | "_"){"." | digit | stuff } .
    Whitespace = " " | "\t" | "\n" | "\r" .
    stuff = ":" | "@" | ";" | "-" | "_" .
    Punct = "!"…"/"  .
    alpha = "a"…"z" | "A"…"Z" .
    digit = "0"…"9" .`))

	var parser = participle.MustBuild(&ParseGraph{}, participle.UseLookahead(1), participle.Lexer(graphLexer),
		participle.Elide("Comment", "Whitespace"))
	var output Graph
	var edges []Edge
	pgraph := ParseGraph{}
	parser.ParseString(s, &pgraph)
	// if err != nil {
	// 	fmt.Println("Couldn't parse input: ")
	// 	panic(err)
	// }
	encoding := make(map[int]string)
	encode = 1 // initialize to 1

	pgraph.m = make(map[string]int)
	//fix first numbers for edge names
	for _, e := range pgraph.Edges {
		_, ok := pgraph.m[e.Name]
		if ok {
			log.Panicln("Edge names not unique, not a vald hypergraph!")
		}

		pgraph.m[e.Name] = encode
		encoding[encode] = e.Name
		encode++
	}
	for _, e := range pgraph.Edges {
		var outputEdges []int
		for _, n := range e.Vertices {
			i, ok := pgraph.m[n]
			if ok {
				outputEdges = append(outputEdges, i)
			} else {
				pgraph.m[n] = encode
				encoding[encode] = n
				outputEdges = append(outputEdges, encode)
				encode++
			}
		}
		edges = append(edges, Edge{Name: pgraph.m[e.Name], Vertices: outputEdges})
	}
	output.Edges = NewEdges(edges)
	m = encoding
	return output, pgraph
}

func (p *ParseGraph) GetEdge(input string) Edge {

	graphLexer := lexer.Must(ebnf.New(`
    Comment = ("%" | "//") { "\u0000"…"\uffff"-"\n" } .
    Ident = (alpha | "_") { "_" | alpha | digit | stuff } .
    Number = ("." | digit | "_"){"." | digit | stuff } .
    Whitespace = " " | "\t" | "\n" | "\r" .
    stuff = ":" | "@" | ";" | "-" | "_" .
    Punct = "!"…"/"  .
    alpha = "a"…"z" | "A"…"Z" .
    digit = "0"…"9" .`))

	var parser = participle.MustBuild(&ParseEdge{}, participle.UseLookahead(1), participle.Lexer(graphLexer),
		participle.Elide("Comment", "Whitespace"))
	pEdge := ParseEdge{}
	parser.ParseString(input, &pEdge)
	var vertices []int
	for _, v := range pEdge.Vertices {
		val, ok := p.m[v]
		if ok {
			vertices = append(vertices, val)
		} else {
			p.m[v] = encode
			m[encode] = v
			vertices = append(vertices, encode)
			encode++
		}
	}
	m[encode] = pEdge.Name
	encode++
	return Edge{Vertices: vertices, Name: encode - 1}
}

func GetGraphPACE(s string) Graph {

	graphLexer := lexer.Must(ebnf.New(`
    Comment = ("c" | "//") { "\u0000"…"\uffff"-"\n" } Newline.
    Begin = "p htd" .
    Number = ("." | digit | "_"){"." | digit | stuff } .
    Whitespace = " " | "\t" | "\n" | "\r" .
    stuff = ":" | "@" | ";" | "-" | "_" .
    Punct = "!"…"/"  .
    Newline = "\n" .

    digit = "0"…"9" .`))

	var parser = participle.MustBuild(&ParseGraphPACE{}, participle.UseLookahead(1), participle.Lexer(graphLexer),
		participle.Elide("Comment", "Whitespace"))
	var output Graph
	var edges []Edge
	pgraph := ParseGraphPACE{}
	err := parser.ParseString(s, &pgraph)
	if err != nil {
		fmt.Println("Couldn't parse input: ")
		panic(err)
	}
	encode = 1 // initialize to 1

	encoding := make(map[int]string)
	pgraph.m = make(map[int]int)

	for _, e := range pgraph.Edges {
		encoding[encode] = "E" + strconv.Itoa(e.Name)
		pgraph.m[e.Name] = encode
		encode++
	}

	for _, e := range pgraph.Edges {
		var outputEdges []int
		for _, n := range e.Vertices {
			i, ok := pgraph.m[n+pgraph.Info.Edges]
			if ok {
				outputEdges = append(outputEdges, i)
			} else {
				pgraph.m[n+pgraph.Info.Edges] = encode
				encoding[encode] = "V" + strconv.Itoa(n)
				outputEdges = append(outputEdges, encode)
				encode++

			}
		}
		edges = append(edges, Edge{Name: pgraph.m[e.Name], Vertices: outputEdges})
	}

	m = encoding

	output.Edges = NewEdges(edges)

	// log.Println("Edges", pgraph.Info.Edges)
	// log.Println("Vertices", pgraph.Info.Vertices)

	// for _, e := range output.Edges.Slice() {
	// 	log.Println(e.FullString())
	// }

	// log.Panicln("")
	return output
}

func GetGraphUpdate(s string) (Graph, Graph, []Special) {

	graphLexer := lexer.Must(ebnf.New(`
    Comment = ("%" | "//") { "\u0000"…"\uffff"-"\n" } .
    Ident = (alpha | "_") { "_" | alpha | digit | stuff } .
    Number = ("." | digit | "_"){"." | digit | "_"} .
    Whitespace = " " | "\t" | "\n" | "\r" .
    stuff = ":" | "@" | ";" | "-" .
    Punct = "!"…"/"  .
    alpha = "a"…"z" | "A"…"Z" .
    SpecialSep = "✨" .
    GhostSep  = "👻" .
    digit = "0"…"9" .`))

	var parser = participle.MustBuild(&ParseGraphUpdate{}, participle.UseLookahead(1), participle.Lexer(graphLexer),
		participle.Elide("Comment", "Whitespace"))
	var output Graph
	var ghostGraph Graph
	var edges []Edge
	var ghostEdges []Edge
	var special []Special

	pgraph := ParseGraphUpdate{}
	err := parser.ParseString(s, &pgraph)
	if err != nil {
		fmt.Println("Couldn't parse input: ")
		panic(err)
	}
	encode = 1 // initialize to 1

	encoding := make(map[int]string)
	pgraph.m = make(map[string]int)

	for _, e := range pgraph.Edges {
		_, ok := pgraph.m[e.Name]
		if ok {
			log.Panicln("Edge names not unique, not a vald hypergraph!")
		}
		pgraph.m[e.Name] = encode
		encoding[encode] = e.Name
		encode++
	}

	for _, e := range pgraph.Ghost {
		encoding[encode] = "👻" + e.Name
		pgraph.m["👻"+e.Name] = encode
		encode++
	}

	for _, e := range pgraph.Special {
		encoding[encode] = e.Name + " ✨"
		pgraph.m[e.Name+" ✨"] = encode
		encode++
	}

	for _, e := range pgraph.Edges {
		var outputEdge []int
		for _, n := range e.Vertices {
			i, ok := pgraph.m[n]
			if ok {
				outputEdge = append(outputEdge, i)
			} else {
				pgraph.m[n] = encode
				encoding[encode] = n
				outputEdge = append(outputEdge, encode)
				encode++
			}
		}
		edges = append(edges, Edge{Name: pgraph.m[e.Name], Vertices: outputEdge})
	}

	for _, e := range pgraph.Ghost {
		var outputEdge []int
		for _, n := range e.Vertices {
			i, ok := pgraph.m[n]
			if ok {
				outputEdge = append(outputEdge, i)
			} else {
				pgraph.m[n] = encode
				encoding[encode] = n
				outputEdge = append(outputEdge, encode)
				encode++

			}
		}
		ghostEdges = append(ghostEdges, Edge{Name: pgraph.m["👻"+e.Name], Vertices: outputEdge})
	}

	for _, s := range pgraph.Special {
		var outputSpecialEdge []int
		for _, n := range s.Vertices {
			i, ok := pgraph.m[n]
			if ok {
				outputSpecialEdge = append(outputSpecialEdge, i)
			} else {
				pgraph.m[n] = encode
				encoding[encode] = n
				outputSpecialEdge = append(outputSpecialEdge, encode)
				encode++
			}
		}
		dummyEdges := NewEdges([]Edge{Edge{Name: pgraph.m[s.Name+" ✨"], Vertices: outputSpecialEdge}})
		special = append(special, Special{Vertices: outputSpecialEdge, Edges: dummyEdges})
	}

	m = encoding

	output.Edges = NewEdges(edges)
	ghostGraph.Edges = NewEdges(append(edges, ghostEdges...))

	// log.Println("Edges", pgraph.Info.Edges)
	// log.Println("Vertices", pgraph.Info.Vertices)

	// for _, e := range output.Edges.Slice() {
	// 	log.Println(e.FullString())
	// }

	// for _, e := range special {
	// 	log.Println(e)
	// }

	return ghostGraph, output, special
}
