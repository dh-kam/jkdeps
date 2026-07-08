package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dh-kam/jkdeps/internal/cliutil"
	"github.com/dh-kam/jkdeps/internal/flagutil"
	"github.com/dh-kam/jkdeps/internal/mixedgraph"
)

func runGraph(args []string) int {
	fs := cliutil.NewFlagSet("graph", args)

	parseFlags := addMixedParseCommandFlags(fs)
	groupBy := fs.String("group-by", string(mixedgraph.GroupByPackage), "Graph grouping: package|dir")
	minEdgeCount := fs.Int("min-edge-count", 0, "Keep only edges with count >= this value")
	printJSON := fs.Bool("json", false, "Print graph JSON to stdout")
	outPath := fs.String("out", "jkdeps-mixed-graph", "Output path base (or explicit .html/.json path)")
	var inventoryPaths stringListFlag
	fs.Var(&inventoryPaths, "inventory", "Path to external inventory JSON (repeatable or comma-separated)")
	var includePrefixes stringListFlag
	var excludePrefixes stringListFlag
	fs.Var(&includePrefixes, "include-prefix", "Keep nodes whose names start with this prefix (repeatable or comma-separated)")
	fs.Var(&excludePrefixes, "exclude-prefix", "Drop nodes whose names start with this prefix (repeatable or comma-separated)")

	if ok, code := cliutil.ParseFlagSet(fs, args); !ok {
		return code
	}

	if err := startGraphProfiling(); err != nil {
		fmt.Fprintf(os.Stderr, "profile init failed: %v\n", err)
		return 1
	}
	defer stopGraphProfiling()

	inventoryPaths, externalIndex, err := loadExternalIndexFlags(inventoryPaths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load inventory: %v\n", err)
		return 1
	}
	includePrefixes = stringListFlag(flagutil.UniqueStrings(includePrefixes))
	excludePrefixes = stringListFlag(flagutil.UniqueStrings(excludePrefixes))

	group, err := parseGroupByFlag(*groupBy)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	result, err := parseFlags.parseRepository()
	if err != nil {
		fmt.Fprintf(os.Stderr, "graph failed: %v\n", err)
		return 1
	}

	filter := mixedgraph.GraphFilter{
		MinEdgeCount:  *minEdgeCount,
		IncludePrefix: []string(includePrefixes),
		ExcludePrefix: []string(excludePrefixes),
	}
	graph := mixedgraph.BuildFilteredGraph(result, externalIndex, group, filter)
	result.Files = nil
	exitCode := failOnErrorExitCode(parseFlags.failOnErrorEnabled(), result.FailedFiles)
	if *printJSON {
		if err := cliutil.WritePrettyJSON(os.Stdout, graph); err != nil {
			fmt.Fprintf(os.Stderr, "encode graph: %v\n", err)
			return 1
		}
		if *outPath == "" {
			return exitCode
		}
	}

	htmlPath, jsonPath := cliutil.GraphOutputPaths(*outPath, "jkdeps-mixed-graph")
	if !*printJSON {
		if err := writeGraphArtifacts(htmlPath, jsonPath, graph); err != nil {
			fmt.Fprintf(os.Stderr, "write graph artifacts: %v\n", err)
			return 1
		}
	} else {
		if err := cliutil.WritePrettyJSONFile(jsonPath, graph); err != nil {
			fmt.Fprintf(os.Stderr, "write graph artifacts: %v\n", err)
			return 1
		}
	}

	if !*printJSON {
		writeMixedRepositorySummary(os.Stdout, result, "", false)
		writeSlowParseFilesSummary(os.Stdout, result, parseFlags.topParseFilesCount())
		cliutil.WriteSummaryLine(os.Stdout, "Group By", "%s", graph.GroupBy)
		cliutil.WriteSummaryLine(os.Stdout, "Graph", "nodes=%d edges=%d", len(graph.Nodes), len(graph.Edges))
		if *minEdgeCount > 0 || len(includePrefixes) > 0 || len(excludePrefixes) > 0 {
			cliutil.WriteSummaryLine(os.Stdout, "Filter", "min-edge=%d include=%d exclude=%d", *minEdgeCount, len(includePrefixes), len(excludePrefixes))
		}
		if len(inventoryPaths) > 0 {
			cliutil.WriteSummaryLine(os.Stdout, "Inventory", "files=%d packages=%d symbols=%d", len(inventoryPaths), len(externalIndex.Packages), len(externalIndex.Symbols))
		}
		cliutil.WriteSummaryLine(os.Stdout, "Output", "%s", jsonPath)
		cliutil.WriteSummaryLine(os.Stdout, "Viewer", "%s", htmlPath)
	}
	return exitCode
}

func writeGraphArtifacts(htmlPath, jsonPath string, graph mixedgraph.Graph) error {
	if err := os.MkdirAll(filepath.Dir(htmlPath), 0o755); err != nil {
		return err
	}
	if err := cliutil.WritePrettyJSONFile(jsonPath, graph); err != nil {
		return err
	}

	html := buildGraphHTML(filepath.Base(jsonPath))
	if err := os.WriteFile(htmlPath, []byte(html), 0o644); err != nil {
		return err
	}
	return nil
}

func buildGraphHTML(dataFile string) string {
	quotedDataFile := strconv.Quote(dataFile)
	return strings.ReplaceAll(graphHTMLTemplate, "__DATA_FILE__", quotedDataFile)
}

const graphHTMLTemplate = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>jkdeps Mixed Graph Viewer</title>
  <script src="https://unpkg.com/vis-network/standalone/umd/vis-network.min.js"></script>
  <style>
    :root {
      --bg: #f2f5f8;
      --card: #ffffff;
      --text: #152335;
      --subtle: #586779;
      --internal: #0f7a53;
      --external: #235fa2;
      --unknown: #b04b3d;
    }
    html, body {
      margin: 0;
      padding: 0;
      height: 100%;
      background: radial-gradient(circle at 10% 10%, #ffffff 0%, var(--bg) 45%);
      color: var(--text);
      font-family: "Avenir Next", "Segoe UI", Arial, sans-serif;
    }
    .layout {
      height: 100%;
      display: grid;
      grid-template-rows: auto 1fr;
    }
    .header {
      padding: 16px 20px;
      border-bottom: 1px solid #dbe3ea;
      background: linear-gradient(95deg, #ffffff 0%, #f1f5f9 100%);
    }
    .title {
      margin: 0;
      font-size: 20px;
      line-height: 1.2;
      letter-spacing: 0.02em;
    }
    .meta {
      margin-top: 6px;
      color: var(--subtle);
      font-size: 13px;
    }
    #graph {
      width: 100%;
      height: 100%;
      background: var(--card);
    }
    .legend {
      margin-top: 8px;
      font-size: 12px;
      color: var(--subtle);
      display: flex;
      gap: 14px;
      flex-wrap: wrap;
    }
    .chip::before {
      content: "";
      display: inline-block;
      width: 9px;
      height: 9px;
      border-radius: 50%;
      margin-right: 6px;
      vertical-align: middle;
    }
    .chip.internal::before { background: var(--internal); }
    .chip.external::before { background: var(--external); }
    .chip.unknown::before { background: var(--unknown); }
  </style>
</head>
<body>
  <div class="layout">
    <div class="header">
      <h1 class="title">jkdeps Mixed Dependency Graph</h1>
      <div id="meta" class="meta">Loading...</div>
      <div class="legend">
        <span class="chip internal">internal</span>
        <span class="chip external">external</span>
        <span class="chip unknown">unknown</span>
      </div>
    </div>
    <div id="graph"></div>
  </div>
  <script>
    const dataFile = __DATA_FILE__;

    function colorByKind(kind) {
      if (kind === "internal") return "#0f7a53";
      if (kind === "external") return "#235fa2";
      return "#b04b3d";
    }

    fetch(dataFile)
      .then((res) => {
        if (!res.ok) throw new Error("failed to load graph JSON");
        return res.json();
      })
      .then((graph) => {
        document.getElementById("meta").textContent =
          "Root: " + graph.root + " | GroupBy: " + graph.group_by + " | Nodes: " + graph.nodes.length + " | Edges: " + graph.edges.length;

        const nodes = graph.nodes.map((n) => ({
          id: n.id,
          label: n.name,
          color: colorByKind(n.kind),
          title: n.name + "\nkind=" + n.kind + "\nin=" + n.in_degree + " out=" + n.out_degree,
          shape: "dot",
          size: 8 + Math.min(26, n.in_degree + n.out_degree)
        }));

        const edges = graph.edges.map((e) => ({
          from: e.from_id,
          to: e.to_id,
          value: e.count,
          width: 1 + Math.log2(e.count + 1),
          title: "count=" + e.count,
          arrows: "to",
          color: { color: "#8aa3b7", opacity: 0.65 }
        }));

        const container = document.getElementById("graph");
        const network = new vis.Network(
          container,
          { nodes: new vis.DataSet(nodes), edges: new vis.DataSet(edges) },
          {
            interaction: { hover: true, navigationButtons: true, zoomView: true },
            physics: {
              stabilization: false,
              barnesHut: {
                gravitationalConstant: -2200,
                springLength: 180,
                springConstant: 0.04
              }
            },
            nodes: {
              borderWidth: 0,
              font: { color: "#152335", size: 13, face: "Avenir Next" }
            },
            edges: {
              smooth: { enabled: true, type: "dynamic" },
              arrows: { to: { enabled: true, scaleFactor: 0.6 } }
            }
          }
        );

        window.network = network;
      })
      .catch((err) => {
        document.getElementById("meta").textContent = "Failed to load graph: " + err.message;
      });
  </script>
</body>
</html>
`
