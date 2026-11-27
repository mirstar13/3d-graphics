# Behavior Workshop

Two crates have the same behavior with different launch impulses. Build either
worker for your current machine and put its executable in
`assets/worker/behavior-worker`. On Windows, set `behavior_worker` in
`project.yaml` to the executable's actual `.exe` path.

## Build a C# worker

Run from this example directory with a .NET 8 SDK. This publishes a self-contained
Linux x64 worker. Use the matching .NET runtime identifier for your machine.

```sh
dotnet publish workers/csharp -c Release -r linux-x64 --self-contained true -o assets/worker
```

## Build a C++ worker

Use a C++17 compiler and the [nlohmann/json single header](https://github.com/nlohmann/json/releases/tag/v3.11.3).
Point the include path at the directory containing `json.hpp`.

```sh
mkdir -p assets/worker
c++ -std=c++17 -O2 -I /path/to/json-header workers/cpp/main.cpp -o assets/worker/behavior-worker
```

## Use it

- Run `go run .` in this example directory and press **E**. The blue crate gets
  a stronger impulse than the orange crate. Both workers request the `launch`
  sound once per press when the engine advertises it.
- Open the example from the engine with
  `go run . -project examples/behavior-workshop`. Edit **Behavior parameters
  (JSON)** on either crate and press **Ctrl+W** for Scripted Play. Press **E**
  to test the change, then **F8** to stop and discard the live simulation.
- **Ctrl+S** saves the scene. **F9** exports the Go game and the worker’s
  platform-specific files in the same archive.

## Worker protocol

One JSON request arrives on standard input before each fixed tick of a scene
with live attachments. The first request to a newly started worker has
`start: true`. The worker can keep state by actor ID until stdin closes.
Write one response per line, flush it, and put diagnostic logs on stderr.

```json
{"version":1,"tick":1,"delta":0.0166666667,"start":true,"actions":["launch"],"actors":[{"id":"left","behavior":"launch","parameters":{"height_impulse":3},"position":[-1.2,0.5,0],"velocity":[0,0,0]}],"contacts":[]}
```

Actions come from `gameplay_bindings`. Actor positions and velocities use world
coordinates. Contacts describe the preceding tick and contain `a`, `b`,
`phase` (`begin`, `stay` or `end`), and `sensor`.

```json
{"version":1,"tick":1,"impulses":[{"entity":"left","vector":[0,3,0]}]}
```

Echo the version and tick. A response can carry an empty impulse list or an
`error` string. Impulses target live dynamic entities using scene IDs. The
first protocol version supports impulses in newton-seconds. When the request
includes `sound_cues`, the reply may also include `"sounds":["launch"]` using
cue IDs from that list. The host accepts up to 32 sounds per tick and publishes
them after the tick succeeds. Workers can omit the field. Each exchange has a
one-second deadline and a 256 KiB message limit. A failed response stops
scripted advancement at that tick. Stopping or restarting closes the old worker.

Project opening only loads data. **Ctrl+W** and launching a built game can
execute the project worker with the current user's permissions.
