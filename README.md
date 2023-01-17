# peppy

This is a serverless peer to peer matchmaking service based off of Slippi (having no idea of how Slippi matchmaking actually works).

The infrastructure is based on AppSync + DynamoDB global tables/streams and can be run in N regions which are all active at once.
Geolocation uses ipwho.is which is the only single point of failure, though the responses are cached to mitigate this. 

This currently only supports matchmaking 2 individuals-- there is no party-up system.
However, that shouldn't be too bad of an extension.

There are a handful of lambda functions that support this, all of which are written in Rust.
I knew 0 Rust going into this so I'm sure the code can use many improvements.

todo:
- vtl tests in rust
- dolphin build
- queue removal mechanisms
- invalidate subscriptions on region failure (read through queue, remove + invalidate each player)
- filter out recent matches
- queue heartbeat (?)
- process game results
- lock table healthcheck
- integration tests
- rust unit tests