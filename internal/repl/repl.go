// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package repl

import (
        "bufio"
        "fmt"
        "lunex/internal/compiler"
        "lunex/internal/lexer"
        "lunex/internal/parser"
        "lunex/internal/runtime"
        "os"
        "strings"
)

const banner = `
  Lunex v%s
  Type .help for commands, .exit to quit
`

type REPL struct {
        compiler *compiler.Compiler
        interp   *runtime.Interpreter
        history  []string
}

func New(c *compiler.Compiler) *REPL {
        return &REPL{
                compiler: c,
                interp:   c.Interpreter(),
        }
}

func (r *REPL) Run() {
        fmt.Printf(banner, compiler.NTLVersion)

        scanner := bufio.NewScanner(os.Stdin)
        env := runtime.NewEnvironment(nil)
        _ = env

        for {
                fmt.Print("\x1b[36mntl>\x1b[0m ")
                if !scanner.Scan() {
                        break
                }
                line := scanner.Text()
                line = strings.TrimSpace(line)
                if line == "" {
                        continue
                }

                switch line {
                case ".exit", ".quit", "exit", "quit":
                        fmt.Println("Goodbye.")
                        return
                case ".help":
                        r.printHelp()
                        continue
                case ".version":
                        fmt.Printf("Lunex v%s\n", compiler.NTLVersion)
                        continue
                case ".clear":
                        fmt.Print("\033[H\033[2J")
                        continue
                }

                r.history = append(r.history, line)
                r.eval(line)
        }
}

func (r *REPL) eval(source string) {
        tokens, err := lexer.Tokenize(source, "<repl>")
        if err != nil {
                fmt.Fprintf(os.Stderr, "\x1b[31mlex error:\x1b[0m %v\n", err)
                return
        }

        tree, err := parser.Parse(tokens, "<repl>")
        if err != nil {
                fmt.Fprintf(os.Stderr, "\x1b[31msyntax error:\x1b[0m %v\n", err)
                return
        }

        result, execErr := r.interp.Exec(tree)
        if execErr != nil {
                fmt.Fprintf(os.Stderr, "\x1b[31merror:\x1b[0m %v\n", execErr)
                return
        }

        if result != nil && result != runtime.Undefined && result != runtime.Null {
                fmt.Println("\x1b[90m=>\x1b[0m " + result.Inspect())
        }
}

func (r *REPL) printHelp() {
        fmt.Print(`Commands:
    .help      — show this help
    .exit      — quit the REPL
    .version   — show version
    .clear     — clear screen

  Lunex Syntax:
    val x = 42              immutable binding
    var y = "hello"         mutable binding
    fn add(a, b) { ... }    function
    log x                   print value
    range(5)                [0,1,2,3,4]
    sleep(100)              sleep ms
`)
}
