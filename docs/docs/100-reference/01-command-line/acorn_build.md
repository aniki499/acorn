---
title: "acorn build"
---
## acorn build

Build an app from a Acornfile file

### Synopsis

Build all dependent container and app images from your Acornfile file

```
acorn build [flags] DIRECTORY
```

### Examples

```

# Build from Acornfile file in the local directory
acorn build .
```

### Options

```
  -f, --file string        Name of the build file (default "DIRECTORY/Acornfile")
  -h, --help               help for build
  -p, --platform strings   Target platforms (form os/arch[/variant][:osversion] example linux/amd64)
      --profile strings    Profile to assign default values
      --push               Push image after build
  -t, --tag strings        Apply a tag to the final build
```

### Options inherited from parent commands

```
  -A, --all-projects        Use all known projects
      --context string      Context to use in the resolved kubeconfig file
      --debug               Enable debug logging
      --debug-level int     Debug log level (valid 0-9) (default 7)
      --kubeconfig string   Explicitly use kubeconfig file, overriding current project
      --namespace string    Namespace to work in resolved connection (default "acorn")
  -j, --project string      Project to work in
```

### SEE ALSO

* [acorn](acorn.md)	 - 

