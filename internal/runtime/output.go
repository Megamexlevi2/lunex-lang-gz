// Lunex lang
  // Created by David Dev · GitHub: https://github.com/Megamexlevi2
  // (c) David Dev 2026. License.
  //
  // output.go — global output writer for the Lunex runtime.
  // In normal builds this points to os.Stdout.
  // In WebAssembly builds, web/main.go swaps it per-call to a bytes.Buffer
  // so lunexRun() can return captured output to JavaScript.
  package runtime

  import (
      "fmt"
      "io"
      "os"
  )

  // OutputWriter is the writer used by all Lunex io.log / io.warn / io.err functions.
  // Defaults to os.Stdout. WebAssembly builds replace it temporarily per call.
  var OutputWriter io.Writer = os.Stdout

  // out returns the current writer, falling back to os.Stdout if nil.
  func out() io.Writer {
      if OutputWriter != nil {
          return OutputWriter
      }
      return os.Stdout
  }

  // PrintLn writes a line to OutputWriter.
  func PrintLn(args ...interface{}) {
      fmt.Fprintln(out(), args...)
  }

  // Print writes to OutputWriter without a trailing newline.
  func Print(args ...interface{}) {
      fmt.Fprint(out(), args...)
  }

  // PrintF writes a formatted string to OutputWriter.
  func PrintF(format string, args ...interface{}) {
      fmt.Fprintf(out(), format, args...)
  }

  // BrowserUnavailableError is returned by WebAssembly stubs for features
  // that require OS-level capabilities not available in the browser.
  type BrowserUnavailableError struct {
      Feature string
  }

  func (e *BrowserUnavailableError) Error() string {
      return e.Feature + " is not available in browser (WebAssembly). " +
          "This feature requires OS-level access such as TCP sockets, subprocesses, or filesystem writes."
  }
  