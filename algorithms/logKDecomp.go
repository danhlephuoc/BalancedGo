package algorithms

import (
	"fmt"
	"log"
	"math"
	"reflect"
	"sync"

	. "github.com/cem-okulmus/BalancedGo/lib"
)

// A log-depth version of DetK, without the restriction to strong NF HDs

type LogKDecomp struct {
	Graph    Graph
	SubEdge  bool
	Depth    int
	cache    map[uint32]*CompCache
	cacheMux sync.RWMutex
}

func (d *LogKDecomp) addPositive(sep Edges, comp Graph) {
	d.cacheMux.Lock()
	d.cache[sep.Hash()].Succ = append(d.cache[sep.Hash()].Succ, comp.Edges.Hash())
	d.cacheMux.Unlock()
}

func (d *LogKDecomp) addNegative(sep Edges, comp Graph) {
	d.cacheMux.Lock()
	d.cache[sep.Hash()].Fail = append(d.cache[sep.Hash()].Fail, comp.Edges.Hash())
	d.cacheMux.Unlock()
}

func (d *LogKDecomp) checkNegative(sep Edges, comp Graph) bool {
	d.cacheMux.RLock()
	defer d.cacheMux.RUnlock()

	compCachePrev, _ := d.cache[sep.Hash()]
	for i := range compCachePrev.Fail {
		if comp.Edges.Hash() == compCachePrev.Fail[i] {
			//  log.Println("Comp ", comp, "(hash ", comp.Edges.Hash(), ")  known as negative for sep ", sep)
			return true
		}

	}

	return false
}

func (d *LogKDecomp) checkPositive(sep Edges, comp Graph) bool {
	d.cacheMux.RLock()
	defer d.cacheMux.RUnlock()

	compCachePrev, _ := d.cache[sep.Hash()]
	for i := range compCachePrev.Fail {
		if comp.Edges.Hash() == compCachePrev.Succ[i] {
			//  log.Println("Comp ", comp, " known as negative for sep ", sep)
			return true
		}

	}

	return false
}

func (d *LogKDecomp) findDecomp(K int, H Graph, oldSep []int, Sp []Special, depth int) Decomp {
	if depth > d.Depth {
		return Decomp{} // stop computation once recursion depth reached
	}

	verticesCurrent := append(H.Vertices(), VerticesSpecial(Sp)...)
	verticesExtended := append(verticesCurrent, oldSep...)
	conn := Inter(oldSep, verticesCurrent)
	compVertices := Diff(verticesCurrent, oldSep)
	bound := FilterVertices(d.Graph.Edges, conn)

	// log.Printf("\n\nD Current oldSep: %v, Conn: %v\n", PrintVertices(oldSep), PrintVertices(conn))
	// log.Printf("D Current SubGraph: %v ( %v hash) \n", H, H.Edges.Hash())
	// log.Printf("D Current SubGraph: %v ( %v edges) (hash: %v )\n", H, H.Edges.Len(), H.Edges.Hash())
	// log.Printf("D Current Special Edges: %v\n\n", Sp)
	// log.Println("D Hedges ", H)
	// log.Println("D Comp Vertices: ", PrintVertices(compVertices))

	// Base case if H <= K
	if H.Edges.Len() == 0 && len(Sp) <= 1 {
		return baseCaseDetK(H, Sp)
	}

	gen := NewCover(K, conn, bound, H.Edges)

OUTER:
	for gen.HasNext {
		out := gen.NextSubset()

		if out == -1 {
			if gen.HasNext {
				log.Panicln(" -1 but hasNext not false!")
			}
			continue
		}

		var sep Edges
		sep = GetSubset(bound, gen.Subset)

		if !Subset(conn, sep.Vertices()) {
			log.Panicln("Cover messed up! 137")
		}

		// log.Println("Next Cover ", sep)

		addEdges := false

		//check if sep "makes some progress" into separating H

		if len(Inter(sep.Vertices(), compVertices)) == 0 {
			addEdges = true
		}

		if !addEdges || K-sep.Len() > 0 {
			// i_add := 0

			genCombin := GetCombin(H.Edges.Len(), K-sep.Len())

		addingEdges:
			for !addEdges || genCombin.HasNext() {
				var sepActual Edges

				sepAdded := GetSubset(H.Edges, genCombin.Combination)

				if addEdges {
					sepActual = NewEdges(append(sep.Slice(), sepAdded.Slice()...))
					genCombin.Confirm()
				} else {
					sepActual = sep
				}

				// sepActualOrigin := sepActual
				var sepSub *SepSub
				var sepConst []Edge
				var sepChanging []Edge
				if d.SubEdge {
					for i, v := range gen.Subset {
						if gen.InComp[v] {
							sepChanging = append(sepChanging, sep.Slice()[i])
						} else {
							sepConst = append(sepConst, sep.Slice()[i])
						}
					}
					if addEdges {
						sepChanging = append(sepChanging, sepAdded.Slice()...)
					}
				}

			subEdges:
				for true {

					// log.Println("Sep chosen ", sepActual, " out ", out)
					comps, compsSp, _ := H.GetComponents(sepActual, Sp)

					//check chache for previous encounters
					d.cacheMux.RLock()
					_, ok := d.cache[sepActual.Hash()]
					d.cacheMux.RUnlock()
					if !ok {
						var newCache CompCache
						d.cacheMux.Lock()
						d.cache[sepActual.Hash()] = &newCache
						d.cacheMux.Unlock()

					} else {
						for j := range comps {
							if d.checkNegative(sepActual, comps[j]) { //TODO: Add positive check and cutNodes
								//fmt.Println("Skipping a sep", sepActual)
								if addEdges {
									// i_add++
									continue addingEdges
								} else {
									continue OUTER
								}
							}
						}
					}

					// log.Printf("Comps of Sep: %v, len: %v\n", comps, len(comps))

					var subtrees []Node
					bag := Inter(sepActual.Vertices(), verticesExtended)

					// log.Println("sep", sep, "\nsepActual", sepActual, "\n B of SepActual", PrintVertices(sepActual.Vertices()), "\noldSep ", PrintVertices(oldSep),
					//  "\nvertices of C", PrintVertices(verticesCurrent), "\n\nunion o both", PrintVertices(verticesExtended), "\n bag: ", PrintVertices(bag))

					// for i := range sepActual.Vertices() {
					//  if Mem(verticesCurrent, sepActual.Vertices()[i]) && !Mem(bag, sepActual.Vertices()[i]) {

					//      fmt.Println("Another union: ", PrintVertices(append(oldSep, verticesCurrent...)))

					//      fmt.Println("Another intersect: ", PrintVertices(Inter(sepActual.Vertices(), verticesExtended)))

					//      fmt.Println("sep", sep, "\nsepActual", sepActual, "\n B of SepActual", PrintVertices(sepActual.Vertices()), "\noldSep ", PrintVertices(oldSep),
					//          "\nvertices of C", PrintVertices(verticesCurrent), "\n\nunion o both", PrintVertices(verticesExtended), "\n bag: ", PrintVertices(bag))

					//      log.Panicln("something is not right in the state of this program!")
					//  }
					// }

					lowFlag := false
					for i := range comps {

						decomp := d.findDecomp(K, comps[i], bag, compsSp[i], depth+1)
						if reflect.DeepEqual(decomp, Decomp{}) {
							//cache[sepActual.Hash()].Fail = append(cache[sepActual.Hash()].Fail, comps[i].Edges.Hash())
							d.addNegative(sepActual, comps[i])
							// log.Printf("detK REJECTING %v: couldn't decompose %v with SP %v \n", Graph{Edges: sepActual}, comps[i], compsSp[i])
							// log.Printf("\n\nCurrent oldSep: %v\n", PrintVertices(oldSep))
							// log.Printf("Current SubGraph: %v ( %v edges)\n", H, H.Edges.Len(), H.Edges.Hash())
							// log.Printf("Current Special Edges: %v\n\n", Sp)

							if d.SubEdge {
								if sepSub == nil {
									sepSub = GetSepSub(d.Graph.Edges, NewEdges(sepChanging), K)
								}

								nextBalsepFound := false

								for !nextBalsepFound {
									if sepSub.HasNext() {
										sepActual = sepSub.GetCurrent()
										sepActual = NewEdges(append(sepActual.Slice(), sepConst...))
										// log.Printf("Testing SSep: %v of %v , Special Edges %v \n", Graph{Edges: sepActual}, Graph{Edges: sepActualOrigin}, Sp)
										//log.Println("Sep const: ", sepConst, "sepChang ", sepChanging)
										// log.Println("SubSep: ")
										// for _, s := range sepSub.Edges {
										//  log.Println(s.Combination)
										// }
										if connectingSep(sepActual.Vertices(), conn, compVertices) {
											nextBalsepFound = true
										}
									} else {
										// log.Printf("No SubSep found for %v with Sp %v  \n", Graph{Edges: sepActualOrigin}, Sp)
										if addEdges {
											// i_add++
											continue addingEdges
										} else {
											continue OUTER
										}
									}
								}
								// log.Printf("Sub Sep chosen: %vof %v , %v \n", Graph{Edges: sepActual}, Graph{Edges: sepActualOrigin}, Sp)
								continue subEdges
							}

							if addEdges {
								// i_add++
								continue addingEdges
							} else {
								continue OUTER
							}
						}
						//cache[sepActual.Hash()].Succ = append(cache[sepActual.Hash()].Succ, comps[i].Edges.Hash())
						//d.addPositive(sepActual, comps[i])

						// log.Printf("Produced Decomp: %v\n", decomp)
						subtrees = append(subtrees, decomp.Root)
					}

					return Decomp{Graph: H, Root: Node{LowConnecting: lowFlag, Bag: bag, Cover: sepActual, Children: subtrees}}
				}
			}

		}

	}

	return Decomp{} // Reject if no separator could be found
}

func (d *LogKDecomp) FindHD(K int, currentGraph Graph, Sp []Special) Decomp {
	d.cache = make(map[uint32]*CompCache)

	// using log(N) + K as the depth here
	d.Depth = int(math.Ceil(math.Log2(float64(d.Graph.Edges.Len())) + float64(K)))

	fmt.Println("Looking for HD of depth", d.Depth)

	return d.findDecomp(K, currentGraph, []int{}, Sp, 0)
}

func (d LogKDecomp) FindDecomp(K int) Decomp {
	return d.FindHD(K, d.Graph, []Special{})
}

func (d LogKDecomp) Name() string {
	if d.SubEdge {
		return "LogK with local BIP"
	} else {
		return "LogK"
	}
}

func (d LogKDecomp) FindDecompGraph(G Graph, K int) Decomp {
	return d.FindHD(K, G, []Special{})
}

// func (d LogKDecomp) FindDecompUpdate(K int, currentGraph Graph, savedScenes map[uint32]SceneValue) Decomp {
// 	d.cache = make(map[uint32]*CompCache)
// 	return d.findDecompUpdate(K, currentGraph, []int{}, savedScenes)
// }

// func (d *LogKDecomp) findDecompUpdate(K int, H Graph, oldSep []int, savedScenes map[uint32]SceneValue) Decomp {

// 	//Check current scenario for saved scene
// 	// usingScene := false
// 	// usingSep := Edges{}
// 	// for i := range savedScenes {
// 	//  if Equiv(savedScenes[i].Sub.Vertices(), H.Vertices()) {
// 	//      usingScene = true
// 	//      usingBag = savedScenes[i].Sep
// 	//      log.Println("Using saved scene!")
// 	//      break
// 	//  }
// 	// }

// 	verticesCurrent := H.Vertices()
// 	verticesExtended := append(verticesCurrent, oldSep...)
// 	conn := Inter(oldSep, verticesCurrent)
// 	compVertices := Diff(verticesCurrent, oldSep)
// 	bound := FilterVertices(d.Graph.Edges, conn)

// 	// log.Printf("\n\nDU Current oldSep: %v, Conn: %v\n", PrintVertices(oldSep), PrintVertices(conn))
// 	// log.Printf("DU Current SubGraph: %v ( %v hash) \n", H, H.Edges.Hash())
// 	// log.Printf("DU Current SubGraph: %v ( %v edges) (hash: %v )\n", H, H.Edges.Len(), H.Edges.Hash())

// 	// log.Println("DU Hedges ", H)
// 	// log.Println("DU Comp Vertices: ", PrintVertices(compVertices))

// 	// Base case if H <= K
// 	if H.Edges.Len() == 0 {
// 		if d.Divide {
// 			out := baseCaseDetK(H, []Special{})
// 			out.Root.LowConnecting = true

// 			return out
// 		}
// 		return baseCaseDetK(H, []Special{})
// 	}

// 	gen := NewCover(K, conn, bound, H.Edges)

// OUTER:
// 	for gen.HasNext {

// 		val, ok := savedScenes[IntHash(verticesCurrent)]

// 		if !val.Perm { // delete one-time cached scene from map
// 			delete(savedScenes, IntHash(verticesCurrent))
// 		}
// 		if !Subset(conn, val.Sep.Vertices()) {
// 			ok = false // ignore this choice of separator if it breaks connectedness
// 		}

// 		var sep Edges
// 		addEdges := false

// 		if !ok {
// 			out := gen.NextSubset()

// 			if out == -1 {
// 				if gen.HasNext {
// 					log.Panicln(" -1 but hasNext not false!")
// 				}
// 				continue
// 			}
// 			sep = GetSubset(bound, gen.Subset)

// 			//check if sep "makes some progress" into separating H
// 			if len(Inter(sep.Vertices(), compVertices)) == 0 {
// 				addEdges = true
// 			}

// 			if !Subset(conn, sep.Vertices()) {
// 				log.Panicln("Cover messed up! 137")
// 			}

// 		} else {
// 			sep = val.Sep

// 		}

// 		log.Println("Next Cover ", sep)

// 		if !addEdges || K-sep.Len() > 0 {
// 			i_add := 0

// 		addingEdges:
// 			for !addEdges || i_add < H.Edges.Len() {
// 				var sepActual Edges

// 				if addEdges {
// 					sepActual = NewEdges(append(sep.Slice(), H.Edges.Slice()[i_add]))
// 				} else {
// 					sepActual = sep
// 				}

// 				// sepActualOrigin := sepActual
// 				var sepSub *SepSub
// 				var sepConst []Edge
// 				var sepChanging []Edge
// 				if d.SubEdge {
// 					for i, v := range gen.Subset {
// 						if gen.InComp[v] {
// 							sepChanging = append(sepChanging, sep.Slice()[i])
// 						} else {
// 							sepConst = append(sepConst, sep.Slice()[i])
// 						}
// 					}
// 					if addEdges {
// 						sepChanging = append(sepChanging, H.Edges.Slice()[i_add])
// 					}
// 				}

// 			subEdges:
// 				for true {

// 					// log.Println("Sep chosen ", sepActual)

// 					// if usingScene {
// 					//  sep := NewEdges([]Edge{Edge{Vertices: usingBag}})
// 					//  comps, _, _ = H.GetComponents(sep, []Special{})
// 					// } else {
// 					//
// 					// }
// 					comps, _, _ := H.GetComponents(sepActual, []Special{})

// 					//check chache for previous encounters
// 					d.cacheMux.RLock()
// 					_, ok := d.cache[sepActual.Hash()]
// 					d.cacheMux.RUnlock()
// 					if !ok {
// 						var newCache CompCache
// 						d.cacheMux.Lock()
// 						d.cache[sepActual.Hash()] = &newCache
// 						d.cacheMux.Unlock()

// 					} else {
// 						for j := range comps {
// 							if d.checkNegative(sepActual, comps[j]) { //TODO: Add positive check and cutNodes
// 								//fmt.Println("Skipping a sep", sepActual)
// 								if addEdges {
// 									i_add++
// 									continue addingEdges
// 								} else {
// 									continue OUTER
// 								}
// 							}
// 						}
// 					}

// 					// log.Printf("Comps of Sep: %v, len: %v\n", comps, len(comps))

// 					var subtrees []Node
// 					bag := Inter(sepActual.Vertices(), verticesExtended)

// 					lowFlag := false
// 					for i := range comps {
// 						if comps[i].Edges.Len() == 0 && d.Divide {
// 							lowFlag = true //since special Edge would connect to current sep, if accepting
// 							continue
// 						}
// 						decomp := d.findDecompUpdate(K, comps[i], bag, savedScenes)
// 						if reflect.DeepEqual(decomp, Decomp{}) {
// 							//cache[sepActual.Hash()].Fail = append(cache[sepActual.Hash()].Fail, comps[i].Edges.Hash())
// 							d.addNegative(sepActual, comps[i])
// 							// log.Printf("DU detK REJECTING %v: couldn't decompose %v  \n", Graph{Edges: sepActual}, comps[i])
// 							// log.Printf("\n\nDU Current oldSep: %v\n", PrintVertices(oldSep))
// 							// log.Printf("DU Current SubGraph: %v ( %v edges)\n", H, H.Edges.Len(), H.Edges.Hash())

// 							if d.SubEdge {
// 								if sepSub == nil {
// 									sepSub = GetSepSub(d.Graph.Edges, NewEdges(sepChanging), K)
// 								}

// 								nextBalsepFound := false

// 								for !nextBalsepFound {
// 									if sepSub.HasNext() {
// 										sepActual = sepSub.GetCurrent()
// 										sepActual = NewEdges(append(sepActual.Slice(), sepConst...))
// 										// log.Printf("Testing SSep: %v of %v , Special Edges %v \n", Graph{Edges: sepActual}, Graph{Edges: sepActualOrigin}, Sp)
// 										//log.Println("Sep const: ", sepConst, "sepChang ", sepChanging)
// 										// log.Println("SubSep: ")
// 										// for _, s := range sepSub.Edges {
// 										//  log.Println(s.Combination)
// 										// }
// 										if connectingSep(sepActual.Vertices(), conn, compVertices) {
// 											nextBalsepFound = true
// 										}
// 									} else {
// 										// log.Printf("No SubSep found for %v with Sp %v  \n", Graph{Edges: sepActualOrigin}, Sp)
// 										if addEdges {
// 											i_add++
// 											continue addingEdges
// 										} else {
// 											continue OUTER
// 										}
// 									}
// 								}
// 								// log.Printf("Sub Sep chosen: %vof %v , %v \n", Graph{Edges: sepActual}, Graph{Edges: sepActualOrigin}, Sp)
// 								continue subEdges
// 							}

// 							if addEdges {
// 								i_add++
// 								continue addingEdges
// 							} else {
// 								continue OUTER
// 							}
// 						}
// 						//cache[sepActual.Hash()].Succ = append(cache[sepActual.Hash()].Succ, comps[i].Edges.Hash())
// 						//d.addPositive(sepActual, comps[i])

// 						// log.Printf("Produced Decomp: %v\n", decomp)
// 						subtrees = append(subtrees, decomp.Root)
// 					}

// 					return Decomp{Graph: H, Root: Node{LowConnecting: lowFlag, Bag: bag, Cover: sepActual, Children: subtrees}}
// 				}
// 			}

// 		}

// 	}

// 	return Decomp{} // Reject if no separator could be found
// }