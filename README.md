# synctags
The synctags project - syncing tags across multiple tools.  Allows you to use one tool as your Source of Truth (SOT) for tagging, then create a YAML file that will be synced across all tools that support tagging.

Use Case: with today's proliferation of security tools, making sure teams across the org are talking about the same resources.  Most modern security tools use Tagging.  However, every vendor implements tagging differently.

synctags uses a CLI format similar to other tools:
command tool action [flags]
```
Get all the tags from Qualys and write to a YAML file
    $ synctags qualys get
Create the SOT from YAML tag list
    $ synctags qualys create
Sync the Tags from the YML Master with Qualys
    $ synctags qualys sync
```
For CrowdStrike the vendor name would change but is otherwise the same set of actions (get, create, sync). A NinjaOne client exists but is not wired in yet. Wanted to add ThreatAware as well, but doesn't appear they have added an API to their tags :(

## Layout

```
cmd/synctags/          CLI entry point (cobra)
internal/tags/         vendor-neutral Tag type and YAML format
internal/qualys/       Qualys QPS Tagging API client
internal/crowdstrike/  CrowdStrike Falcon API client
internal/ninjaone/     NinjaOne API client (not wired in yet)
internal/version/      build stamp, set by `task build`
```

## Build

Uses [Task](https://taskfile.dev) (`brew install go-task`).

```
task build     # bin/synctags, stamped with tag and commit
task install   # build, then copy to ~/bin
task check     # fmt, vet, tests, golangci-lint, staticcheck, gosec, govulncheck
task ship      # everything in check, then git push
```

__THIS IS VERY MUCH A WIP__. Use at your own risk (constructive criticism is welcome however).
