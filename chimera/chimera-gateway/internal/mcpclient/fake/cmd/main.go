// Minimal stdio MCP server for manual checks and subprocess tests.
package main

import (
	"flag"
	"os"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/mcpclient/fake"
)

func main() {
	collision := flag.Bool("collision-tool", false, "expose read_file MCP tool for collision tests")
	flag.Parse()
	_ = fake.Serve(os.Stdin, os.Stdout, fake.Config{CollisionTool: *collision})
}
