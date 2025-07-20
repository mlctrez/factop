# factop

Factorio operator.

### Rationale for using a soft mod v.s. an actual mod

* [grok answer](https://x.com/i/grok/share/9eEbNfDbw9s6PPf7qMjJNIW0l)

### Why this and not [factorio-server-manager](https://github.com/OpenFactorioServerManager/factorio-server-manager)?

* This project manages the factorio headless server in a similar manner but adds:
    * An api for applying a softmod to the currently running save, which greatly speeds up development.
        * The stop, save file changes, and start steps are handled when a new softmod is applied.
    * The factop service also exposes a [web server](https://factorio.mlctrez.com). Right now it does nothing. Future
      enhancements could be:
        * Tracking player progress, statistics, etc
        * Administrative functions like resetting the map, etc.
    * A [nats](https://docs.nats.io/nats-concepts/what-is-nats) server is embedded in the factop service.
        * The Factorio stdin, stdout, and stderr are exposed as nats subjects.
    * A rcon connection is managed by the factop service and exposed via a [nrpc](github.com/nats-rpc/nrpc) api.
    * A mage build target for executing lua code via this rcon connection.

### Softmod work

The code under softmod and examples are experiments and by no means are they complete. They are test cases
and experiments. While some parts rely on the softmod code, most of the interactions require an external process.
It is much easier to iterate to see what works without having to reload a mod each time.