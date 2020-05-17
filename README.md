# DConfig - Dynamic Config

Dynamic config provider based on Raft.

## Why?

At $JOB we have multiple flavors of config, some static, some dynamic, and config can be start-time args, runtime-args, 
feature flags, kill switches, knobs for various features... the list goes on. That system requires a number of moving
pieces and can be confusing to debug plus grok all the required dependencies to make this happen, let alone manage these
as infra grows more complex over time. This project aims to make an OSS tool to solve this repeatable problem.

## What does this support?

- Config fallback interface (Static, or your existing source to migrate)
- Typical startup or runtime config
- Feature Flags
- Kill Switches

