package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	methodStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	pathStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	hostStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	ipStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	indexStyle  = lipgloss.NewStyle().Faint(true)
	labelStyle  = lipgloss.NewStyle().Faint(true).Width(12)
	headerKey   = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	headerVal   = lipgloss.NewStyle().Faint(true)
	bodyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	divider     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	totalStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
)

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Execute the traffic query and display HTTP requests",
	Long:  `Executes the configured traffic query (parquet, gor, etc.) and displays the resulting HTTP requests without running WAF analysis.`,
	Run:   runQuery,
}

func init() {
	queryCmd.Flags().StringP("gor-file", "g", "", "Path to traffic file (.gor or .custom)")
	queryCmd.Flags().Bool("more", false, "Show full request details (headers, body)")
	queryCmd.Flags().IntP("limit", "n", 0, "Limit number of requests displayed (0 = no limit)")
}

func runQuery(cmd *cobra.Command, args []string) {
	gorFile, _ := cmd.Flags().GetString("gor-file")
	verbose, _ := cmd.Flags().GetBool("more")
	limit, _ := cmd.Flags().GetInt("limit")

	// Resolve from config if flag not set
	if gorFile == "" && appConfig != nil {
		gorFile = appConfig.Traffic.File
	}

	reader, cleanup := openTrafficReader(gorFile)
	defer cleanup()
	defer reader.Close()

	requests, errChan := reader.ReadRequests(1000)

	count := 0
	headerPrinted := false
	for req := range requests {
		count++
		if limit > 0 && count > limit {
			break
		}

		if verbose {
			printRequestVerbose(count, req)
		} else {
			if !headerPrinted {
				printSummaryHeader()
				headerPrinted = true
			}
			printRequestSummary(count, req)
		}
	}

	if err := <-errChan; err != nil {
		slog.Warn("traffic read error", "error", err)
	}

	fmt.Printf("\n%s\n", totalStyle.Render(fmt.Sprintf("Total: %d requests", count)))
}

var columnHeader = lipgloss.NewStyle().Bold(true).Underline(true).Faint(true)

func printSummaryHeader() {
	fmt.Printf("%s %s %s %s %s\n",
		columnHeader.Render(fmt.Sprintf("%-5s", "#")),
		columnHeader.Render(fmt.Sprintf("%-6s", "METHOD")),
		columnHeader.Render(fmt.Sprintf("%-50s", "PATH")),
		columnHeader.Render(fmt.Sprintf("%-30s", "HOST")),
		columnHeader.Render("CLIENT IP"),
	)
}

func printRequestSummary(index int, req *http.Request) {
	host := req.Host
	if host == "" {
		host = "-"
	}
	clientIP := req.RemoteAddr
	if clientIP == "" {
		clientIP = "-"
	}

	fmt.Printf("%s %s %s %s %s\n",
		indexStyle.Render(fmt.Sprintf("#%-4d", index)),
		methodStyle.Render(fmt.Sprintf("%-6s", req.Method)),
		pathStyle.Render(fmt.Sprintf("%-50s", truncate(req.URL.Path, 50))),
		hostStyle.Render(fmt.Sprintf("%-30s", truncate(host, 30))),
		ipStyle.Render(clientIP),
	)
}

func printRequestVerbose(index int, req *http.Request) {
	fmt.Println(divider.Render(fmt.Sprintf("─── Request #%d ───────────────────────────────────────", index)))

	fmt.Printf("%s%s\n", labelStyle.Render("Method"), methodStyle.Render(req.Method))
	fmt.Printf("%s%s\n", labelStyle.Render("Path"), pathStyle.Render(req.URL.Path))
	fmt.Printf("%s%s\n", labelStyle.Render("Host"), hostStyle.Render(req.Host))
	fmt.Printf("%s%s\n", labelStyle.Render("Proto"), req.Proto)
	fmt.Printf("%s%s\n", labelStyle.Render("Client IP"), ipStyle.Render(req.RemoteAddr))

	if len(req.Header) > 0 {
		fmt.Printf("%s\n", labelStyle.Render("Headers"))
		keys := make([]string, 0, len(req.Header))
		for k := range req.Header {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("  %s %s\n",
				headerKey.Render(k+":"),
				headerVal.Render(strings.Join(req.Header[k], ", ")),
			)
		}
	}

	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err == nil && len(body) > 0 {
			fmt.Printf("%s\n", labelStyle.Render(fmt.Sprintf("Body (%d B)", len(body))))
			fmt.Printf("  %s\n", bodyStyle.Render(string(body)))
		}
	}

	fmt.Println()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
