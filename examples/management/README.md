# A generated notes Space

This example is a small surface to read and run while assessing the API's feel:

1. [layout.go](layout.go) declares ordinary Conduit storage. The notebook README
   is seeded content: creation renders it, and later manifest edits leave it alone.
2. [management.go](management.go) declares the named topology and typed creation
   input. Its callbacks supply values, render context, and a domain rule.
3. [example_test.go](example_test.go) uses `NewNotes → Bootstrap → Create →
   Manifest.Update → List`. Each operation delegates to the shared runtime.
4. [management_gen.go](management_gen.go) is the generated, browsable wrapper.
   It contains typed bindings and delegation, without application orchestration.

From the repository root:

```sh
go generate ./examples/management
go test ./examples/management -run Example -v
```

The example uses a temporary directory. Neither generation nor `NewNotes` creates
managed files. `Bootstrap` and `Create` perform explicit initialization.

The root is a Space because the application chooses a notes directory without
project recognition. A Scope represents each notebook inside that Space.
`notebook.Readme` exposes only Read; `notebook.Manifest` exposes ordinary document
operations. The layout remains available through `Layout()` for application logic.
