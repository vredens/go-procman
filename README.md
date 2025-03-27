# Go Process (Routine) Manager

This is not a replacement for proper handling control channels or contexts and cancelation. The goal of this package is to provide a controller for handling multiple long-running go routines. Best way to describe it is a lightweight supervisor.

## Examples

Take a look at the [examples file](example_test.go)

## Future improvements

> WIP

- Extensive improvements on how things start and stop and general internals of things.
- Provide helpers to create several common worker types.
- More usage examples.
