# os

  Operating system interface: process execution, environment variables, paths, file system metadata, and system info.

  ## Import

  ```ntl
  use os
  ```

  ---

  ## Process Execution

  ### os.exec(command, options?)

  Run a command and wait for it to finish. Returns `{ stdout, stderr, code, ok }`.

  ```ntl
  use os
  use io

  val result = os.exec("git status")
  io.log(result.stdout)
  io.log("exit code:", result.code)

  val result2 = os.exec("npm install", {
    cwd:     "/my/project",
    timeout: 30000,
    env:     { NODE_ENV: "production" }
  })
  if result2.ok {
    io.success("Installed")
  } else {
    io.error(result2.stderr)
  }
  ```

  Options:
  - `cwd` — working directory
  - `env` — extra environment variables (merged with current env)
  - `timeout` — milliseconds before the process is killed

  ### os.spawn(command, options?)

  Start a process in the background without waiting. Returns `{ pid, wait(), kill() }`.

  ```ntl
  use os
  use io

  val proc = os.spawn("python3 server.py")
  io.log("started PID", proc.pid)

  val code = proc.wait()
  io.log("exited with", code)

  proc.kill()
  ```

  ---

  ## Environment Variables

  ### os.getenv(key)

  Get an environment variable.

  ```ntl
  val token = os.getenv("API_TOKEN")
  ```

  ### os.setenv(key, value)

  Set an environment variable for the current process.

  ### os.unsetenv(key)

  Remove an environment variable.

  ### os.environ()

  Get all environment variables as an object.

  ```ntl
  val env = os.environ()
  io.log(env["PATH"])
  ```

  ### os.expandEnv(str)

  Expand `$VAR` and `${VAR}` placeholders in a string.

  ```ntl
  val home = os.expandEnv("$HOME/projects")
  ```

  ---

  ## Process Info

  ### os.getpid()

  Current process ID.

  ### os.getppid()

  Parent process ID.

  ### os.args()

  Command-line arguments as an array.

  ```ntl
  val args = os.args()
  io.log("running:", args[0])
  ```

  ### os.exit(code?)

  Exit the process. Default exit code is `0`.

  ---

  ## System Info

  ### os.platform()

  Operating system name: `"linux"`, `"darwin"`, `"windows"`, etc.

  ### os.arch()

  CPU architecture: `"amd64"`, `"arm64"`, etc.

  ### os.cpus()

  Number of logical CPUs.

  ### os.hostname()

  Machine hostname.

  ```ntl
  use os
  use io

  io.log(os.platform(), os.arch(), os.cpus() + " CPUs")
  ```

  ---

  ## Paths

  ### os.join(...parts)

  Join path segments.

  ```ntl
  val p = os.join(os.homeDir, "projects", "myapp")
  ```

  ### os.dirname(path)

  Parent directory of a path.

  ### os.basename(path)

  Last segment of a path.

  ### os.extname(path)

  File extension including the dot.

  ### os.abs(path)

  Resolve a relative path to an absolute path.

  ### os.sep

  Path separator character (`/` or `\\`).

  ### os.homeDir

  Current user's home directory.

  ---

  ## File System Metadata

  ### os.stat(path)

  Get file/directory info. Returns `null` if path does not exist.

  ```ntl
  val info = os.stat("./myfile.txt")
  if info !== null {
    io.log(info.name, info.size, info.isFile, info.modTime)
  }
  ```

  Fields: `name`, `size`, `isDir`, `isFile`, `mode`, `modTime`

  ### os.exists(path)

  Returns `true` if the path exists.

  ### os.listDir(path?)

  List directory contents. Default is current directory.

  ```ntl
  val entries = os.listDir(".")
  each e in entries {
    io.log(e.name, e.isDir ? "[dir]" : e.size + " bytes")
  }
  ```

  ### os.glob(pattern)

  Find paths matching a glob pattern.

  ```ntl
  val files = os.glob("src/**/*.ntl")
  each f in files { io.log(f) }
  ```

  ### os.mkdir(path, recursive?)

  Create a directory. Pass `true` to create parent directories.

  ### os.remove(path, recursive?)

  Delete a file or directory. Pass `true` to remove recursively.

  ### os.rename(src, dst)

  Rename or move a file.

  ### os.tempDir()

  Path to the system temp directory.

  ### os.tempFile(prefix?)

  Create a temporary file and return its path.

  ---

  ## Examples

  ### Run a build script and log output

  ```ntl
  use os
  use io

  val r = os.exec("go build ./...", { cwd: os.getcwd() })
  if r.ok {
    io.success("Build complete")
  } else {
    io.error("Build failed")
    io.error(r.stderr)
    os.exit(1)
  }
  ```

  ### Walk directory and list all .ntl files

  ```ntl
  use os
  use io

  val files = os.glob("**/*.ntl")
  io.log("Found " + files.length + " NTL files:")
  each f in files { io.log("  " + f) }
  ```

  ### Spawn a background process

  ```ntl
  use os
  use io

  val server = os.spawn("node server.js", { cwd: "./backend" })
  io.log("Server PID:", server.pid)
  io.log("Press Ctrl+C to stop")
  server.wait()
  ```
  