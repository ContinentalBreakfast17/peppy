# peppy

This is a serverless peer to peer matchmaking service based off of Slippi (having no idea of how Slippi matchmaking actually works).

The infrastructure is based on AppSync + DynamoDB global tables/streams and can be run in N regions which are all active at once.
Geolocation uses ipwho.is which is the only single point of failure, though the responses are cached to mitigate this. 

This currently only supports matchmaking 2 individuals-- there is no party-up system.
However, that shouldn't be too bad of an extension.

There are a handful of lambda functions that support this, all of which are written in Rust.
I knew 0 Rust going into this so I'm sure the code can use many improvements.

todo:
- route53 healthcheck (alarm in place)
- queue heartbeat
- filter out recent matches
- process game results
- rust unit tests
- integration tests

jmook skilled player but that is not normally, This very very insane....They need to check him controller and disc.....Maybe he not cheating but maybe he using the game deficit ...and this cant seem on game screen..He needs to check-up