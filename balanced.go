package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"time"
)

func logActive(b bool) {
	log.SetFlags(0)
	if b {
		log.SetOutput(os.Stderr)
	} else {
		log.SetOutput(ioutil.Discard)
	}
}

func check(e error) {
	if e != nil {
		panic(e)
	}

}

//BalancedFactor is used by balsep algorithms to determine how strict the balancedness check should be (default 2)
var BalancedFactor int

var hinge bool

func main() {

	// m = make(map[int]string)

	//Command-Line Argument Parsing
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")
	logging := flag.Bool("log", false, "turn on extensive logs")
	computeSubedges := flag.Bool("sub", false, "Compute the subedges of the graph and print it out")
	width := flag.Int("width", 0, "a positive, non-zero integer indicating the width of the GHD to search for")
	graphPath := flag.String("graph", "", "the file path to a hypergraph \n\t(see http://hyperbench.dbai.tuwien.ac.at/downloads/manual.pdf, 1.3 for correct format)")
	choose := flag.Int("choice", 0, "only run one version\n\t1 ... Full Parallelism\n\t2 ... Search Parallelism\n\t3 ... Comp. Parallelism\n\t4 ... Sequential execution\n\t5 ... Local Full Parallelism\n\t6 ... Local Search Parallelism\n\t7 ... Local Comp. Parallelism\n\t8 ... Local Sequential execution.")
	balanceFactorFlag := flag.Int("balfactor", 2, "Determines the factor that balanced separator check uses")
	useHeuristic := flag.Int("heuristic", 0, "turn on to activate edge ordering\n\t1 ... Degree Ordering\n\t2 ... Max. Separator Ordering\n\t3 ... MCSO")
	gyö := flag.Bool("g", false, "perform a GYÖ reduct and show the resulting graph")
	typeC := flag.Bool("t", false, "perform a Type Collapse and show the resulting graph")
	hingeFlag := flag.Bool("hinge", false, "use isHinge Optimization")
	numCPUs := flag.Int("cpu", -1, "Set number of CPUs to use")

	akatovTest := flag.Bool("akatov", false, "compute balanced decomposition")
	detKTest := flag.Bool("det", false, "Test out DetKDecomp")

	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)

		defer pprof.StopCPUProfile()
	}

	hinge = *hingeFlag

	logActive(*logging)

	BalancedFactor = *balanceFactorFlag

	runtime.GOMAXPROCS(*numCPUs)

	if *graphPath == "" || *width <= 0 {
		fmt.Fprintf(os.Stderr, "Usage of %s: \n", os.Args[0])
		flag.VisitAll(func(f *flag.Flag) {
			if f.Name != "width" && f.Name != "graph" {
				return
			}
			s := fmt.Sprintf("%T", f.Value)
			fmt.Printf("  -%s \t%s\n", f.Name, s[6:len(s)-5])
			fmt.Println("\t" + f.Usage)
		})

		fmt.Println("\nOptional Arguments: ")
		flag.VisitAll(func(f *flag.Flag) {
			if f.Name == "width" || f.Name == "graph" {
				return
			}
			s := fmt.Sprintf("%T", f.Value)
			fmt.Printf("  -%s \t%s\n", f.Name, s[6:len(s)-5])
			fmt.Println("\t" + f.Usage)
		})

		return
	}

	dat, err := ioutil.ReadFile(*graphPath)
	check(err)

	parsedGraph := getGraph(string(dat))
	var reducedGraph Graph

	if *typeC {
		count := 0
		fmt.Println("\n\n", *graphPath)
		fmt.Println("Graph after Type Collapse:")
		reducedGraph, _, count = parsedGraph.typeCollapse()
		for _, e := range reducedGraph.edges {
			fmt.Printf("%v %v\n", e, Edge{vertices: e.vertices})
		}
		fmt.Println("Removed ", count, " vertex/vertices")
		parsedGraph = reducedGraph
	}

	if *gyö {
		fmt.Println("Graph after GYÖ:")
		var ops []GYÖReduct
		if *typeC {
			reducedGraph, ops = reducedGraph.GYÖReduct()
		} else {
			reducedGraph, ops = parsedGraph.GYÖReduct()
		}

		fmt.Println(reducedGraph)

		fmt.Println("Reductions:")
		fmt.Println(ops)
		parsedGraph = reducedGraph

	}
	//fmt.Println("Graph ", parsedGraph)
	//fmt.Println("Min Distance", getMinDistances(parsedGraph))
	//return

	// count := 0
	// for _, e := range parsedGraph.edges {
	// 	isOW, _ := e.OWcheck(parsedGraph.edges)
	// 	if isOW {
	// 		count++
	// 	}
	// }
	// fmt.Println("No of OW edges: ", count)

	// collapsedGraph, _ := parsedGraph.typeCollapse()

	// fmt.Println("No of vertices collapsable: ", len(parsedGraph.Vertices())-len(collapsedGraph.Vertices()))

	if *computeSubedges {
		parsedGraph = parsedGraph.computeSubEdges(*width)

		fmt.Println("Graph with subedges \n", parsedGraph)
	}

	start := time.Now()
	switch *useHeuristic {
	case 1:
		fmt.Print("Using degree ordering")
		parsedGraph.edges = getDegreeOrder(parsedGraph.edges)
	case 2:
		fmt.Print("Using max seperator ordering")
		parsedGraph.edges = getMaxSepOrder(parsedGraph.edges)
	case 3:
		fmt.Print("Using MSC ordering")
		parsedGraph.edges = getMSCOrder(parsedGraph.edges)
	}
	d := time.Now().Sub(start)

	if *useHeuristic > 0 {
		fmt.Println(" as a heuristic")
		msec := d.Seconds() * float64(time.Second/time.Millisecond)
		fmt.Printf("Time for heuristic: %.5f ms\n", msec)
		log.Printf("Ordering: %v", parsedGraph.String())

	}

	if *akatovTest {
		var decomp Decomp
		start := time.Now()

		switch *choose {
		case 1:
			decomp = balKDecomp{graph: parsedGraph}.findBDFullParallel(*width)
		// case 2:
		// 	decomp = global.findGHDParallelSearch(*width)
		// case 3:
		// 	decomp = global.findGHDParallelComp(*width)
		case 4:
			decomp = balKDecomp{graph: parsedGraph}.findBD(*width)
		default:
			panic("Not a valid choice")
		}

		d := time.Now().Sub(start)
		msec := d.Seconds() * float64(time.Second/time.Millisecond)

		fmt.Println("Result \n", decomp)
		fmt.Println("Time", msec, " ms")
		fmt.Println("Width: ", decomp.checkWidth())
		fmt.Println("GHD-Width: ", decomp.blowup().checkWidth())
		fmt.Println("Correct: ", decomp.correct(parsedGraph))
		return
	}

	if *detKTest {
		var decomp Decomp
		start := time.Now()

		var Sp []Special
		m[encode] = "test"
		m[encode+1] = "test2"
		Sp = []Special{Special{vertices: []int{16, 18}, edges: []Edge{Edge{name: encode, vertices: []int{16, 18}}}}, Special{vertices: []int{15, 17, 19}, edges: []Edge{Edge{name: encode + 1, vertices: []int{15, 17, 19}}}}}
		encode = encode + 2

		det := detKDecomp{graph: parsedGraph}
		switch *choose {
		case 1:
			decomp = det.findHDParallelFull(*width, Sp)
		case 2:
			decomp = det.findHDParallelSearch(*width, Sp)
		case 3:
			decomp = det.findHDParallelDecomp(*width, Sp)
		case 4:
			decomp = det.findHD(*width, Sp)
		default:
			panic("Not a valid choice")
		}

		d := time.Now().Sub(start)
		msec := d.Seconds() * float64(time.Second/time.Millisecond)

		fmt.Println("Result \n", decomp)
		fmt.Println("Time", msec, " ms")
		fmt.Println("Width: ", decomp.checkWidth())
		fmt.Println("Correct: ", decomp.correct(parsedGraph))
		return
	}

	global := balsepGlobal{graph: parsedGraph}
	local := balsepLocal{graph: parsedGraph}
	if *choose != 0 {
		var decomp Decomp
		start := time.Now()
		switch *choose {
		case 1:
			decomp = global.findGHDParallelFull(*width)
		case 2:
			decomp = global.findGHDParallelSearch(*width)
		case 3:
			decomp = global.findGHDParallelComp(*width)
		case 4:
			decomp = global.findGHD(*width)
		case 5:
			decomp = local.findGHDParallelFull(*width)
		case 6:
			decomp = local.findGHDParallelSearch(*width)
		case 7:
			decomp = local.findGHDParallelComp(*width)
		case 8:
			decomp = local.findGHD(*width)
		default:
			panic("Not a valid choice")
		}
		d := time.Now().Sub(start)
		msec := d.Seconds() * float64(time.Second/time.Millisecond)

		fmt.Println("Result \n", decomp)
		fmt.Println("Time", msec, " ms")
		fmt.Println("Width: ", *width)
		fmt.Println("Correct: ", decomp.correct(parsedGraph))
		return
	}

	var output string

	// f, err := os.OpenFile("result.csv", os.O_APPEND|os.O_WRONLY, 0666)
	// if os.IsNotExist(err) {
	// 	f, err = os.Create("result.csv")
	// 	check(err)
	// 	f.WriteString("graph;edges;vertices;width;time parallel (ms) F;decomposed;time parallel S (ms);decomposed;time parallel C (ms);decomposed; time sequential (ms);decomposed\n")
	// }
	// defer f.Close()

	//fmt.Println("Width: ", *width)
	//fmt.Println("graphPath: ", *graphPath)

	output = output + *graphPath + ";"

	//f.WriteString(*graphPath + ";")
	//f.WriteString(fmt.Sprintf("%v;", *width))

	//fmt.Printf("parsedGraph %+v\n", parsedGraph)

	output = output + fmt.Sprintf("%v;", len(parsedGraph.edges))
	output = output + fmt.Sprintf("%v;", len(Vertices(parsedGraph.edges)))
	output = output + fmt.Sprintf("%v;", *width)

	// Parallel Execution FULL
	start = time.Now()
	decomp := global.findGHDParallelFull(*width)

	//fmt.Printf("Decomp of parsedGraph:\n%v\n", decomp.root)

	//fmt.Println("Elapsed time for parallel:", time.Now().Sub(start))
	//fmt.Println("Correct decomposition:", decomp.correct())
	d = time.Now().Sub(start)
	msec := d.Seconds() * float64(time.Second/time.Millisecond)
	output = output + fmt.Sprintf("%.5f;", msec)

	// Parallel Execution Search
	start = time.Now()
	decomp = global.findGHDParallelSearch(*width)

	//fmt.Printf("Decomp of parsedGraph:\n%v\n", decomp.root)

	//fmt.Println("Elapsed time for parallel:", time.Now().Sub(start))
	//fmt.Println("Correct decomposition:", decomp.correct())
	d = time.Now().Sub(start)
	msec = d.Seconds() * float64(time.Second/time.Millisecond)
	output = output + fmt.Sprintf("%.5f;", msec)

	// Parallel Execution Comp
	start = time.Now()
	decomp = global.findGHDParallelComp(*width)

	//fmt.Printf("Decomp of parsedGraph:\n%v\n", decomp.root)

	//fmt.Println("Elapsed time for parallel:", time.Now().Sub(start))
	//fmt.Println("Correct decomposition:", decomp.correct())
	d = time.Now().Sub(start)
	msec = d.Seconds() * float64(time.Second/time.Millisecond)
	output = output + fmt.Sprintf("%.5f;", msec)
	// Sequential Execution
	start = time.Now()
	decomp = global.findGHD(*width)

	//fmt.Printf("Decomp of parsedGraph: %v\n", decomp.root)
	d = time.Now().Sub(start)
	msec = d.Seconds() * float64(time.Second/time.Millisecond)
	output = output + fmt.Sprintf("%.5f;", msec)
	output = output + fmt.Sprintf("%v\n", decomp.correct(parsedGraph))
	//fmt.Println("Elapsed time for sequential:", time.Now().Sub(start))
	//fmt.Println("Correct decomposition:", decomp.correct())

	fmt.Print(output)

}
